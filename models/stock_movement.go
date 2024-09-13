package models

import (
	"gofalre.io/shop/models/enum"
	"time"
)

type StockMovement struct {
	ID            uint64                          `json:"id"`
	StockID       uint64                          `json:"stock_id"`
	Quantity      uint64                          `json:"quantity"`
	Type          enum.StockMovementType          `json:"type"`
	ReferenceType enum.StockMovementReferenceType `json:"reference_type"`
	ReferenceID   uint64                          `json:"reference_id"`
	CreatedAt     time.Time                       `json:"created_at"`
}
