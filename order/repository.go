package order

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"goflare.io/ember"
	"time"

	"github.com/jackc/pgx/v5"
	"gofalre.io/shop/driver"
	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
	"gofalre.io/shop/sqlc"
)

var _ Repository = (*repository)(nil)

type Repository interface {
	CreateOrder(ctx context.Context, tx pgx.Tx, order *models.Order) (*models.Order, error)
	GetOrder(ctx context.Context, tx pgx.Tx, orderID uint64) (*models.Order, error)
	GetOrderByPaymentIntentID(ctx context.Context, tx pgx.Tx, paymentIntentID string) (*models.Order, error)
	GetOrderByRefundID(ctx context.Context, tx pgx.Tx, chargeID string) (*models.Order, error)
	GetOrderByInvoiceID(ctx context.Context, tx pgx.Tx, invoiceID string) (*models.Order, error)
	GetOrderByCustomerIDAndSubscriptionID(ctx context.Context, tx pgx.Tx, customerID, subscriptionID string) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, tx pgx.Tx, orderID uint64, status enum.OrderStatus, updatedAt time.Time) error
	UpdateOrderTotals(ctx context.Context, tx pgx.Tx, orderID uint64, tax, subtotal, discount, total float64, updatedAt time.Time) error
	ListOrders(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.Order, error)
	DeleteOrder(ctx context.Context, tx pgx.Tx, orderID uint64) error

	AddOrderItems(ctx context.Context, tx pgx.Tx, items []*models.OrderItem) error
	ListOrderItems(ctx context.Context, tx pgx.Tx, orderID uint64) ([]*models.OrderItem, error)
	UpdateOrderItem(ctx context.Context, tx pgx.Tx, item *models.OrderItem) error
	DeleteOrderItem(ctx context.Context, tx pgx.Tx, orderItemID uint64) error
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

func (r *repository) CreateOrder(ctx context.Context, tx pgx.Tx, order *models.Order) (*models.Order, error) {
	var cartID uint64
	if order.CartID != nil {
		cartID = *order.CartID
	}
	sqlcOrder, err := sqlc.New(r.conn).WithTx(tx).CreateOrder(ctx, sqlc.CreateOrderParams{
		CustomerID: order.CustomerID,
		CartID:     cartID,
		Status:     sqlc.OrderStatus(order.Status),
		Currency:   sqlc.Currency(order.Currency),
		Subtotal:   order.Subtotal,
		Tax:        order.Tax,
		Total:      order.Total,
		Discount:   order.Discount,
	})
	if err != nil {
		r.logger.Error("Failed to create order", zap.Error(err))
		return nil, err
	}

	createdOrder := new(models.Order).ConvertSqlcOrder(sqlcOrder)

	// 更新快取
	cacheKey := fmt.Sprintf("order:%d", createdOrder.ID)
	if err := r.cache.Set(ctx, cacheKey, createdOrder, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache order", zap.Error(err))
	}

	return createdOrder, nil
}

func (r *repository) GetOrder(ctx context.Context, tx pgx.Tx, orderID uint64) (*models.Order, error) {
	cacheKey := fmt.Sprintf("order:%d", orderID)
	var order models.Order

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &order)
	if err != nil {
		r.logger.Warn("Failed to get order from cache", zap.Error(err))
	}
	if found {
		return &order, nil
	}

	sqlcOrder, err := sqlc.New(r.conn).WithTx(tx).GetOrder(ctx, int32(orderID))
	if err != nil {
		r.logger.Error("Failed to get order", zap.Error(err))
		return nil, err
	}

	order = *new(models.Order).ConvertSqlcOrder(sqlcOrder)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, order, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache order", zap.Error(err))
	}

	return &order, nil
}

func (r *repository) GetOrderByPaymentIntentID(ctx context.Context, tx pgx.Tx, paymentIntentID string) (*models.Order, error) {
	cacheKey := fmt.Sprintf("order:payment_intent:%s", paymentIntentID)
	var order models.Order

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &order)
	if err != nil {
		r.logger.Warn("Failed to get order by payment intent from cache", zap.Error(err))
	}
	if found {
		return &order, nil
	}

	sqlcOrder, err := sqlc.New(r.conn).WithTx(tx).GetOrderByPaymentIntentID(ctx, &paymentIntentID)
	if err != nil {
		r.logger.Error("Failed to get order by payment intent", zap.Error(err))
		return nil, err
	}

	order = *new(models.Order).ConvertSqlcOrder(sqlcOrder)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, order, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache order by payment intent", zap.Error(err))
	}

	return &order, nil
}

