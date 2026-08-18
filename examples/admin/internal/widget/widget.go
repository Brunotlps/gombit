package widget

import "time"

// Widget is the small model registered with the admin in this example.
type Widget struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
