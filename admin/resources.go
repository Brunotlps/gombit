package admin

import (
	"context"
	"errors"
	"strings"

	"github.com/gombit-dev/gombit/contract"
	"github.com/gombit-dev/gombit/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type listInput struct {
	Slug     string `path:"slug" doc:"Registered model slug"`
	Page     int    `query:"page" doc:"1-based page"`
	PerPage  int    `query:"per_page" doc:"Page size"`
	Search   string `query:"search" doc:"Search term applied to Options.Search"`
	Ordering string `query:"ordering" doc:"Field from Options.Ordering; prefix with - for DESC"`
}

type writeInput struct {
	Slug string         `path:"slug" doc:"Registered model slug"`
	Body map[string]any `doc:"Writable field values keyed by registered field names"`
}

type itemInput struct {
	Slug string `path:"slug" doc:"Registered model slug"`
	ID   string `path:"id" doc:"Primary key value"`
}

type patchInput struct {
	Slug string         `path:"slug" doc:"Registered model slug"`
	ID   string         `path:"id" doc:"Primary key value"`
	Body map[string]any `doc:"Writable field values keyed by registered field names"`
}

// row is a registered model's field values keyed by field name. It exists
// so the admin data plane's generic responses have a named type: an
// anonymous map[string]any as a generic type parameter has no
// reflect.Type.Name(), so Huma's DefaultSchemaNamer falls back to Go's
// unnamed-type string ("map[string]interface {}") and that literal space
// and braces survive into the OpenAPI component name — producing e.g.
// "DataMetaListMapStringInterface {}PageMeta", which fails OpenAPI 3.1
// validation (component names must match ^[a-zA-Z0-9._-]+$).
type row map[string]any

type rowOutput struct {
	Body contract.Data[row]
}

type listOutput struct {
	Body contract.DataMeta[[]row, contract.PageMeta]
}

type deleteResult struct {
	OK bool `json:"ok" example:"true"`
}

type deleteOutput struct {
	Body contract.Data[deleteResult]
}

func (h *handlers) listResources(ctx context.Context, input *listInput) (*listOutput, error) {
	m, err := h.modelForAction(
		ctx,
		input.Slug,
		func(a Actions) bool { return a.List },
		func(p Permissions) string { return p.View },
	)
	if err != nil {
		return nil, err
	}
	db, err := h.db()
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("admin database is not attached"))
	}

	page, perPage := contract.ClampPage(input.Page, input.PerPage)

	q := db.WithContext(ctx).Model(m.newInstance())
	q, err = applySearch(q, m, input.Search)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Validation("The request contains invalid fields.", map[string][]string{
			"search": {err.Error()},
		}))
	}
	q, err = applyFilters(ctx, q, m, queryValues(ctx))
	if err != nil {
		return nil, err
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("count resources"))
	}

	q, err = applyOrdering(q, m, input.Ordering)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Validation("The request contains invalid fields.", map[string][]string{
			"ordering": {err.Error()},
		}))
	}

	slice := m.newSlice()
	if err := q.Offset(contract.PageOffset(page, perPage)).Limit(perPage).Find(slice).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("list resources"))
	}

	rows := make([]row, 0)
	m.forEach(slice, func(item any) {
		rows = append(rows, m.toRow(item))
	})
	return &listOutput{
		Body: contract.DataMeta[[]row, contract.PageMeta]{
			Data: rows,
			Meta: &contract.PageMeta{Page: page, PerPage: perPage, Total: total},
		},
	}, nil
}

func (h *handlers) createResource(ctx context.Context, input *writeInput) (*rowOutput, error) {
	m, err := h.modelForAction(
		ctx,
		input.Slug,
		func(a Actions) bool { return a.Create },
		func(p Permissions) string { return p.Create },
	)
	if err != nil {
		return nil, err
	}
	db, err := h.db()
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("admin database is not attached"))
	}
	inst := m.newInstance()
	if err := applyWrite(ctx, m, inst, input.Body, true); err != nil {
		return nil, err
	}
	if err := db.WithContext(ctx).Create(inst).Error; err != nil {
		return nil, database.MapPersistError(ctx, err, "resource already exists", "persist resource")
	}
	return &rowOutput{Body: contract.Data[row]{Data: m.toRow(inst)}}, nil
}

func (h *handlers) getResource(ctx context.Context, input *itemInput) (*rowOutput, error) {
	m, err := h.modelForAction(
		ctx,
		input.Slug,
		func(a Actions) bool { return a.Detail },
		func(p Permissions) string { return p.View },
	)
	if err != nil {
		return nil, err
	}
	inst, err := h.loadByID(ctx, m, input.ID)
	if err != nil {
		return nil, err
	}
	return &rowOutput{Body: contract.Data[row]{Data: m.toRow(inst)}}, nil
}

func (h *handlers) updateResource(ctx context.Context, input *patchInput) (*rowOutput, error) {
	m, err := h.modelForAction(
		ctx,
		input.Slug,
		func(a Actions) bool { return a.Update },
		func(p Permissions) string { return p.Update },
	)
	if err != nil {
		return nil, err
	}
	inst, err := h.loadByID(ctx, m, input.ID)
	if err != nil {
		return nil, err
	}
	if err := applyWrite(ctx, m, inst, input.Body, false); err != nil {
		return nil, err
	}
	db, err := h.db()
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("admin database is not attached"))
	}
	if err := db.WithContext(ctx).Save(inst).Error; err != nil {
		return nil, database.MapPersistError(ctx, err, "resource already exists", "persist resource")
	}
	return &rowOutput{Body: contract.Data[row]{Data: m.toRow(inst)}}, nil
}