func (r *repository) GetOrderByRefundID(ctx context.Context, tx pgx.Tx, chargeID string) (*models.Order, error) {
	cacheKey := fmt.Sprintf("order:refund:%s", chargeID)
	var order models.Order

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &order)
	if err != nil {
		r.logger.Warn("Failed to get order by refund from cache", zap.Error(err))
	}
	if found {
		return &order, nil
	}

	sqlcOrder, err := sqlc.New(r.conn).WithTx(tx).GetOrderByRefundID(ctx, &chargeID)
	if err != nil {
		r.logger.Error("Failed to get order by refund", zap.Error(err))
		return nil, err
	}

	order = *new(models.Order).ConvertSqlcOrder(sqlcOrder)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, order, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache order by refund", zap.Error(err))
	}

	return &order, nil
}

func (r *repository) GetOrderByInvoiceID(ctx context.Context, tx pgx.Tx, invoiceID string) (*models.Order, error) {
	cacheKey := fmt.Sprintf("order:invoice:%s", invoiceID)
	var order models.Order

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &order)
	if err != nil {
		r.logger.Warn("Failed to get order by invoice from cache", zap.Error(err))
	}
	if found {
		return &order, nil
	}

	sqlcOrder, err := sqlc.New(r.conn).WithTx(tx).GetOrderByInvoiceID(ctx, &invoiceID)
	if err != nil {
		r.logger.Error("Failed to get order by invoice", zap.Error(err))
		return nil, err
	}

	order = *new(models.Order).ConvertSqlcOrder(sqlcOrder)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, order, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache order by invoice", zap.Error(err))
	}

	return &order, nil
}

func (r *repository) UpdateOrderStatus(ctx context.Context, tx pgx.Tx, orderID uint64, status enum.OrderStatus, updatedAt time.Time) error {
	err := sqlc.New(r.conn).WithTx(tx).UpdateOrderStatus(ctx, sqlc.UpdateOrderStatusParams{
		ID:        int32(orderID),
		Status:    sqlc.OrderStatus(status),
		UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
	})
	if err != nil {
		r.logger.Error("Failed to update order status", zap.Error(err))
		return err
	}

	// 使相關的快取失效
	r.invalidateOrderCache(ctx, orderID)
	return nil
}

func (r *repository) UpdateOrderTotals(ctx context.Context, tx pgx.Tx, orderID uint64, tax, subtotal, discount, total float64, updatedAt time.Time) error {
	err := sqlc.New(r.conn).WithTx(tx).UpdateOrderTotals(ctx, sqlc.UpdateOrderTotalsParams{
		ID:        int32(orderID),
		Tax:       tax,
		Subtotal:  subtotal,
		Discount:  discount,
		Total:     total,
		UpdatedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
	})
	if err != nil {
		r.logger.Error("Failed to update order totals", zap.Error(err))
		return err
	}

	// 使相關的快取失效
	r.invalidateOrderCache(ctx, orderID)
	return nil
}

func (r *repository) GetOrderByCustomerIDAndSubscriptionID(ctx context.Context, tx pgx.Tx, subscriptionID, customerID string) (*models.Order, error) {
	cacheKey := fmt.Sprintf("order:customer:%s:subscription:%s", customerID, subscriptionID)
	var order models.Order

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &order)
	if err != nil {
		r.logger.Warn("Failed to get order by customer and subscription from cache", zap.Error(err))
	}
	if found {
		return &order, nil
	}

	sqlcOrder, err := sqlc.New(r.conn).WithTx(tx).GetOrderByCustomerIDAndSubscriptionID(ctx, sqlc.GetOrderByCustomerIDAndSubscriptionIDParams{
		CustomerID:     customerID,
		SubscriptionID: &subscriptionID,
	})
	if err != nil {
		r.logger.Error("Failed to get order by customer and subscription", zap.Error(err))
		return nil, err
	}

	order = *new(models.Order).ConvertSqlcOrder(sqlcOrder)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, order, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache order by customer and subscription", zap.Error(err))
	}

	return &order, nil
}

func (r *repository) ListOrders(ctx context.Context, tx pgx.Tx, customerID string, limit, offset uint64) ([]*models.Order, error) {
	cacheKey := fmt.Sprintf("orders:customer:%s:limit:%d:offset:%d", customerID, limit, offset)
	var orders []*models.Order

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &orders)
	if err != nil {
		r.logger.Warn("Failed to get orders from cache", zap.Error(err))
	}
	if found {
		return orders, nil
	}

	sqlcOrders, err := sqlc.New(r.conn).WithTx(tx).ListOrders(ctx, sqlc.ListOrdersParams{
		CustomerID: customerID,
		Limit:      int64(limit),
		Offset:     int64(offset),
	})
	if err != nil {
		r.logger.Error("Failed to list orders", zap.Error(err))
		return nil, err
	}

	orders = make([]*models.Order, 0, len(sqlcOrders))
	for _, sqlcOrder := range sqlcOrders {
		orders = append(orders, new(models.Order).ConvertSqlcOrder(sqlcOrder))
	}

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, orders, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache orders", zap.Error(err))
	}

	return orders, nil
}

