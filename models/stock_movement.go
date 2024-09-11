package models

import (
	"gofalre.io/shop/models/enum"
	"time"
)

type StockMovement struct {
	ID            int                             `json:"id"`
	StockID       int                             `json:"stock_id"`
	Quantity      int                             `json:"quantity"`
	Type          enum.StockMovementType          `json:"type"`
	ReferenceType enum.StockMovementReferenceType `json:"reference_type"`
	ReferenceID   int                             `json:"reference_id"`
	CreatedAt     time.Time                       `json:"created_at"`
}
