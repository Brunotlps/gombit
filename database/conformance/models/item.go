// Package models holds GORM fixtures for the multi-DB conformance suite.
package models

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Item exercises portable schema features required by M2-4 conformance:
// timestamps (via gorm.Model), nullable columns, unique + non-unique indexes,
// and decimal storage.
type Item struct {
	gorm.Model
	Code  string          `gorm:"size:64;uniqueIndex;not null"`
	Name  string          `gorm:"size:120;index;not null"`
	Notes *string         `gorm:"size:255"`
	Price decimal.Decimal `gorm:"type:decimal(19,4);not null"`
}

// TableName returns the stable table name for Atlas-generated migrations.
func (Item) TableName() string {
	return "items"
}
