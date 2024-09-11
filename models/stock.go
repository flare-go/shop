package models

import "time"

type Stock struct {
	ID               int       `json:"id"`
	ProductID        string    `json:"product_id"`
	Quantity         int       `json:"quantity"`
	ReservedQuantity int       `json:"reserved_quantity"`
	Location         string    `json:"location"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
