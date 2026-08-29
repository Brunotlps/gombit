// Package types holds framework value types that are shared by generated
// models, handler DTOs, and the admin data plane. They exist so a single Go
// type flows through the model, the Huma contract (OpenAPI/TS client), and GORM
// persistence without the DTO drifting from the model (see #222 / #218).
package types

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/shopspring/decimal"
)

// Decimal is a fixed-point decimal for money and other exact numeric values.
//
// It wraps shopspring/decimal.Decimal, so it inherits exact arithmetic, JSON
// (a quoted string — no float rounding), text (un)marshaling, and the database
// sql.Scanner / driver.Valuer used by GORM. On top of that it advertises an
// OpenAPI schema (a string with format "decimal") via huma.SchemaProvider, so
// the generated OpenAPI document and TypeScript client see a string instead of
// an opaque object. Use the gorm tag `type:decimal(p,s)` on the model field to
// pin precision/scale for the migration (gombit make resource emits
// decimal(19,4) by default).
type Decimal struct {
	decimal.Decimal
}

// NewDecimal wraps a shopspring decimal.
func NewDecimal(d decimal.Decimal) Decimal {
	return Decimal{Decimal: d}
}

// NewDecimalFromString parses a decimal string (e.g. "19.99").
func NewDecimalFromString(s string) (Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Decimal{}, err
	}
	return Decimal{Decimal: d}, nil
}

// Schema implements huma.SchemaProvider so the contract represents a Decimal as
// a JSON string (matching its MarshalJSON), not a reflected struct.
func (Decimal) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{
		Type:    huma.TypeString,
		Format:  "decimal",
		Pattern: `^-?[0-9]+(\.[0-9]+)?$`,
		Examples: []any{
			"19.99",
		},
	}
}

// GormDataType tells GORM the default column family when a model field omits an
// explicit `type:` tag. Generated models pin exact precision with
// `gorm:"type:decimal(19,4)"`, which takes precedence over this default.
func (Decimal) GormDataType() string {
	return "decimal"
}
