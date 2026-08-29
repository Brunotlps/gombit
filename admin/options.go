package admin

// Closed field-type set for v1 (ADR-013). ADMIN-1 may add members with a
// docs bump; do not invent a parallel type system.
const (
	TypeString   FieldType = "string"
	TypeText     FieldType = "text"
	TypeInteger  FieldType = "integer"
	TypeFloat    FieldType = "float"
	TypeDecimal  FieldType = "decimal"
	TypeBoolean  FieldType = "boolean"
	TypeDateTime FieldType = "datetime"
	TypeDate     FieldType = "date"
	TypeUUID     FieldType = "uuid"
	TypeJSON     FieldType = "json"
	TypeRelation FieldType = "relation"
)

// Relation kinds for v1.
const (
	RelBelongsTo  = "belongs_to"
	RelHasMany    = "has_many"
	RelManyToMany = "many_to_many"
)

// Implicit timestamp names allowed in List and Ordering even when omitted
// from Fields. They are GORM's default created_at / updated_at columns.
const (
	ImplicitCreatedAt = "created_at"
	ImplicitUpdatedAt = "updated_at"
)

// FieldType is a closed admin field type string.
type FieldType string

// Options is the source of truth for one registered admin model.
type Options struct {
	// Slug is the URL key (products). Required, lowercase, unique per app.
	Slug string
	// Singular and Plural are UI labels. Empty values are derived at Register
	// from the Go type name and slug.
	Singular string
	Plural   string
	// PK is the JSON/field name of the primary key. Empty means derive the
	// GORM primary key at Register and store it.
	PK string
	// Fields is the concrete field list handlers read. Empty means derive a
	// default from the struct once, inside Register (see FieldsFrom).
	Fields []Field
	// List is the list-view column order.
	List []string
	// Search is the list of field names the search query param applies to.
	Search []string
	// Filter is the list of field names that may appear as list query keys.
	Filter []string
	// Ordering is the list of field names the ordering query param may use.
	// created_at and updated_at may appear here even if omitted from Fields.
	Ordering []string
	// Actions enables list / detail / create / update / delete. The zero
	// value (all false) defaults to all enabled.
	Actions Actions
	// Permissions are authorization keys enforced by the admin handlers.
	// Empty values default to admin.{slug}.{action}.
	Permissions Permissions
}

// Field describes one registered admin field.
//
// Name is the JSON object key used in meta and in data-plane row payloads.
// For v1, Name is also the GORM/SQL column unless Column is set (when the
// Go exported name or GORM column differs from the JSON key).
type Field struct {
	Name     string    `json:"name"`
	Type     FieldType `json:"type"`
	Required bool      `json:"required"`
	ReadOnly bool      `json:"readonly"`
	Related  *Relation `json:"related,omitempty"`
	// Column is the GORM/SQL column name. Empty means Name == JSON key ==
	// column (the v1 default). Not emitted in meta.
	Column string `json:"-"`
}

// Relation describes a belongs_to, has_many, or many_to_many field. belongs_to
// is stored as the foreign key on create/update. has_many is meta-only: the
// data plane does not nest related collections. many_to_many reads the related
// primary keys and, on write, syncs the join table to the submitted id list
// (#223).
type Relation struct {
	Slug       string `json:"slug"`
	Kind       string `json:"kind"`
	LabelField string `json:"label_field"`
}

// Actions names which data-plane operations are enabled for a model.
type Actions struct {
	List   bool `json:"list"`
	Detail bool `json:"detail"`
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

// Permissions holds the keys enforced for each admin operation.
type Permissions struct {
	View   string `json:"view"`
	Create string `json:"create"`
	Update string `json:"update"`
	Delete string `json:"delete"`
}

func (a Actions) zero() bool {
	return !a.List && !a.Detail && !a.Create && !a.Update && !a.Delete
}

func defaultActions() Actions {
	return Actions{List: true, Detail: true, Create: true, Update: true, Delete: true}
}

func validFieldType(t FieldType) bool {
	switch t {
	case TypeString, TypeText, TypeInteger, TypeFloat, TypeDecimal,
		TypeBoolean, TypeDateTime, TypeDate, TypeUUID, TypeJSON, TypeRelation:
		return true
	default:
		return false
	}
}

func implicitTimestamp(name string) bool {
	return name == ImplicitCreatedAt || name == ImplicitUpdatedAt
}
