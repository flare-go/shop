package cart

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

var _ Repository = (*repository)(nil)

type Repository interface {
	CreateCart(ctx context.Context, tx pgx.Tx, cart *models.Cart) error
	GetCart(ctx context.Context, tx pgx.Tx, id uint64) (*models.Cart, error)
	GetActiveCartByCustomerID(ctx context.Context, tx pgx.Tx, customerID string) (*models.Cart, error)
	GetCartItemByProductID(ctx context.Context, tx pgx.Tx, cartID uint64, productID string) (*models.CartItem, error)
	AddCartItem(ctx context.Context, tx pgx.Tx, cartID uint64, item *models.CartItem) error
	RemoveCartItem(ctx context.Context, tx pgx.Tx, cartItemID uint64) error
	ListCartItems(ctx context.Context, tx pgx.Tx, cartID uint64) ([]*models.CartItem, error)
	ClearCartItems(ctx context.Context, tx pgx.Tx, cartID uint64) error
	UpdateCartStatus(ctx context.Context, tx pgx.Tx, id uint64, status enum.CartStatus) error
	GetCartItem(ctx context.Context, tx pgx.Tx, id uint64) (*models.CartItem, error)
	UpdateCartItem(ctx context.Context, tx pgx.Tx, cartItem *models.CartItem) error
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

func (r *repository) CreateCart(ctx context.Context, tx pgx.Tx, cart *models.Cart) error {
	err := sqlc.New(r.conn).WithTx(tx).CreateCart(ctx, sqlc.CreateCartParams{
		CustomerID: cart.CustomerID,
		Status:     sqlc.CartStatus(cart.Status),
		Currency:   sqlc.Currency(cart.Currency),
		ExpiresAt:  pgtype.Timestamptz{Time: cart.ExpiresAt, Valid: true},
	})
	if err != nil {
		r.logger.Error("Failed to create cart", zap.Error(err))
		return err
	}

	// 更新快取
	cacheKey := fmt.Sprintf("cart:%d", cart.ID)
	if err := r.cache.Set(ctx, cacheKey, cart, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache cart", zap.Error(err))
	}

	return nil
}

func (r *repository) GetCart(ctx context.Context, tx pgx.Tx, id uint64) (*models.Cart, error) {
	cacheKey := fmt.Sprintf("cart:%d", id)
	var cart models.Cart

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &cart)
	if err != nil {
		r.logger.Warn("Failed to get cart from cache", zap.Error(err))
	}
	if found {
		return &cart, nil
	}

	sqlcCart, err := sqlc.New(r.conn).WithTx(tx).GetCart(ctx, int32(id))
	if err != nil {
		r.logger.Error("Failed to get cart", zap.Error(err))
		return nil, err
	}

	cart = *new(models.Cart).ConvertSqlcCart(sqlcCart)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, cart, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache cart", zap.Error(err))
	}

	return &cart, nil
}

func (r *repository) GetActiveCartByCustomerID(ctx context.Context, tx pgx.Tx, customerID string) (*models.Cart, error) {
	cacheKey := fmt.Sprintf("active_cart:%s", customerID)
	var cart models.Cart

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &cart)
	if err != nil {
		r.logger.Warn("Failed to get active cart from cache", zap.Error(err))
	}
	if found {
		return &cart, nil
	}

	sqlcCart, err := sqlc.New(r.conn).WithTx(tx).FindActiveCartByCustomerID(ctx, customerID)
	if err != nil {
		r.logger.Error("Failed to get active cart", zap.Error(err))
		return nil, err
	}

	cart = *new(models.Cart).ConvertSqlcCart(sqlcCart)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, cart, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache active cart", zap.Error(err))
	}

	return &cart, nil
}

func (r *repository) UpdateCartStatus(ctx context.Context, tx pgx.Tx, id uint64, status enum.CartStatus) error {
	err := sqlc.New(r.conn).WithTx(tx).UpdateCartStatus(ctx, sqlc.UpdateCartStatusParams{
		ID:     int32(id),
		Status: sqlc.CartStatus(status),
	})
	if err != nil {
		r.logger.Error("Failed to update cart status", zap.Error(err))
		return err
	}

	// 更新快取
	r.invalidateCartCache(ctx, id)

	return nil
}

func (r *repository) AddCartItem(ctx context.Context, tx pgx.Tx, cartID uint64, item *models.CartItem) error {
	err := sqlc.New(r.conn).WithTx(tx).AddCartItem(ctx, sqlc.AddCartItemParams{
		CartID:    cartID,
		ProductID: item.ProductID,
		PriceID:   item.PriceID,
		StockID:   item.StockID,
		Quantity:  item.Quantity,
		UnitPrice: item.UnitPrice,
		Subtotal:  item.Subtotal,
	})
	if err != nil {
		r.logger.Error("Failed to add cart item", zap.Error(err))
		return err
	}

	// 更新快取
	r.invalidateCartCache(ctx, cartID)
	r.invalidateCartItemsCache(ctx, cartID)

	return nil
}

