package models

import (
	"gofalre.io/shop/models/enum"
	"time"
)

// Cart 代表購物車
type Cart struct {
	ID         int             `json:"id"`
	CustomerID string          `json:"customer_id"`
	Status     enum.CartStatus `json:"status"`
	Currency   string          `json:"currency"`
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
	ID        int     `json:"id"`
	CartID    int     `json:"cart_id"`
	ProductID string  `json:"product_id"`
	PriceID   string  `json:"price_id"`
	StockID   int     `json:"stock_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
}
