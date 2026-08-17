package platform

import (
	"github.com/LAA-Software-Engineering/gombit/auth"
	"github.com/LAA-Software-Engineering/gombit/config"
	"github.com/LAA-Software-Engineering/gombit/database"

	"github.com/example/demo/internal/product"
)

// OpenDatabase opens the SQL database from typed config.
func OpenDatabase(cfg config.DatabaseConfig) (*database.DB, error) {
	return database.Open(cfg)
}

// AutoMigrate runs GORM AutoMigrate for runtime auth tables and
// feature-package models so the example API can serve before Atlas
// migrations. Auth models must stay in this call: gombit make resource
// and gombit db makemigrations collect every AutoMigrate argument as
// the entire desired Atlas schema.
func AutoMigrate(db *database.DB) error {
	return db.AutoMigrate(&auth.User{}, &auth.RefreshToken{}, &product.Product{})
}
