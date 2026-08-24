package database

import (
	"context"
	"errors"
	"strings"

	"github.com/gombit-dev/gombit/contract"
	"gorm.io/gorm"
)

// IsUniqueViolation reports duplicate-key errors. Open does not set
// gorm.Config.TranslateError, so ErrDuplicatedKey is usually unset and
// the driver error string is the portable signal across SQLite, Postgres,
// and MySQL.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

// MapLoadError maps a GORM read/load error to a D10 category error:
// record-not-found becomes not_found; any other driver failure becomes
// internal. Unique/duplicate is not treated as conflict on load.
func MapLoadError(ctx context.Context, err error, notFound, internal string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return contract.WithContext(ctx, contract.NotFound(notFound))
	}
	return contract.WithContext(ctx, contract.Internal(internal))
}

// MapPersistError maps a GORM write error to a D10 category error:
// unique/duplicate becomes conflict; any other failure becomes internal.
func MapPersistError(ctx context.Context, err error, conflict, internal string) error {
	if err == nil {
		return nil
	}
	if IsUniqueViolation(err) {
		return contract.WithContext(ctx, contract.Conflict(conflict))
	}
	return contract.WithContext(ctx, contract.Internal(internal))
}
