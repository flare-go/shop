package models

import (
	"gofalre.io/shop/models/enum"
	"gofalre.io/shop/sqlc"
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

func (sm *StockMovement) ConvertSqlcStockMovement(sqlcStockMovement any) *StockMovement {

	var id, stockID, referenceID, quantity uint64
	var stockMovementType enum.StockMovementType
	var referenceType enum.StockMovementReferenceType
	var createdAt time.Time

	switch sp := sqlcStockMovement.(type) {
	case *sqlc.StockMovement:
		id = uint64(sp.ID)
		stockID = sp.StockID
		if sp.ReferenceID != nil {
			referenceID = uint64(*sp.ReferenceID)
		}
		quantity = sp.Quantity
		stockMovementType = enum.StockMovementType(sp.Type)
		if sp.ReferenceType.Valid {
			referenceType = enum.StockMovementReferenceType(
				sp.ReferenceType.StockMovementReferenceType)
		}
		createdAt = sp.CreatedAt.Time
	default:
		return nil
	}

	sm.ID = id
	sm.StockID = stockID
	sm.Quantity = quantity
	sm.ReferenceID = referenceID
	sm.ReferenceType = referenceType
	sm.Type = stockMovementType
	sm.CreatedAt = createdAt

	return sm
}
