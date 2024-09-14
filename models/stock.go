package models

import (
	"gofalre.io/shop/sqlc"
	"time"
)

type Stock struct {
	ID               uint64    `json:"id"`
	ProductID        string    `json:"product_id"`
	Quantity         uint64    `json:"quantity"`
	ReservedQuantity uint64    `json:"reserved_quantity"`
	Location         string    `json:"location"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (s *Stock) ConvertSqlcStock(sqlcStock any) *Stock {

	var id, quantity, reservedQuantity uint64
	var productID, location string
	var createdAt, updatedAt time.Time

	switch sp := sqlcStock.(type) {
	case *sqlc.Stock:
		id = uint64(sp.ID)
		quantity = sp.Quantity
		reservedQuantity = uint64(sp.ReservedQuantity)
		productID = sp.ProductID
		if sp.Location != nil {
			location = *sp.Location
		}
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
	default:
		return nil
	}

	s.ID = id
	s.ProductID = productID
	s.Quantity = quantity
	s.ReservedQuantity = reservedQuantity
	s.Location = location
	s.CreatedAt = createdAt
	s.UpdatedAt = updatedAt

	return s
}
