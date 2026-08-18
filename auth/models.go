package auth

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// User is the auth identity. IsSuperuser bypasses every permission check;
// regular users receive permissions directly or through groups.
type User struct {
	ID           uint         `gorm:"primaryKey"`
	Email        string       `gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string       `gorm:"not null"`
	IsSuperuser  bool         `gorm:"not null;default:false"`
	Groups       []Group      `gorm:"many2many:auth_user_groups;"`
	Permissions  []Permission `gorm:"many2many:auth_user_permissions;"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Group collects permissions that can be assigned to users.
type Group struct {
	ID          uint         `gorm:"primaryKey"`
	Name        string       `gorm:"uniqueIndex;not null;size:100"`
	Permissions []Permission `gorm:"many2many:auth_group_permissions;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Permission is a stable authorization key, such as admin.widgets.view.
type Permission struct {
	ID          uint   `gorm:"primaryKey"`
	Key         string `gorm:"column:permission_key;uniqueIndex;not null;size:120"` // not "key": reserved in MySQL
	Description string `gorm:"size:255"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
	return []any{&User{}, &RefreshToken{}, &Group{}, &Permission{}}
}

// Migrate creates the auth identity, token, group, and permission tables.
func Migrate(db *gorm.DB) error {
	if db == nil {
		return errors.New("auth: nil database")
	}
	return db.AutoMigrate(Models()...)
}
