package cart

import (
	"context"
	"gofalre.io/shop/models"
)

type Service interface {
	CreateCart(ctx context.Context, customerID string, currency string) (*models.Cart, error)
	GetCart(ctx context.Context, id int) (*models.Cart, error)
	AddItemToCart(ctx context.Context, cartID int, item *models.CartItem) error
	RemoveItemFromCart(ctx context.Context, cartID int, itemID int) error
	UpdateCartItemQuantity(ctx context.Context, cartID int, itemID int, quantity int) error
	AbandonCart(ctx context.Context, id int) error
	ConvertCartToOrder(ctx context.Context, cartID int) (*models.Order, error)
}
