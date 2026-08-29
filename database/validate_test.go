package database

import (
	"context"
	"errors"
	"testing"

	"github.com/gombit-dev/gombit/contract"
	"gorm.io/gorm"
)

type validatedRow struct {
	ID     uint `gorm:"primaryKey"`
	Name   string
	Amount int
}

// Validate enforces a domain invariant: Amount must be non-negative and Name
// must be present. It runs on every create/update via the Open callback.
func (r validatedRow) Validate(_ context.Context, _ *gorm.DB) error {
	fields := map[string][]string{}
	if r.Name == "" {
		fields["name"] = []string{"is required"}
	}
	if r.Amount < 0 {
		fields["amount"] = []string{"must be non-negative"}
	}
	if len(fields) > 0 {
		return NewValidationError("The request contains invalid fields.", fields)
	}
	return nil
}

func TestValidatorHookRunsOnCreate(t *testing.T) {
	db := openSQLite(t)
	if err := db.AutoMigrate(&validatedRow{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	// Valid row succeeds.
	if err := db.Create(&validatedRow{Name: "ok", Amount: 5}).Error; err != nil {
		t.Fatalf("valid create error = %v, want nil", err)
	}

	// Invalid row is rejected by the hook, and the error is a *ValidationError.
	err := db.Create(&validatedRow{Name: "", Amount: -1}).Error
	if err == nil {
		t.Fatal("invalid create error = nil, want validation error")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %T (%v), want *ValidationError", err, err)
	}
	if len(ve.Fields["name"]) == 0 || len(ve.Fields["amount"]) == 0 {
		t.Fatalf("fields = %#v, want name+amount", ve.Fields)
	}

	// The row must not have persisted.
	var count int64
	db.Model(&validatedRow{}).Count(&count)
	if count != 1 {
		t.Fatalf("row count = %d, want 1 (invalid row must not persist)", count)
	}
}

func TestValidatorHookRunsOnUpdate(t *testing.T) {
	db := openSQLite(t)
	if err := db.AutoMigrate(&validatedRow{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	row := validatedRow{Name: "ok", Amount: 5}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	row.Amount = -3
	err := db.Save(&row).Error
	if err == nil {
		t.Fatal("invalid save error = nil, want validation error")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %T (%v), want *ValidationError", err, err)
	}
}

func TestMapPersistErrorMapsValidationErrorTo422(t *testing.T) {
	err := MapPersistError(
		context.Background(),
		NewValidationError("bad", map[string][]string{"name": {"is required"}}),
		"conflict", "internal",
	)
	var env *contract.ErrorEnvelope
	if !errors.As(err, &env) {
		t.Fatalf("error = %T, want *contract.ErrorEnvelope", err)
	}
	if env.GetStatus() != 422 {
		t.Fatalf("status = %d, want 422", env.GetStatus())
	}
	if env.Body.Code != contract.CodeValidationError {
		t.Fatalf("code = %q, want %q", env.Body.Code, contract.CodeValidationError)
	}
	if len(env.Body.Fields["name"]) == 0 {
		t.Fatalf("fields = %#v, want name", env.Body.Fields)
	}
}
