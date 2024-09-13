package order

import (
	"context"
	"github.com/jackc/pgx/v5"
	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
)

type Repository interface {
	CreateOrder(ctx context.Context, tx pgx.Tx, order *models.Order) error
	GetOrder(ctx context.Context, tx pgx.Tx, orderID uint64) (*models.Order, error)
	GetOrderByPaymentIntentID(ctx context.Context, tx pgx.Tx, paymentIntentID string) (*models.Order, error)
	GetOrderByChargeID(ctx context.Context, tx pgx.Tx, chargeID string) (*models.Order, error)
	GetOrderByInvoiceID(ctx context.Context, tx pgx.Tx, invoiceID string) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, tx pgx.Tx, orderID uint64, status enum.OrderStatus) error
	UpdateOrderTotals(ctx context.Context, tx pgx.Tx, orderID uint64) error
	UpdateOrderStatusBySubscriptionID(ctx context.Context, tx pgx.Tx, subscription string, status enum.OrderStatus) error
	ListOrders(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.Order, error)
	DeleteOrder(ctx context.Context, tx pgx.Tx, orderID uint64) error

	AddOrderItems(ctx context.Context, tx pgx.Tx, items []*models.OrderItem) error
	GetOrderItem(ctx context.Context, tx pgx.Tx, orderItemID uint64) (*models.OrderItem, error)
	ListOrderItems(ctx context.Context, tx pgx.Tx, orderID uint64) ([]*models.OrderItem, error)
	UpdateOrderItem(ctx context.Context, tx pgx.Tx, item *models.OrderItem) error
	DeleteOrderItem(ctx context.Context, tx pgx.Tx, orderItemID uint64) error
}
