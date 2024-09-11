package models

import (
	"encoding/json"
	"gofalre.io/shop/models/enum"
	"time"
)

// Order 代表訂單
type Order struct {
	ID              int              `json:"id"`
	CustomerID      string           `json:"customer_id"`
	CartID          *int             `json:"cart_id,omitempty"`
	Status          enum.OrderStatus `json:"status"`
	Currency        string           `json:"currency"`
	Subtotal        float64          `json:"subtotal"`
	Tax             float64          `json:"tax"`
	Discount        float64          `json:"discount"`
	Total           float64          `json:"total"`
	PaymentIntentID string           `json:"payment_intent_id"`
	ShippingAddress json.RawMessage  `json:"shipping_address"`
	BillingAddress  json.RawMessage  `json:"billing_address"`
	Items           []OrderItem      `json:"items"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// OrderItem 代表訂單中的單個商品項目
type OrderItem struct {
	ID        int     `json:"id"`
	OrderID   int     `json:"order_id"`
	ProductID string  `json:"product_id"`
	PriceID   string  `json:"price_id"`
	StockID   int     `json:"stock_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
}
