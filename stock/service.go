package stock

import (
	"context"

	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
)

type Service interface {
	CreateStock(ctx context.Context, stock *models.Stock) error
	GetStock(ctx context.Context, id int) (*models.Stock, error)
	UpdateStock(ctx context.Context, stock *models.Stock) error
	ListStocks(ctx context.Context, productID string, limit, offset int) ([]*models.Stock, error)
	AdjustStock(ctx context.Context, stockID int, quantity int, movementType enum.StockMovementType, referenceType enum.StockMovementReferenceType, referenceID int) error
}
