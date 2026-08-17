package platform

import (
	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"

	"github.com/example/demo/internal/product"
)

// OpenDatabase opens the SQL database from typed config.
func OpenDatabase(cfg config.DatabaseConfig) (*database.DB, error) {
	return database.Open(cfg)
}

// AutoMigrate runs GORM AutoMigrate for feature-package models so the
// example product API can serve before Atlas migrations exist.
func AutoMigrate(db *database.DB) error {
	return db.AutoMigrate(&product.Product{})
}
