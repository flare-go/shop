package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stripe/stripe-go/v79"
	"gofalre.io/shop/models/enum"
	"time"
)

// Order 代表訂單
type Order struct {
	ID              uint64           `json:"id"`
	CustomerID      string           `json:"customer_id"`
	CartID          *uint64          `json:"cart_id,omitempty"`
	Status          enum.OrderStatus `json:"status"`
	Currency        stripe.Currency  `json:"currency"`
	Subtotal        float64          `json:"subtotal"`
	Tax             float64          `json:"tax"`
	Discount        float64          `json:"discount"`
	Total           float64          `json:"total"`
	PaymentIntentID string           `json:"payment_intent_id"`
	SubscriptionID  string           `json:"subscription_id"`
	InvoiceID       string           `json:"invoice_id"`
	ShippingAddress json.RawMessage  `json:"shipping_address"`
	BillingAddress  json.RawMessage  `json:"billing_address"`
	Items           []*OrderItem     `json:"items"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// OrderItem 代表訂單中的單個商品項目
type OrderItem struct {
	ID        uint64  `json:"id"`
	OrderID   uint64  `json:"order_id"`
	ProductID string  `json:"product_id"`
	PriceID   string  `json:"price_id"`
	StockID   uint64  `json:"stock_id"`
	Quantity  uint64  `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
}

var AllowedTransitions = map[enum.OrderStatus][]enum.OrderStatus{
	enum.OrderStatusPending: {
		enum.OrderStatusPaid,
		enum.OrderStatusCancelled,
		enum.OrderStatusFailed,
	},
	enum.OrderStatusPaid: {
		enum.OrderStatusCompleted,
		enum.OrderStatusRefunded,
		enum.OrderStatusPartiallyRefunded,
		enum.OrderStatusDispute,
	},
	enum.OrderStatusFailed: {
		enum.OrderStatusPending, // 可能重試支付
	},
	enum.OrderStatusCancelled: {}, // 終止狀態
	enum.OrderStatusRefunded:  {}, // 終止狀態
	enum.OrderStatusPartiallyRefunded: {
		enum.OrderStatusRefunded,
	},
	enum.OrderStatusDispute: {
		enum.OrderStatusPaid,
		enum.OrderStatusRefunded,
	},
	enum.OrderStatusCompleted: {}, // 終止狀態
}

func (o *Order) AllowChangeStatus(newStatus enum.OrderStatus) bool {
	allowed, exists := AllowedTransitions[o.Status]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}
	return false
}

func (o *Order) CanCancel() bool {
	switch o.Status {
	case enum.OrderStatusPending:
		return true
	case enum.OrderStatusProcessing:
		// 可以添加額外的邏輯，例如檢查訂單創建時間是否在特定時間範圍內
		return time.Since(o.CreatedAt) <= 24*time.Hour
	default:
		return false
	}
}

func (o *Order) Validate() error {
	if o.CustomerID == "" {
		return errors.New("customer ID is required")
	}
	if o.Currency == "" {
		return errors.New("currency is required")
	}
	if len(o.Items) == 0 {
		return errors.New("order must have at least one item")
	}
	if o.Total <= 0 {
		return errors.New("total must be greater than zero")
	}
	if o.Subtotal <= 0 {
		return errors.New("subtotal must be greater than zero")
	}
	if o.Tax < 0 {
		return errors.New("tax cannot be negative")
	}
	if o.Discount < 0 {
		return errors.New("discount cannot be negative")
	}
	if o.Total != o.Subtotal+o.Tax-o.Discount {
		return errors.New("total does not match subtotal, tax, and discount")
	}

	// 驗證每個訂單項
	for _, item := range o.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("invalid order item: %w", err)
		}
	}

	return nil
}

func (oi *OrderItem) Validate() error {
	if oi.ProductID == "" {
		return errors.New("product ID is required")
	}
	if oi.Quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}
	if oi.UnitPrice <= 0 {
		return errors.New("unit price must be greater than zero")
	}
	if oi.Subtotal != float64(oi.Quantity)*oi.UnitPrice {
		return errors.New("subtotal does not match quantity and unit price")
	}
	return nil
}

func NewOrder() *Order {
	return new(Order)
}