func (r *repository) GetCartItem(ctx context.Context, tx pgx.Tx, id uint64) (*models.CartItem, error) {
	cacheKey := fmt.Sprintf("cart_item:%d", id)
	var cartItem models.CartItem

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &cartItem)
	if err != nil {
		r.logger.Warn("Failed to get cart item from cache", zap.Error(err))
	}
	if found {
		return &cartItem, nil
	}

	sqlcCartItem, err := sqlc.New(r.conn).WithTx(tx).GetCartItem(ctx, int32(id))
	if err != nil {
		r.logger.Error("Failed to get cart item", zap.Error(err))
		return nil, err
	}

	cartItem = *new(models.CartItem).ConvertSqlcCartItem(sqlcCartItem)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, cartItem, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache cart item", zap.Error(err))
	}

	return &cartItem, nil
}

func (r *repository) UpdateCartItem(ctx context.Context, tx pgx.Tx, item *models.CartItem) error {
	err := sqlc.New(r.conn).WithTx(tx).UpdateCartItem(ctx, sqlc.UpdateCartItemParams{
		ID:       int32(item.ID),
		Quantity: item.Quantity,
		Subtotal: item.Subtotal,
	})
	if err != nil {
		r.logger.Error("Failed to update cart item", zap.Error(err))
		return err
	}

	// 更新快取
	r.invalidateCartCache(ctx, item.CartID)
	r.invalidateCartItemsCache(ctx, item.CartID)
	cacheKey := fmt.Sprintf("cart_item:%d", item.ID)
	if err := r.cache.Set(ctx, cacheKey, item, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache updated cart item", zap.Error(err))
	}

	return nil
}

func (r *repository) RemoveCartItem(ctx context.Context, tx pgx.Tx, itemID uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).RemoveCartItem(ctx, int32(itemID))
	if err != nil {
		r.logger.Error("Failed to remove cart item", zap.Error(err))
		return err
	}

	// 更新快取
	cacheKey := fmt.Sprintf("cart_item:%d", itemID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to remove cart item from cache", zap.Error(err))
	}

	return nil
}

func (r *repository) GetCartItemByProductID(ctx context.Context, tx pgx.Tx, cartID uint64, productID string) (*models.CartItem, error) {
	cacheKey := fmt.Sprintf("cart_item:%d:%s", cartID, productID)
	var cartItem models.CartItem

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &cartItem)
	if err != nil {
		r.logger.Warn("Failed to get cart item by product ID from cache", zap.Error(err))
	}
	if found {
		return &cartItem, nil
	}

	sqlcCartItem, err := sqlc.New(r.conn).WithTx(tx).FindCartItemByProductID(ctx, sqlc.FindCartItemByProductIDParams{
		CartID:    cartID,
		ProductID: productID,
	})
	if err != nil {
		r.logger.Error("Failed to get cart item by product ID", zap.Error(err))
		return nil, err
	}

	cartItem = *new(models.CartItem).ConvertSqlcCartItem(sqlcCartItem)

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, cartItem, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache cart item by product ID", zap.Error(err))
	}

	return &cartItem, nil
}

func (r *repository) ListCartItems(ctx context.Context, tx pgx.Tx, cartID uint64) ([]*models.CartItem, error) {
	cacheKey := fmt.Sprintf("cart_items:%d", cartID)
	var cartItems []*models.CartItem

	// 嘗試從快取中獲取
	found, err := r.cache.Get(ctx, cacheKey, &cartItems)
	if err != nil {
		r.logger.Warn("Failed to get cart items from cache", zap.Error(err))
	}
	if found {
		return cartItems, nil
	}

	sqlcCartItems, err := sqlc.New(r.conn).WithTx(tx).ListCartItems(ctx, cartID)
	if err != nil {
		r.logger.Error("Failed to list cart items", zap.Error(err))
		return nil, err
	}

	cartItems = make([]*models.CartItem, 0, len(sqlcCartItems))
	for _, sqlcCartItem := range sqlcCartItems {
		cartItems = append(cartItems, new(models.CartItem).ConvertSqlcCartItem(sqlcCartItem))
	}

	// 更新快取
	if err := r.cache.Set(ctx, cacheKey, cartItems, 30*time.Minute); err != nil {
		r.logger.Warn("Failed to cache cart items", zap.Error(err))
	}

	return cartItems, nil
}

func (r *repository) ClearCartItems(ctx context.Context, tx pgx.Tx, cartID uint64) error {
	err := sqlc.New(r.conn).WithTx(tx).ClearCartItems(ctx, cartID)
	if err != nil {
		r.logger.Error("Failed to clear cart items", zap.Error(err))
		return err
	}

	// 更新快取
	r.invalidateCartCache(ctx, cartID)
	r.invalidateCartItemsCache(ctx, cartID)

	return nil
}

func (r *repository) invalidateCartCache(ctx context.Context, cartID uint64) {
	cacheKey := fmt.Sprintf("cart:%d", cartID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to invalidate cart cache", zap.Error(err))
	}
}

func (r *repository) invalidateCartItemsCache(ctx context.Context, cartID uint64) {
	cacheKey := fmt.Sprintf("cart_items:%d", cartID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		r.logger.Warn("Failed to invalidate cart items cache", zap.Error(err))
	}
}
