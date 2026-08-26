// Package project implements the canonical /api/projects CRUD API
// (benchmarks/docs/schema.md) as a normal Gombit feature package: Huma
// handlers, GORM models, framework.App wiring — the same shape
// `gombit make resource` would emit, hand-extended with the update/delete
// and pagination the generator doesn't produce yet.
package project

import "time"

// User and Project are the same schema benchmarks/apps/gin-gorm/models.go
// implements, expressed through this app's own migration mechanism —
// Atlas, via `gombit db makemigrations`/`migrate` (AGENTS.md D3: never
// AutoMigrate for a real Gombit app) — instead of GORM AutoMigrate.
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

// Project.Owner is a belongsTo association, preloaded on the list/get
// queries so the API never N+1s (see Handler.list).
type Project struct {
	ID          uint      `gorm:"primaryKey"`
	OwnerID     uint      `gorm:"index;not null"`
	Owner       User      `gorm:"foreignKey:OwnerID"`
	Name        string    `gorm:"not null"`
	Description string    `gorm:"not null;default:''"`
	CreatedAt   time.Time `gorm:"not null;index"`
	UpdatedAt   time.Time `gorm:"not null"`
}
