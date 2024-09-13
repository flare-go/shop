package cart

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"gofalre.io/shop/driver"
	"gofalre.io/shop/sqlc"

	"github.com/jackc/pgx/v5"
	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
)

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
	conn driver.PostgresPool
}

func NewRepository(conn driver.PostgresPool) Repository {
	return &repository{conn: conn}
}
func (r *repository) CreateCart(ctx context.Context, tx pgx.Tx, cart *models.Cart) error {

	return sqlc.New(r.conn).WithTx(tx).CreateCart(ctx, sqlc.CreateCartParams{
		CustomerID: cart.CustomerID,
		Status:     sqlc.CartStatus(cart.Status),
		Currency:   sqlc.Currency(cart.Currency),
		ExpiresAt:  pgtype.Timestamptz{Time: cart.ExpiresAt, Valid: true},
	})
}

func (r *repository) GetCart(ctx context.Context, tx pgx.Tx, id uint64) (*models.Cart, error) {

	sqlcCart, err := sqlc.New(r.conn).WithTx(tx).GetCart(ctx, id)
	if err != nil {
		return nil, err
	}

	return models.NewCart().ConvertFromSQLCCart(sqlcCart), nil
}

func (r *repository) GetActiveCartByCustomerID(ctx context.Context, tx pgx.Tx, customerID string) (*models.Cart, error) {

	sqlcCart, err := sqlc.New(r.conn).WithTx(tx).FindActiveCartByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return models.NewCart().ConvertFromSQLCCart(sqlcCart), nil
}

func (r *repository) UpdateCartStatus(ctx context.Context, tx pgx.Tx, id uint64, status enum.CartStatus) error {
	return sqlc.New(r.conn).WithTx(tx).UpdateCartStatus(ctx, sqlc.UpdateCartStatusParams{
		ID:     id,
		Status: sqlc.CartStatus(status),
	})
}

func (r *repository) AddCartItem(ctx context.Context, tx pgx.Tx, cartID uint64, item *models.CartItem) error {

	return sqlc.New(r.conn).WithTx(tx).AddCartItem(ctx, sqlc.AddCartItemParams{
		CartID:    cartID,
		ProductID: item.ProductID,
		PriceID:   item.PriceID,
		StockID:   item.StockID,
		Quantity:  item.Quantity,
		UnitPrice: item.UnitPrice,
		Subtotal:  item.Subtotal,
	})
}

func (r *repository) GetCartItem(ctx context.Context, tx pgx.Tx, id uint64) (*models.CartItem, error) {

	sqlcCartItem, err := sqlc.New(r.conn).WithTx(tx).GetCartItem(ctx, id)
	if err != nil {
		return nil, err
	}

	return models.NewCartItem().ConvertFromSQLCCartItem(sqlcCartItem), nil
}

func (r *repository) UpdateCartItem(ctx context.Context, tx pgx.Tx, item *models.CartItem) error {

	return sqlc.New(r.conn).WithTx(tx).UpdateCartItem(ctx, sqlc.UpdateCartItemParams{
		ID:       item.ID,
		Quantity: item.Quantity,
		Subtotal: item.Subtotal,
	})
}

func (r *repository) RemoveCartItem(ctx context.Context, tx pgx.Tx, itemID uint64) error {

	return sqlc.New(r.conn).WithTx(tx).RemoveCartItem(ctx, itemID)
}

func (r *repository) UpdateCartItemQuantity(ctx context.Context, tx pgx.Tx, id, quantity uint64, subtotal float64) error {

	return sqlc.New(r.conn).WithTx(tx).UpdateCartItemQuantity(ctx, sqlc.UpdateCartItemQuantityParams{
		ID:       id,
		Quantity: quantity,
		Subtotal: subtotal,
	})
}

func (r *repository) GetCartItemByProductID(ctx context.Context, tx pgx.Tx, cartID uint64, productID string) (*models.CartItem, error) {

	sqlcCartItem, err := sqlc.New(r.conn).WithTx(tx).FindCartItemByProductID(ctx, sqlc.FindCartItemByProductIDParams{
		CartID:    cartID,
		ProductID: productID,
	})
	if err != nil {
		return nil, err
	}

	return models.NewCartItem().ConvertFromSQLCCartItem(sqlcCartItem), nil
}

func (r *repository) ListCartItems(ctx context.Context, tx pgx.Tx, cartID uint64) ([]*models.CartItem, error) {

	sqlcCartItems, err := sqlc.New(r.conn).WithTx(tx).ListCartItems(ctx, cartID)
	if err != nil {
		return nil, err
	}

	cartItems := make([]*models.CartItem, 0, len(sqlcCartItems))
	for _, sqlcCartItem := range sqlcCartItems {
		cartItems = append(cartItems, models.NewCartItem().ConvertFromSQLCCartItem(sqlcCartItem))
	}

	return cartItems, nil
}

func (r *repository) ClearCartItems(ctx context.Context, tx pgx.Tx, cartID uint64) error {

	return sqlc.New(r.conn).WithTx(tx).ClearCartItems(ctx, cartID)
}

func (r *repository) UpdateCartTotals(ctx context.Context, tx pgx.Tx, id uint64) error {

	return sqlc.New(r.conn).WithTx(tx).UpdateCartTotals(ctx, id)
}
