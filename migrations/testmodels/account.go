package testmodels

import "gorm.io/gorm"

// Account is a second, unrelated GORM model used by makemigrations
// integration tests to exercise a migration that adds one new model without
// repeating a model an earlier migration already covers.
type Account struct {
	gorm.Model
	Owner string `gorm:"size:120;not null"`
}
