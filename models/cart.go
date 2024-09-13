package models

import (
	"github.com/stripe/stripe-go/v79"
	"gofalre.io/shop/models/enum"
	"gofalre.io/shop/sqlc"
	"time"
)

// Cart 代表購物車
type Cart struct {
	ID         uint64          `json:"id"`
	CustomerID string          `json:"customer_id"`
	Status     enum.CartStatus `json:"status"`
	Currency   stripe.Currency `json:"currency"`
	Subtotal   float64         `json:"subtotal"`
	Tax        float64         `json:"tax"`
	Discount   float64         `json:"discount"`
	Total      float64         `json:"total"`
	Items      []CartItem      `json:"items"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

// CartItem 代表購物車中的單個商品項目
type CartItem struct {
	ID        uint64  `json:"id"`
	CartID    uint64  `json:"cart_id"`
	ProductID string  `json:"product_id"`
	PriceID   string  `json:"price_id"`
	StockID   uint64  `json:"stock_id"`
	Quantity  uint64  `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
}

func NewCart() *Cart {
	return new(Cart)
}

func NewCartItem() *CartItem {
	return new(CartItem)
}

func (c *Cart) ConvertFromSQLCCart(sqlcCart any) *Cart {

	var id uint64
	var customerID string
	var status enum.CartStatus
	var currency stripe.Currency
	var subtotal, tax, discount, total float64
	var createdAt, updatedAt, expiresAt time.Time

	switch sp := sqlcCart.(type) {
	case *sqlc.Cart:
		id = sp.ID
		customerID = sp.CustomerID
		status = enum.CartStatus(sp.Status)
		currency = stripe.Currency(sp.Currency)
		subtotal = sp.Subtotal
		taxFloat8, _ := sp.Tax.Float64Value()
		if taxFloat8.Valid {
			subtotal = taxFloat8.Float64
		}
		discountFloat8, _ := sp.Discount.Float64Value()
		if discountFloat8.Valid {
			subtotal = discountFloat8.Float64
		}
		totalFloat8, _ := sp.Total.Float64Value()
		if totalFloat8.Valid {
			subtotal = totalFloat8.Float64
		}
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
		expiresAt = sp.ExpiresAt.Time
	case *sqlc.GetCartRow:
		id = sp.ID
		customerID = sp.CustomerID
		status = enum.CartStatus(sp.Status)
		currency = stripe.Currency(sp.Currency)
		subtotal = sp.Subtotal
		taxFloat8, _ := sp.Tax.Float64Value()
		if taxFloat8.Valid {
			subtotal = taxFloat8.Float64
		}
		discountFloat8, _ := sp.Discount.Float64Value()
		if discountFloat8.Valid {
			subtotal = discountFloat8.Float64
		}
		totalFloat8, _ := sp.Total.Float64Value()
		if totalFloat8.Valid {
			subtotal = totalFloat8.Float64
		}
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
		expiresAt = sp.ExpiresAt.Time
	case *sqlc.FindActiveCartByCustomerIDRow:
		id = sp.ID
		customerID = sp.CustomerID
		status = enum.CartStatus(sp.Status)
		currency = stripe.Currency(sp.Currency)
		subtotal = sp.Subtotal
		taxFloat8, _ := sp.Tax.Float64Value()
		if taxFloat8.Valid {
			subtotal = taxFloat8.Float64
		}
		discountFloat8, _ := sp.Discount.Float64Value()
		if discountFloat8.Valid {
			subtotal = discountFloat8.Float64
		}
		totalFloat8, _ := sp.Total.Float64Value()
		if totalFloat8.Valid {
			subtotal = totalFloat8.Float64
		}
		createdAt = sp.CreatedAt.Time
		updatedAt = sp.UpdatedAt.Time
		expiresAt = sp.ExpiresAt.Time
	default:
		return nil
	}

	c.ID = id
	c.CustomerID = customerID
	c.Status = status
	c.Currency = currency
	c.Subtotal = subtotal
	c.Tax = tax
	c.Discount = discount
	c.Total = total
	c.ExpiresAt = expiresAt
	c.CreatedAt = createdAt
	c.UpdatedAt = updatedAt

	return c
}

func (ci *CartItem) ConvertFromSQLCCartItem(sqlcCartItem any) *CartItem {

	var id, cartID, stockID, quantity uint64
	var productID, priceID string
	var subtotal, unitPrice float64

	switch sp := sqlcCartItem.(type) {
	case *sqlc.CartItem:
		id = sp.ID
		cartID = sp.CartID
		stockID = sp.StockID
		quantity = sp.Quantity
		productID = sp.ProductID
		priceID = sp.PriceID
		subtotal = sp.Subtotal
		unitPrice = sp.UnitPrice
	default:
		return nil
	}

	ci.ID = id
	ci.CartID = cartID
	ci.ProductID = productID
	ci.PriceID = priceID
	ci.StockID = stockID
	ci.Quantity = quantity
	ci.UnitPrice = unitPrice
	ci.Subtotal = subtotal

	return ci
}
