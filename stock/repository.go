package stock

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"gofalre.io/shop/driver"
	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
	"gofalre.io/shop/sqlc"
	"goflare.io/ember"
	"time"
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

type repository struct {
	conn   driver.PostgresPool
	cache  *ember.Ember
	logger *zap.Logger
}

func NewRepository(conn driver.PostgresPool, cache *ember.Ember, logger *zap.Logger) Repository {
	return &repository{
		conn:   conn,
		cache:  cache,
		logger: logger,
	}
}

func (r *repository) GetStock(ctx context.Context, tx pgx.Tx, stockID uint64) (*models.Stock, error) {
	cacheKey := fmt.Sprintf("stock:%d", stockID)
	var stock models.Stock

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &stock)
	if err != nil {
		r.logger.Error("failed to get stock", zap.Uint64("stock_id", stockID), zap.Error(err))
	}
	if found {
		r.logger.Info("found stock", zap.Uint64("stock_id", stockID))
		return &stock, nil
	}

	// 從資料庫中獲取
	sqlcStock, err := sqlc.New(r.conn).WithTx(tx).GetStock(ctx, int32(stockID))
	if err != nil {
		r.logger.Error("failed to get stock", zap.Uint64("stock_id", stockID), zap.Error(err))
		return nil, err
	}

	stock = *new(models.Stock).ConvertSqlcStock(sqlcStock)

	if err = r.cache.Set(ctx, cacheKey, stock); err != nil {
		r.logger.Error("failed to cache stock", zap.Uint64("stock_id", stockID), zap.Error(err))
	}

	return &stock, nil
}

func (r *repository) AdjustStock(ctx context.Context, tx pgx.Tx, params []AdjustStockParams) error {
	var batchError error
	batch := make([]sqlc.AdjustStockParams, 0, len(params))
	for _, param := range params {
		batch = append(batch, sqlc.AdjustStockParams{
			ID:               int32(param.StockID),
			ReservedQuantity: int32(param.Quantity),
			UpdatedAt:        pgtype.Timestamptz{Time: param.LastUpdated, Valid: true},
		})
	}
	batchResults := sqlc.New(r.conn).WithTx(tx).AdjustStock(ctx, batch)
	defer func(batchResults *sqlc.AdjustStockBatchResults) {
		if err := batchResults.Close(); err != nil {
			r.logger.Error("failed to close batch", zap.Error(err))
		}
	}(batchResults)

	batchResults.Exec(func(index int, err error) {
		if err != nil {
			r.logger.Error("failed to execute batch", zap.Error(err))
			batchError = err
			return
		}
		// 更新快取
		stockID := params[index].StockID
		r.updateStockCache(ctx, stockID)
	})

	return batchError
}

func (r *repository) ReleaseStock(ctx context.Context, tx pgx.Tx, params []ReleaseStockParams) error {
	var batchError error
	batch := make([]sqlc.ReleaseStockParams, 0, len(params))
	for _, param := range params {
		batch = append(batch, sqlc.ReleaseStockParams{
			ID:               int32(param.StockID),
			ReservedQuantity: int32(param.Quantity),
			UpdatedAt:        pgtype.Timestamptz{Time: param.LastUpdated, Valid: true},
		})
	}
	batchResults := sqlc.New(r.conn).WithTx(tx).ReleaseStock(ctx, batch)
	defer func(batchResults *sqlc.ReleaseStockBatchResults) {
		if err := batchResults.Close(); err != nil {
			r.logger.Error("failed to close batch", zap.Error(err))
		}
	}(batchResults)

	batchResults.Exec(func(index int, err error) {
		if err != nil {
			r.logger.Error("failed to execute batch", zap.Error(err))
			batchError = err
			return
		}
		// 更新快取
		stockID := params[index].StockID
		r.updateStockCache(ctx, stockID)
	})

	return batchError
}

func (r *repository) ReduceStock(ctx context.Context, tx pgx.Tx, params []ReduceStockParams) error {
	var batchError error
	batch := make([]sqlc.ReduceStockParams, 0, len(params))
	for _, param := range params {
		batch = append(batch, sqlc.ReduceStockParams{
			ID:        int32(param.StockID),
			Quantity:  param.Quantity,
			UpdatedAt: pgtype.Timestamptz{Time: param.LastUpdated, Valid: true},
		})
	}
	batchResults := sqlc.New(r.conn).WithTx(tx).ReduceStock(ctx, batch)
	defer func(batchResults *sqlc.ReduceStockBatchResults) {
		if err := batchResults.Close(); err != nil {
			r.logger.Error("failed to close batch", zap.Error(err))
		}
	}(batchResults)

	batchResults.Exec(func(index int, err error) {
		if err != nil {
			r.logger.Error("failed to execute batch", zap.Error(err))
			batchError = err
			return
		}
		// 更新快取
		stockID := params[index].StockID
		r.updateStockCache(ctx, stockID)
	})

	return batchError
}

