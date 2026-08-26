package project

import (
	"context"
	"strconv"
	"strings"

	"github.com/gombit-dev/gombit/benchmarks/apps/shared"
	"github.com/gombit-dev/gombit/contract"
	"github.com/gombit-dev/gombit/database"
	"gorm.io/gorm"
)

// Handler serves the canonical /api/projects routes (benchmarks/docs/schema.md)
// over Gombit's normal public APIs: Huma-typed operations, GORM, and the
// framework's own contract.ErrorEnvelope / database.MapLoadError /
// database.MapPersistError for error mapping — used as-is, not patched or
// worked around, because the whole point of this app is measuring Gombit's
// actual behavior. Response success shapes come from benchmarks/apps/shared
// so the wire format matches benchmarks/apps/gin-gorm's exactly (issue #141
// "same JSON shape"); error mapping deliberately does not, since Gombit's
// own D10 error mapping is itself one of the things being measured.
type Handler struct {
	DB *gorm.DB
}

type listProjectsInput struct {
	Page  int `query:"page" doc:"1-based page"`
	Limit int `query:"limit" doc:"Page size"`
}

type listProjectsOutput struct {
	Body shared.DataMeta[[]shared.ProjectData, shared.PageMeta]
}

type getProjectInput struct {
	ID string `path:"id" doc:"Project identifier"`
}

type getProjectOutput struct {
	Body shared.Data[shared.ProjectData]
}

type createProjectBody struct {
	// minimum:"1" rejects owner_id:0 at validation, matching
	// benchmarks/apps/gin-gorm's binding:"required" on the same field
	// (Gin's "required" treats the zero value of a non-pointer uint as
	// absent). Without this, 0 — a present, well-typed field — reached the
	// database and hit the same foreign-key violation POST #141's
	// intentionally-pinned nonexistent-owner case does, conflating two
	// different inputs behind one 500. See
	// benchmarks/apps/gombit/README.md for the case that's still meant to
	// 500 (a nonexistent but well-formed id) versus this one (a value
	// Huma can reject before it ever reaches GORM).
	OwnerID     uint   `json:"owner_id" minimum:"1" doc:"Owner user id"`
	Name        string `json:"name" minLength:"1" maxLength:"255" doc:"Project name"`
	Description string `json:"description,omitempty" doc:"Project description"`
}

type createProjectInput struct {
	Body createProjectBody
}

type createProjectOutput struct {
	Body shared.Data[shared.ProjectData]
}

type updateProjectBody struct {
	Name        *string `json:"name,omitempty" minLength:"1" maxLength:"255" doc:"Project name"`
	Description *string `json:"description,omitempty" doc:"Project description"`
}

type updateProjectInput struct {
	ID   string `path:"id" doc:"Project identifier"`
	Body updateProjectBody
}

type updateProjectOutput struct {
	Body shared.Data[shared.ProjectData]
}

type deleteProjectInput struct {
	ID string `path:"id" doc:"Project identifier"`
}

type deleteProjectOutput struct {
	Body shared.Data[map[string]bool]
}

// list handles GET /api/projects?page=&limit=: a single page of projects,
// deterministically ordered (id DESC), owner preloaded in one batched
// query — see benchmarks/apps/gin-gorm/handlers.go's list for the exact
// query-count contract (benchmarks/docs/schema.md "Canonical CRUD API")
// this mirrors.
func (h *Handler) list(ctx context.Context, input *listProjectsInput) (*listProjectsOutput, error) {
	page, limit := clampPage(input.Page, input.Limit)

	var total int64
	if err := h.DB.WithContext(ctx).Model(&Project{}).Count(&total).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("count projects"))
	}

	var rows []Project
	err := h.DB.WithContext(ctx).
		Preload("Owner").
		Order("id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("list projects"))
	}

	items := make([]shared.ProjectData, 0, len(rows))
	for _, row := range rows {
		items = append(items, toProjectData(row))
	}

	return &listProjectsOutput{
		Body: shared.DataMeta[[]shared.ProjectData, shared.PageMeta]{
			Data: items,
			Meta: &shared.PageMeta{Page: page, Limit: limit, Total: total},
		},
	}, nil
}

