package database

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gombit-dev/gombit/contract"
	"gorm.io/gorm"
)

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "ErrDuplicatedKey", err: gorm.ErrDuplicatedKey, want: true},
		{name: "wrapped ErrDuplicatedKey", err: errors.Join(errors.New("wrap"), gorm.ErrDuplicatedKey), want: true},
		{name: "sqlite unique", err: errors.New("UNIQUE constraint failed: widgets.sku"), want: true},
		{name: "postgres duplicate", err: errors.New("ERROR: duplicate key value violates unique constraint \"widgets_sku_key\" (SQLSTATE 23505)"), want: true},
		{name: "mysql duplicate", err: errors.New("Error 1062 (23000): Duplicate entry 'a' for key 'widgets.sku'"), want: true},
		{name: "record not found", err: gorm.ErrRecordNotFound, want: false},
		{name: "no such table", err: errors.New("no such table: widgets"), want: false},
		{name: "generic", err: errors.New("connection refused"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUniqueViolation(tt.err); got != tt.want {
				t.Fatalf("IsUniqueViolation(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestMapLoadError(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "nil", err: nil, wantStatus: 0, wantCode: ""},
		{name: "not found", err: gorm.ErrRecordNotFound, wantStatus: http.StatusNotFound, wantCode: "not_found"},
		{name: "wrapped not found", err: errors.Join(errors.New("wrap"), gorm.ErrRecordNotFound), wantStatus: http.StatusNotFound, wantCode: "not_found"},
		{name: "no such table", err: errors.New("no such table: widgets"), wantStatus: http.StatusInternalServerError, wantCode: "internal"},
		{name: "unique on load", err: errors.New("UNIQUE constraint failed: widgets.sku"), wantStatus: http.StatusInternalServerError, wantCode: "internal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapLoadError(ctx, tt.err, "widget not found", "load widget")
			assertMapped(t, got, tt.err, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestMapPersistError(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "nil", err: nil, wantStatus: 0, wantCode: ""},
		{name: "sqlite unique", err: errors.New("UNIQUE constraint failed: widgets.sku"), wantStatus: http.StatusConflict, wantCode: "conflict"},
		{name: "ErrDuplicatedKey", err: gorm.ErrDuplicatedKey, wantStatus: http.StatusConflict, wantCode: "conflict"},
		{name: "no such table", err: errors.New("no such table: widgets"), wantStatus: http.StatusInternalServerError, wantCode: "internal"},
		{name: "not found", err: gorm.ErrRecordNotFound, wantStatus: http.StatusInternalServerError, wantCode: "internal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapPersistError(ctx, tt.err, "resource already exists", "persist widget")
			assertMapped(t, got, tt.err, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestMapPersistErrorSQLiteUniqueConstraint(t *testing.T) {
	db := openSQLite(t)
	type uniqueWidget struct {
		ID  uint   `gorm:"primaryKey"`
		SKU string `gorm:"uniqueIndex;size:64;not null"`
	}
	if err := db.AutoMigrate(&uniqueWidget{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	if err := db.Create(&uniqueWidget{SKU: "dup"}).Error; err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	err := db.Create(&uniqueWidget{SKU: "dup"}).Error
	if err == nil {
		t.Fatal("second Create() error = nil, want unique violation")
	}
	if !IsUniqueViolation(err) {
		t.Fatalf("IsUniqueViolation(%v) = false, want true", err)
	}
	mapped := MapPersistError(context.Background(), err, "resource already exists", "persist widget")
	assertMapped(t, mapped, err, http.StatusConflict, "conflict")
}

func assertMapped(t *testing.T, got error, original error, wantStatus int, wantCode string) {
	t.Helper()
	if original == nil {
		if got != nil {
			t.Fatalf("mapped nil error = %v, want nil", got)
		}
		return
	}
	var env *contract.ErrorEnvelope
	if !errors.As(got, &env) {
		t.Fatalf("mapped type = %T (%v), want *contract.ErrorEnvelope", got, got)
	}
	if env.GetStatus() != wantStatus {
		t.Fatalf("status = %d, want %d; code = %q", env.GetStatus(), wantStatus, env.Body.Code)
	}
	if env.Body.Code != wantCode {
		t.Fatalf("code = %q, want %q", env.Body.Code, wantCode)
	}
}
