package product

import "gorm.io/gorm"

// Product is a minimal feature-package model for migration examples.
type Product struct {
	gorm.Model
	Name  string `gorm:"size:120;not null"`
	Price int64  `gorm:"not null"`
}
