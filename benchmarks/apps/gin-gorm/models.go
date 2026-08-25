package main

import "time"

// User and Project implement the canonical schema documented in
// benchmarks/docs/schema.md. Plain explicit fields, not gorm.Model: the
// canonical schema has no soft-delete column, and auto-increment uint IDs
// (not UUIDs) match this repo's existing convention (auth.User, the
// database conformance models).
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

// Project.Owner is this repo's first belongsTo GORM association (no
// existing feature package needed one before this). OwnerID is a plain
// indexed scalar FK, matching the existing FK-as-index convention
// (auth.RefreshToken.UserID); Owner is populated via .Preload("Owner") on
// the list query so /api/projects never N+1s.
type Project struct {
	ID          uint      `gorm:"primaryKey"`
	OwnerID     uint      `gorm:"index;not null"`
	Owner       User      `gorm:"foreignKey:OwnerID"`
	Name        string    `gorm:"not null"`
	Description string    `gorm:"not null;default:''"`
	CreatedAt   time.Time `gorm:"not null;index"`
	UpdatedAt   time.Time `gorm:"not null"`
}
