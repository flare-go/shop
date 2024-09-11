package order

import (
	"context"

	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
)

type Service interface {
	CreateOrder(ctx context.Context, order *models.Order) error
	GetOrder(ctx context.Context, id int) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, id int, status enum.OrderStatus) error
	ListOrders(ctx context.Context, customerID string, limit, offset int) ([]*models.Order, error)
	ProcessPayment(ctx context.Context, orderID int, paymentIntentID string) error
}
