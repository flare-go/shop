package stock

import (
	"context"
	"github.com/jackc/pgx/v5"
	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
)

type Repository interface {
	GetStock(ctx context.Context, tx pgx.Tx, stockID uint64) (*models.Stock, error)
	AdjustStock(ctx context.Context, tx pgx.Tx, params []AdjustStockParams) error
	ReleaseStock(ctx context.Context, tx pgx.Tx, params []ReleaseStockParams) error
	ReduceStock(ctx context.Context, tx pgx.Tx, params []ReduceStockParams) error

	CreateStockMovements(ctx context.Context, tx pgx.Tx, params []CreateStockMovementParams) error
	ListStockMovements(ctx context.Context, tx pgx.Tx, stockID uint64, limit, offset uint64) ([]*models.StockMovement, error)
	GetStockMovementsByReference(ctx context.Context, tx pgx.Tx, referenceType enum.StockMovementReferenceType, referenceID uint64) ([]*models.StockMovement, error)
}
