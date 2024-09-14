package category

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"gofalre.io/shop/driver"
	"gofalre.io/shop/models"
	"gofalre.io/shop/sqlc"
	"goflare.io/ember"
	"time"
)

var _ Repository = (*repository)(nil)

type Repository interface {
	Create(ctx context.Context, tx pgx.Tx, category *models.Category) error
	GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Category, error)
	Update(ctx context.Context, tx pgx.Tx, category *models.Category) error
	Delete(ctx context.Context, tx pgx.Tx, id uint64) error
	List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.Category, error)
	ListSubcategories(ctx context.Context, tx pgx.Tx, parentID uint64) ([]*models.Category, error)
	AssignProductToCategory(ctx context.Context, tx pgx.Tx, productID string, categoryID uint64) error
	RemoveProductFromCategory(ctx context.Context, tx pgx.Tx, productID string, categoryID uint64) error
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

func (r *repository) Create(ctx context.Context, tx pgx.Tx, category *models.Category) error {
	err := sqlc.New(r.conn).WithTx(tx).CreateCategory(ctx, sqlc.CreateCategoryParams{
		Name: category.Name,
	})
	if err != nil {
		r.logger.Error("Failed to create category", zap.Error(err))
		return err
	}

	// 更新快取
	cacheKey := fmt.Sprintf("category:%d", category.ID)
	if err := r.cache.Set(ctx, cacheKey, category, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache category", zap.Error(err))
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, tx pgx.Tx, id uint64) (*models.Category, error) {
	cacheKey := fmt.Sprintf("category:%d", id)
	var category models.Category

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &category)
	if err != nil {
		r.logger.Warn("Failed to get category from cache", zap.Error(err))
	}
	if found {
		return &category, nil
	}

	sqlcCategory, err := sqlc.New(r.conn).WithTx(tx).GetCategoryByID(ctx, int32(id))
	if err != nil {
		r.logger.Error("Failed to get category", zap.Error(err))
		return nil, err
	}

	category = *new(models.Category).ConvertSqlcCategory(sqlcCategory)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, category, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache category", zap.Error(err))
	}

	return &category, nil
}

func (r *repository) Update(ctx context.Context, tx pgx.Tx, category *models.Category) error {
	var parentID int32
	if category.ParentID != nil {
		parentID = int32(*category.ParentID)
	}

	err := sqlc.New(r.conn).WithTx(tx).UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		ID:          int32(category.ID),
		Name:        category.Name,
		Description: &category.Description,
		ParentID:    &parentID,
		UpdatedAt:   pgtype.Timestamptz{Time: category.UpdatedAt, Valid: true},
	})
	if err != nil {
		r.logger.Error("Failed to update category", zap.Error(err))
		return err
	}

	// 更新快取
	cacheKey := fmt.Sprintf("category:%d", category.ID)
	if err := r.cache.Set(ctx, cacheKey, category, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to update category in cache", zap.Error(err))
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tx pgx.Tx, id uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).DeleteCategory(ctx, int32(id))
	if err != nil {
		r.logger.Error("Failed to delete category", zap.Error(err))
		return err
	}

	// 從快取中刪除
	cacheKey := fmt.Sprintf("category:%d", id)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to delete category from cache", zap.Error(err))
	}

	return nil
}

func (r *repository) List(ctx context.Context, tx pgx.Tx, limit, offset uint64) ([]*models.Category, error) {
	cacheKey := fmt.Sprintf("categories:%d:%d", limit, offset)
	var categories []*models.Category

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &categories)
	if err != nil {
		r.logger.Warn("Failed to get categories from cache", zap.Error(err))
	}
	if found {
		return categories, nil
	}

	sqlcCategories, err := sqlc.New(r.conn).WithTx(tx).ListCategories(ctx, sqlc.ListCategoriesParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		r.logger.Error("Failed to list categories", zap.Error(err))
		return nil, err
	}

	categories = make([]*models.Category, 0, len(sqlcCategories))
	for _, sqlcCategory := range sqlcCategories {
		categories = append(categories, new(models.Category).ConvertSqlcCategory(sqlcCategory))
	}

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, categories, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache categories", zap.Error(err))
	}

	return categories, nil
}

func (r *repository) ListSubcategories(ctx context.Context, tx pgx.Tx, parentID uint64) ([]*models.Category, error) {
	cacheKey := fmt.Sprintf("subcategories:%d", parentID)
	var categories []*models.Category

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &categories)
	if err != nil {
		r.logger.Warn("Failed to get subcategories from cache", zap.Error(err))
	}
	if found {
		return categories, nil
	}

	categoryParentID := int32(parentID)
	sqlcCategories, err := sqlc.New(r.conn).WithTx(tx).ListSubcategories(ctx, &categoryParentID)
	if err != nil {
		r.logger.Error("Failed to list subcategories", zap.Error(err))
		return nil, err
	}

	categories = make([]*models.Category, 0, len(sqlcCategories))
	for _, sqlcCategory := range sqlcCategories {
		categories = append(categories, new(models.Category).ConvertSqlcCategory(sqlcCategory))
	}

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, categories, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache subcategories", zap.Error(err))
	}

	return categories, nil
}

func (r *repository) AssignProductToCategory(ctx context.Context, tx pgx.Tx, productID string, categoryID uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).AssignProductToCategory(ctx, sqlc.AssignProductToCategoryParams{
		ProductID:  productID,
		CategoryID: int32(categoryID),
	})
	if err != nil {
		r.logger.Error("Failed to assign product to category", zap.Error(err))
		return err
	}

	// 使相關的快取失效
	r.invalidateCategoryCache(ctx, categoryID)
	return nil
}

func (r *repository) RemoveProductFromCategory(ctx context.Context, tx pgx.Tx, productID string, categoryID uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).RemoveProductFromCategory(ctx, sqlc.RemoveProductFromCategoryParams{
		ProductID:  productID,
		CategoryID: int32(categoryID),
	})
	if err != nil {
		r.logger.Error("Failed to remove product from category", zap.Error(err))
		return err
	}

	r.invalidateCategoryCache(ctx, categoryID)
	return nil
}

func (r *repository) invalidateCategoryCache(ctx context.Context, categoryID uint64) {
	cacheKeys := []string{
		fmt.Sprintf("category:%d", categoryID),
		fmt.Sprintf("subcategories:%d", categoryID),
	}
	for _, key := range cacheKeys {
		if err := r.cache.Delete(ctx, key); err != nil {
			r.logger.Warn("Failed to invalidate category cache", zap.Error(err), zap.String("key", key))
		}
	}
}
