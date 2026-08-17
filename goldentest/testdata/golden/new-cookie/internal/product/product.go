package product

import "gorm.io/gorm"

// Product is the example feature-package GORM model.
type Product struct {
	gorm.Model
	Name  string `gorm:"size:120;not null"`
	Price int64  `gorm:"not null"`
}
