package category

import (
	"context"
	"github.com/jackc/pgx/v5"
	"gofalre.io/shop/models"
)

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
