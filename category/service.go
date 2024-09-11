package category

import (
	"context"
	"gofalre.io/shop/models"
)

type Service interface {
	CreateCategory(ctx context.Context, category *models.Category) error
	GetCategory(ctx context.Context, id int) (*models.Category, error)
	UpdateCategory(ctx context.Context, category *models.Category) error
	DeleteCategory(ctx context.Context, id int) error
	ListCategories(ctx context.Context, limit, offset int) ([]*models.Category, error)
}
