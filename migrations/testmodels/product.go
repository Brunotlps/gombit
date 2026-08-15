package testmodels

import "gorm.io/gorm"

// Product is a GORM model used by makemigrations integration tests.
type Product struct {
	gorm.Model
	Name  string `gorm:"size:120;not null"`
	Price int64  `gorm:"not null"`
}
