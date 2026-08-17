package auth

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// User is the v0.1 auth identity. Groups and permissions are out of scope
// until createsuperuser / admin work; email + password hash is enough.
type User struct {
	ID           uint   `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// RefreshToken is a hashed, rotating refresh credential.
type RefreshToken struct {
	ID         uint       `gorm:"primaryKey"`
	UserID     uint       `gorm:"index;not null"`
	TokenHash  string     `gorm:"uniqueIndex;size:64;not null"`
	ExpiresAt  time.Time  `gorm:"index;not null"`
	RevokedAt  *time.Time `gorm:"index"`
	ReplacedBy *uint
	CreatedAt  time.Time
}

// Models returns GORM models for Atlas / AutoMigrate.
func Models() []any {
	return []any{&User{}, &RefreshToken{}}
}

// Migrate creates the users and refresh_tokens tables.
func Migrate(db *gorm.DB) error {
	if db == nil {
		return errors.New("auth: nil database")
	}
	return db.AutoMigrate(Models()...)
}