func (r *repository) DeleteOrder(ctx context.Context, tx pgx.Tx, orderID uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).DeleteOrder(ctx, int32(orderID))
	if err != nil {
		r.logger.Error("Failed to delete order", zap.Error(err))
		return err
	}

	// 使相關的快取失效
	r.invalidateOrderCache(ctx, orderID)
	return nil
}

func (r *repository) AddOrderItems(ctx context.Context, tx pgx.Tx, items []*models.OrderItem) error {
	var batchError error
	batch := make([]sqlc.AddOrderItemsParams, 0, len(items))
	for _, item := range items {
		batch = append(batch, sqlc.AddOrderItemsParams{
			OrderID:   int32(item.OrderID),
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			PriceID:   item.PriceID,
			StockID:   item.StockID,
			UnitPrice: item.UnitPrice,
			Subtotal:  item.Subtotal,
		})
	}
	batchResults := sqlc.New(r.conn).WithTx(tx).AddOrderItems(ctx, batch)
	defer func(batchResults *sqlc.AddOrderItemsBatchResults) {
		if err := batchResults.Close(); err != nil {
			batchError = err
		}
	}(batchResults)

	batchResults.Exec(func(index int, err error) {
		if err != nil {
			batchError = err
		}
	})

	if batchError != nil {
		r.logger.Error("Failed to add order items", zap.Error(batchError))
		return batchError
	}

	// 使相關的快取失效
	if len(items) > 0 {
		r.invalidateOrderCache(ctx, items[0].OrderID)
	}
	return nil
}

func (r *repository) ListOrderItems(ctx context.Context, tx pgx.Tx, orderID uint64) ([]*models.OrderItem, error) {
	cacheKey := fmt.Sprintf("order_items:%d", orderID)
	var orderItems []*models.OrderItem

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &orderItems)
	if err != nil {
		r.logger.Warn("Failed to get order items from cache", zap.Error(err))
	}
	if found {
		return orderItems, nil
	}

	sqlcOrderItems, err := sqlc.New(r.conn).WithTx(tx).ListOrderItems(ctx, int32(orderID))
	if err != nil {
		r.logger.Error("Failed to list order items", zap.Error(err))
		return nil, err
	}

	orderItems = make([]*models.OrderItem, 0, len(sqlcOrderItems))
	for _, sqlcOrderItem := range sqlcOrderItems {
		orderItems = append(orderItems, new(models.OrderItem).ConvertSqlcOrderItem(sqlcOrderItem))
	}

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, orderItems, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache order items", zap.Error(err))
	}

	return orderItems, nil
}

func (r *repository) UpdateOrderItem(ctx context.Context, tx pgx.Tx, item *models.OrderItem) error {
	err := sqlc.New(r.conn).WithTx(tx).UpdateOrderItem(ctx, sqlc.UpdateOrderItemParams{
		ID:        int32(item.ID),
		Quantity:  item.Quantity,
		UnitPrice: item.UnitPrice,
		Subtotal:  item.Subtotal,
	})
	if err != nil {
		r.logger.Error("Failed to update order item", zap.Error(err))
		return err
	}

	// 使相關的快取失效
	r.invalidateOrderCache(ctx, item.OrderID)
	r.invalidateOrderItemsCache(ctx, item.OrderID)
	return nil
}

func (r *repository) DeleteOrderItem(ctx context.Context, tx pgx.Tx, orderItemID uint64) error {
	// 先獲取 order item 以獲得 order ID
	orderItem, err := sqlc.New(r.conn).WithTx(tx).GetOrderItem(ctx, int32(orderItemID))
	if err != nil {
		r.logger.Error("Failed to get order item", zap.Error(err))
		return err
	}

	err = sqlc.New(r.conn).WithTx(tx).DeleteOrderItem(ctx, int32(orderItemID))
	if err != nil {
		r.logger.Error("Failed to delete order item", zap.Error(err))
		return err
	}

	// 使相關的快取失效
	r.invalidateOrderCache(ctx, uint64(orderItem.OrderID))
	r.invalidateOrderItemsCache(ctx, uint64(orderItem.OrderID))
	return nil
}

func (r *repository) invalidateOrderCache(ctx context.Context, orderID uint64) {
	cacheKeys := []string{
		fmt.Sprintf("order:%d", orderID),
		fmt.Sprintf("order:payment_intent:%d", orderID),
		fmt.Sprintf("order:refund:%d", orderID),
		fmt.Sprintf("order:invoice:%d", orderID),
	}
	for _, key := range cacheKeys {
		if err := r.cache.Delete(ctx, key); err != nil {
			r.logger.Warn("Failed to invalidate order cache", zap.Error(err), zap.String("key", key))
		}
	}
}

func (r *repository) invalidateOrderItemsCache(ctx context.Context, orderID uint64) {
	cacheKey := fmt.Sprintf("order_items:%d", orderID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to invalidate order items cache", zap.Error(err), zap.String("key", cacheKey))
	}
}
