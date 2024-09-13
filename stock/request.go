package stock

import (
	"gofalre.io/shop/models/enum"
	"time"
)

type AdjustStockParams struct {
	StockID     uint64
	Quantity    int64
	LastUpdated time.Time
}

type ReleaseStockParams struct {
	StockID     uint64
	Quantity    int64
	LastUpdated time.Time
}

type ReduceStockParams struct {
	StockID     uint64
	Quantity    int64
	LastUpdated time.Time
}

type CreateStockMovementParams struct {
	StockID       uint64
	Quantity      int64
	Type          enum.StockMovementType
	ReferenceID   uint64
	ReferenceType enum.StockMovementReferenceType
}
