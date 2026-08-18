package admin

import (
	"sync"

	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"
)

// Host is satisfied by *framework.App. It is defined here so this package
// does not import framework (framework.New calls Mount).
type Host interface {
	API() huma.API
	DB() *gorm.DB
	Config() config.Config
}

type Registry struct {
	mu     sync.RWMutex
	models []*registered
	bySlug map[string]*registered
}

func newRegistry() *Registry {
	return &Registry{bySlug: map[string]*registered{}}
}

func (r *Registry) add(m *registered) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.bySlug[m.meta.Slug]; ok {
		return errDuplicateSlug(m.meta.Slug)
	}
	r.models = append(r.models, m)
	r.bySlug[m.meta.Slug] = m
	return nil
}

func (r *Registry) get(slug string) (*registered, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.bySlug[slug]
	return m, ok
}

func (r *Registry) all() []*registered {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*registered, len(r.models))
	copy(out, r.models)
	return out
}

type registered struct {
	meta        ModelMeta
	actions     Actions
	pkColumn    string
	pkType      FieldType
	newInstance func() any
	newSlice    func() any
	forEach     func(slicePtr any, fn func(any))
	fields      []resolvedField
	fieldByName map[string]*resolvedField
	implicit    map[string]string // name -> column (created_at / updated_at)
}

type resolvedField struct {
	Field
	column string
	get    func(inst any) any
	set    func(inst any, raw any) error
}

func (m *registered) field(name string) (*resolvedField, bool) {
	f, ok := m.fieldByName[name]
	return f, ok
}

func (m *registered) columnFor(name string) (string, bool) {
	if f, ok := m.fieldByName[name]; ok {
		return f.column, true
	}
	if col, ok := m.implicit[name]; ok {
		return col, true
	}
	return "", false
}

func (m *registered) toRow(inst any) map[string]any {
	out := make(map[string]any, len(m.fields))
	for i := range m.fields {
		f := &m.fields[i]
		out[f.Name] = f.get(inst)
	}
	return out
}

var registries sync.Map // huma.API -> *Registry

func registryFor(api huma.API) (*Registry, bool) {
	if api == nil {
		return nil, false
	}
	v, ok := registries.Load(api)
	if !ok {
		return nil, false
	}
	reg, ok := v.(*Registry)
	return reg, ok
}

func storeRegistry(api huma.API, reg *Registry) bool {
	_, loaded := registries.LoadOrStore(api, reg)
	return !loaded
}
