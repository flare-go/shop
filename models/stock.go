package models

import "time"

type Stock struct {
	ID               uint64    `json:"id"`
	ProductID        string    `json:"product_id"`
	Quantity         uint64    `json:"quantity"`
	ReservedQuantity uint64    `json:"reserved_quantity"`
	Location         string    `json:"location"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