// get handles GET /api/projects/{id}.
func (h *Handler) get(ctx context.Context, input *getProjectInput) (*getProjectOutput, error) {
	id, err := strconv.ParseUint(input.ID, 10, 64)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.NotFound("project not found"))
	}

	var row Project
	if err := h.DB.WithContext(ctx).Preload("Owner").First(&row, uint(id)).Error; err != nil {
		return nil, database.MapLoadError(ctx, err, "project not found", "load project")
	}

	return &getProjectOutput{Body: shared.Data[shared.ProjectData]{Data: toProjectData(row)}}, nil
}

// create handles POST /api/projects. blankNameError is checked explicitly
// even though Body.Name already carries minLength:"1": Huma's minLength
// rejects an absent/zero-length string, not a whitespace-only one, the same
// gap benchmarks/apps/gin-gorm's binding tags have — checked symmetrically
// here from the start instead of waiting to reproduce that asymmetry.
func (h *Handler) create(ctx context.Context, input *createProjectInput) (*createProjectOutput, error) {
	if err := blankNameError(ctx, input.Body.Name); err != nil {
		return nil, err
	}

	row := Project{
		OwnerID:     input.Body.OwnerID,
		Name:        input.Body.Name,
		Description: input.Body.Description,
	}
	if err := h.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, database.MapPersistError(ctx, err, "project already exists", "create project")
	}
	if err := h.DB.WithContext(ctx).Preload("Owner").First(&row, row.ID).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("reload created project"))
	}

	return &createProjectOutput{Body: shared.Data[shared.ProjectData]{Data: toProjectData(row)}}, nil
}

// update handles PATCH /api/projects/{id}: load, apply the provided fields,
// save the full row — the same load-mutate-save shape the runtime admin's
// generic update handler uses (admin/resources.go), and the same shape
// benchmarks/apps/gin-gorm's update uses.
func (h *Handler) update(ctx context.Context, input *updateProjectInput) (*updateProjectOutput, error) {
	id, err := strconv.ParseUint(input.ID, 10, 64)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.NotFound("project not found"))
	}

	if input.Body.Name != nil {
		if err := blankNameError(ctx, *input.Body.Name); err != nil {
			return nil, err
		}
	}

	var row Project
	if err := h.DB.WithContext(ctx).First(&row, uint(id)).Error; err != nil {
		return nil, database.MapLoadError(ctx, err, "project not found", "load project")
	}

	if input.Body.Name != nil {
		row.Name = *input.Body.Name
	}
	if input.Body.Description != nil {
		row.Description = *input.Body.Description
	}
	if err := h.DB.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, database.MapPersistError(ctx, err, "project already exists", "update project")
	}
	if err := h.DB.WithContext(ctx).Preload("Owner").First(&row, row.ID).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("reload updated project"))
	}

	return &updateProjectOutput{Body: shared.Data[shared.ProjectData]{Data: toProjectData(row)}}, nil
}

// delete handles DELETE /api/projects/{id}.
func (h *Handler) delete(ctx context.Context, input *deleteProjectInput) (*deleteProjectOutput, error) {
	id, err := strconv.ParseUint(input.ID, 10, 64)
	if err != nil {
		return nil, contract.WithContext(ctx, contract.NotFound("project not found"))
	}

	var row Project
	if err := h.DB.WithContext(ctx).First(&row, uint(id)).Error; err != nil {
		return nil, database.MapLoadError(ctx, err, "project not found", "load project")
	}
	if err := h.DB.WithContext(ctx).Delete(&row).Error; err != nil {
		return nil, contract.WithContext(ctx, contract.Internal("delete project"))
	}

	return &deleteProjectOutput{Body: shared.Data[map[string]bool]{Data: map[string]bool{"ok": true}}}, nil
}

func blankNameError(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return contract.WithContext(ctx, contract.Validation("name must not be blank", nil))
	}
	return nil
}

func toProjectData(row Project) shared.ProjectData {
	return shared.ProjectData{
		ID:          row.ID,
		OwnerID:     row.OwnerID,
		OwnerName:   row.Owner.Name,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func clampPage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