func (h *handlers) deleteResource(ctx context.Context, input *itemInput) (*deleteOutput, error) {
	m, err := h.modelForAction(
		ctx,
		input.Slug,
		func(a Actions) bool { return a.Delete },
		func(p Permissions) string { return p.Delete },
	)
	if err != nil {
		return nil, err
	}
	inst, err := h.loadByID(ctx, m, input.ID)
	if err != nil {
		return nil, err
	}
	db, err := h.db()
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("admin database is not attached"))
	}
	if err := db.WithContext(ctx).Delete(inst).Error; err != nil {
		return nil, database.MapDeleteError(ctx, err, "resource is still referenced by other records", "delete resource")
	}
	return &deleteOutput{Body: contract.Data[deleteResult]{Data: deleteResult{OK: true}}}, nil
}

func (h *handlers) modelForAction(
	ctx context.Context,
	slug string,
	enabled func(Actions) bool,
	permission func(Permissions) string,
) (*registered, error) {
	m, ok := h.reg.get(slug)
	if !ok {
		return nil, contract.WithContext(ctx, contract.NotFound("unknown model"))
	}
	if !enabled(m.actions) {
		return nil, contract.WithContext(ctx, contract.Authorization("action disabled"))
	}
	if err := h.requirePermission(ctx, permission(m.meta.Permissions)); err != nil {
		return nil, err
	}
	return m, nil
}

func (h *handlers) loadByID(ctx context.Context, m *registered, id string) (any, error) {
	db, err := h.db()
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("admin database is not attached"))
	}
	pk, err := coercePathID(id, m.pkType)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.NotFound("unknown resource"))
	}
	inst := m.newInstance()
	err = db.WithContext(ctx).Where(clause.Eq{Column: clause.Column{Name: m.pkColumn}, Value: pk}).First(inst).Error
	if err != nil {
		return nil, database.MapLoadError(ctx, err, "unknown resource", "load resource")
	}
	return inst, nil
}

func (h *handlers) db() (*gorm.DB, error) {
	if h.host == nil || h.host.DB() == nil {
		return nil, errors.New("admin: nil database")
	}
	return h.host.DB(), nil
}

func applyWrite(ctx context.Context, m *registered, inst any, body map[string]any, creating bool) error {
	if body == nil {
		body = map[string]any{}
	}
	fields := map[string][]string{}
	seen := map[string]struct{}{}
	for name, raw := range body {
		f, ok := m.field(name)
		if !ok {
			fields[name] = []string{"unknown field"}
			continue
		}
		if f.ReadOnly || (f.Type == TypeRelation && f.Related != nil && f.Related.Kind == RelHasMany) {
			fields[name] = []string{"field is read-only"}
			continue
		}
		if raw == nil && f.Required {
			fields[name] = []string{"is required"}
			continue
		}
		if err := f.set(inst, raw); err != nil {
			fields[name] = []string{err.Error()}
			continue
		}
		seen[name] = struct{}{}
	}
	if creating {
		for i := range m.fields {
			f := &m.fields[i]
			if !f.Required || f.ReadOnly {
				continue
			}
			if _, ok := seen[f.Name]; !ok {
				fields[f.Name] = []string{"is required"}
			}
		}
	}
	if len(fields) > 0 {
		return contract.WithContext(ctx, contract.Validation("The request contains invalid fields.", fields))
	}
	return nil
}

func applySearch(q *gorm.DB, m *registered, term string) (*gorm.DB, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return q, nil
	}
	if len(m.meta.Search) == 0 {
		return q, nil
	}
	pattern := "%" + escapeLike(term) + "%"
	ors := make([]clause.Expression, 0, len(m.meta.Search))
	for _, name := range m.meta.Search {
		col, ok := m.columnFor(name)
		if !ok {
			continue
		}
		ors = append(ors, clause.Expr{
			SQL:  quoteIdent(q, col) + " LIKE ? ESCAPE ?",
			Vars: []any{pattern, `\`},
		})
	}
	if len(ors) == 0 {
		return q, nil
	}
	return q.Where(clause.Or(ors...)), nil
}

func applyFilters(ctx context.Context, q *gorm.DB, m *registered, values interface{ Get(string) string }) (*gorm.DB, error) {
	if values == nil {
		return q, nil
	}
	for _, name := range m.meta.Filter {
		raw := strings.TrimSpace(values.Get(name))
		if raw == "" {
			continue
		}
		f, ok := m.field(name)
		if !ok || f.column == "" {
			continue
		}
		val, err := coerceFilter(raw, f.Type)
		if err != nil {
			return nil, contract.WithContext(ctx, contract.Validation("The request contains invalid fields.", map[string][]string{
				name: {err.Error()},
			}))
		}
		q = q.Where(clause.Eq{Column: clause.Column{Name: f.column}, Value: val})
	}
	return q, nil
}

func applyOrdering(q *gorm.DB, m *registered, ordering string) (*gorm.DB, error) {
	ordering = strings.TrimSpace(ordering)
	if ordering == "" {
		return q, nil
	}
	desc := false
	name := ordering
	if strings.HasPrefix(name, "-") {
		desc = true
		name = strings.TrimPrefix(name, "-")
	}
	allowed := false
	for _, n := range m.meta.Ordering {
		if n == name {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, errors.New("ordering is not allowed for this field")
	}
	col, ok := m.columnFor(name)
	if !ok {
		return nil, errors.New("ordering is not allowed for this field")
	}
	return q.Order(clause.OrderByColumn{Column: clause.Column{Name: col}, Desc: desc}), nil
}

func quoteIdent(db *gorm.DB, name string) string {
	var b strings.Builder
	db.QuoteTo(&b, name)
	return b.String()
}
