package database

import (
	"context"
	"fmt"
	"reflect"

	"gorm.io/gorm"
)

// Validator is implemented by models that enforce domain invariants — rules
// that cross fields, rows, or models, such as "ownership windows must not
// overlap" or "confirm a rental only if every child resource is available".
//
// The framework runs Validate inside the write transaction on BOTH the API
// (generated handler Create/Save) and the admin data-plane write paths, via a
// GORM callback registered in Open. An invariant therefore has exactly one home
// and neither write surface can bypass it. tx is the transaction the write runs
// in, so Validate can query committed-but-in-flight state consistently.
//
// Return a *ValidationError (see NewValidationError) to surface field-level D10
// 422 responses; any other error aborts the write and maps to 500.
type Validator interface {
	Validate(ctx context.Context, tx *gorm.DB) error
}

// ValidationError is a domain validation failure raised by a model's Validate
// hook. MapPersistError maps it to a D10 422 validation_error, preserving the
// per-field messages.
type ValidationError struct {
	Message string
	Fields  map[string][]string
}

// NewValidationError builds a domain ValidationError. Pass nil fields for a
// message-only error.
func NewValidationError(message string, fields map[string][]string) *ValidationError {
	return &ValidationError{Message: message, Fields: fields}
}

// Error satisfies the error interface.
func (e *ValidationError) Error() string {
	if e == nil || e.Message == "" {
		return "validation failed"
	}
	return e.Message
}

// registerValidationCallback wires Validate into the GORM create and update
// callback chains so it runs before the SQL on every write path.
func registerValidationCallback(db *gorm.DB) error {
	create := db.Callback().Create().Before("gorm:create")
	if err := create.Register("gombit:validate", runValidate); err != nil {
		return fmt.Errorf("database: register create validate callback: %w", err)
	}
	update := db.Callback().Update().Before("gorm:update")
	if err := update.Register("gombit:validate", runValidate); err != nil {
		return fmt.Errorf("database: register update validate callback: %w", err)
	}
	return nil
}

// runValidate invokes Validate on the model value(s) being written, aborting the
// operation (and rolling back any surrounding transaction) on the first error.
func runValidate(db *gorm.DB) {
	if db.Error != nil || db.Statement == nil {
		return
	}
	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}
	rv := db.Statement.ReflectValue
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if err := validateValue(ctx, db, rv.Index(i)); err != nil {
				_ = db.AddError(err)
				return
			}
		}
	case reflect.Struct:
		if err := validateValue(ctx, db, rv); err != nil {
			_ = db.AddError(err)
		}
	}
}

func validateValue(ctx context.Context, tx *gorm.DB, rv reflect.Value) error {
	// Prefer the addressable (pointer-receiver) form so a Validate defined on
	// *T is found; fall back to the value form.
	if rv.CanAddr() {
		if v, ok := rv.Addr().Interface().(Validator); ok {
			return v.Validate(ctx, tx)
		}
	}
	if rv.CanInterface() {
		if v, ok := rv.Interface().(Validator); ok {
			return v.Validate(ctx, tx)
		}
	}
	return nil
}