func (r *repository) updateStockCache(ctx context.Context, stockID uint64) {
	stock, err := r.GetStock(ctx, nil, stockID)
	if err != nil {
		r.logger.Error("failed to get stock", zap.Uint64("stock_id", stockID), zap.Error(err))
		return
	}

	cacheKey := fmt.Sprintf("stock:%d", stockID)
	if err = r.cache.Set(ctx, cacheKey, stock, 5*time.Minute); err != nil {
		r.logger.Error("failed to cache stock", zap.Uint64("stock_id", stockID), zap.Error(err))
	}
}

func (r *repository) CreateStockMovements(ctx context.Context, tx pgx.Tx, params []CreateStockMovementParams) error {
	var batchError error
	batch := make([]sqlc.CreateStockMovementParams, 0, len(params))
	for _, param := range params {
		refID := int32(param.ReferenceID)
		batch = append(batch, sqlc.CreateStockMovementParams{
			StockID:     param.StockID,
			Quantity:    param.Quantity,
			Type:        sqlc.StockMovementType(param.ReferenceType),
			ReferenceID: &refID,
			ReferenceType: sqlc.NullStockMovementReferenceType{
				StockMovementReferenceType: sqlc.StockMovementReferenceType(param.ReferenceType),
				Valid:                      param.ReferenceType != "",
			},
		})
	}
	batchResults := sqlc.New(r.conn).WithTx(tx).CreateStockMovement(ctx, batch)
	defer func(batchResults *sqlc.CreateStockMovementBatchResults) {
		if err := batchResults.Close(); err != nil {
			r.logger.Error("failed to close batch", zap.Error(err))
		}
	}(batchResults)

	batchResults.Exec(func(index int, err error) {
		if err != nil {
			r.logger.Error("failed to execute batch", zap.Error(err))
			batchError = err
			return
		}
		// 更新相關的庫存快取
		stockID := params[index].StockID
		r.updateStockCache(ctx, stockID)
	})

	return batchError
}

func (r *repository) ListStockMovements(ctx context.Context, tx pgx.Tx, stockID uint64, limit, offset uint64) ([]*models.StockMovement, error) {
	cacheKey := fmt.Sprintf("stock_movements:%d:%d:%d", stockID, limit, offset)
	var stockMovements []*models.StockMovement

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &stockMovements)
	if err != nil {
		r.logger.Error("failed to get stock movements", zap.Uint64("stock_id", stockID), zap.Error(err))
	}
	if found {
		r.logger.Info("found stock movements", zap.Uint64("stock_id", stockID))
		return stockMovements, nil
	}

	sqlcStockMovements, err := sqlc.New(r.conn).WithTx(tx).ListStockMovements(ctx, sqlc.ListStockMovementsParams{
		StockID: stockID,
		Limit:   int64(limit),
		Offset:  int64(offset),
	})

	if err != nil {
		r.logger.Error("failed to list stock movements", zap.Error(err))
		return nil, err
	}

	stockMovements = make([]*models.StockMovement, 0, len(sqlcStockMovements))
	for _, sqlcStockMovement := range sqlcStockMovements {
		stockMovements = append(stockMovements,
			new(models.StockMovement).ConvertSqlcStockMovement(sqlcStockMovement))
	}

	// 設置快取
	if err = r.cache.Set(ctx, cacheKey, stockMovements, 5*time.Minute); err != nil {
		r.logger.Error("failed to cache stock movements", zap.Error(err))
	}

	return stockMovements, nil
}

func (r *repository) GetStockMovementsByReference(ctx context.Context, tx pgx.Tx, referenceType enum.StockMovementReferenceType, referenceID uint64) ([]*models.StockMovement, error) {
	cacheKey := fmt.Sprintf("stock_movements_ref:%s:%d", referenceType, referenceID)
	var stockMovements []*models.StockMovement

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &stockMovements)
	if err != nil {
		r.logger.Error("failed to get stock movements", zap.Error(err))
	}
	if found {
		r.logger.Info("found stock movements", zap.Uint64("stock_id", referenceID))
		return stockMovements, nil
	}

	refID := int32(referenceID)
	sqlcStockMovements, err := sqlc.New(r.conn).WithTx(tx).GetStockMovementsByReference(ctx,
		sqlc.GetStockMovementsByReferenceParams{
			ReferenceID: &refID,
			ReferenceType: sqlc.NullStockMovementReferenceType{
				StockMovementReferenceType: sqlc.StockMovementReferenceType(referenceType),
				Valid:                      referenceType != "",
			},
		})
	if err != nil {
		r.logger.Error("failed to get stock movements", zap.Error(err))
		return nil, err
	}

	stockMovements = make([]*models.StockMovement, 0, len(sqlcStockMovements))
	for _, sqlcStockMovement := range sqlcStockMovements {
		stockMovements = append(stockMovements,
			new(models.StockMovement).ConvertSqlcStockMovement(sqlcStockMovement))
	}

	// 設置快取
	if err = r.cache.Set(ctx, cacheKey, stockMovements, 5*time.Minute); err != nil {
		r.logger.Error("failed to cache stock movements", zap.Error(err))
	}

	return stockMovements, nil
}
