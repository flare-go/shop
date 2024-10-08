package shop

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
	"gofalre.io/shop/stock"

	"github.com/nats-io/nats.go"
	"github.com/stripe/stripe-go/v79"
	"go.uber.org/zap"
)

type EventHandler func(context.Context, *stripe.Event) error

type EventManager struct {
	natsConn *nats.Conn
	handlers map[stripe.EventType]EventHandler
	logger   *zap.Logger
}

func NewEventManager(natsConn *nats.Conn, logger *zap.Logger) *EventManager {
	return &EventManager{
		natsConn: natsConn,
		handlers: make(map[stripe.EventType]EventHandler),
		logger:   logger,
	}
}

func (em *EventManager) RegisterHandler(eventType stripe.EventType, handler EventHandler) {
	em.handlers[eventType] = handler
}

func (em *EventManager) GetHandler(eventType stripe.EventType) (EventHandler, bool) {
	handler, exists := em.handlers[eventType]
	return handler, exists
}

func (em *EventManager) SubscribeToEvents(wp *WorkerPool) error {
	if _, err := em.natsConn.Subscribe("payment.service.event.>", func(msg *nats.Msg) {
		var event stripe.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			em.logger.Error("Failed to unmarshal event", zap.Error(err))
			return
		}

		wp.Submit(context.Background(), &event)
	}); err != nil {
		em.logger.Error("Failed to subscribe", zap.Error(err))
	}

	return nil
}

func (s *service) registerEventHandlers() {
	eventHandlers := map[stripe.EventType]EventHandler{
		// Payment Intent Events
		stripe.EventTypePaymentIntentSucceeded:     s.handlePaymentIntentSucceeded,
		stripe.EventTypePaymentIntentPaymentFailed: s.handlePaymentIntentPaymentFailed,
		stripe.EventTypePaymentIntentCanceled:      s.handlePaymentIntentCanceled,

		// Refund Events
		stripe.EventTypeRefundCreated:  s.handleRefundCreated,
		stripe.EventTypeRefundUpdated:  s.handleRefundUpdated,
		stripe.EventTypeChargeRefunded: s.handleChargeRefunded,

		// Dispute Events
		stripe.EventTypeChargeDisputeCreated: s.handleChargeDisputeCreated,

		// Checkout Session Events
		stripe.EventTypeCheckoutSessionCompleted: s.handleCheckoutSessionCompleted,

		// Invoice Events
		stripe.EventTypeInvoicePaymentSucceeded: s.handleInvoicePaymentSucceeded,
		stripe.EventTypeInvoicePaymentFailed:    s.handleInvoicePaymentFailed,

		// Subscription Events
		stripe.EventTypeCustomerSubscriptionCreated: s.handleSubscriptionCreated,
		stripe.EventTypeCustomerSubscriptionUpdated: s.handleSubscriptionUpdated,
		stripe.EventTypeCustomerSubscriptionDeleted: s.handleSubscriptionDeleted}

	for eventType, handler := range eventHandlers {
		s.eventManager.RegisterHandler(eventType, handler)
	}
}

func (s *service) handlePaymentIntentSucceeded(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling PaymentIntent succeeded event", zap.String("event_id", event.ID))

	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		s.logger.Error("Failed to unmarshal PaymentIntent", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 根據 PaymentIntent ID 獲取訂單
		order, err := s.order.GetOrderByPaymentIntentID(ctx, tx, paymentIntent.ID)
		if err != nil {
			s.logger.Error("Order not found for PaymentIntent", zap.String("payment_intent_id", paymentIntent.ID), zap.Error(err))
			return err
		}

		// 更新訂單狀態為已支付
		if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, enum.OrderStatusPaid, order.UpdatedAt); err != nil {
			s.logger.Error("Failed to update order status to 'paid'", zap.Error(err))
			return err
		}

		s.logger.Info("Order status updated to 'paid'", zap.Uint64("order_id", order.ID))

		return err
	})
}

func (s *service) handlePaymentIntentPaymentFailed(ctx context.Context, event *stripe.Event) error {

	s.logger.Info("Handling PaymentIntent payment failed event", zap.String("event_id", event.ID))

	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		s.logger.Error("Failed to unmarshal PaymentIntent", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		orderModel, err := s.order.GetOrderByPaymentIntentID(ctx, tx, paymentIntent.ID)
		if err != nil {
			return fmt.Errorf("獲取訂單失敗: %w", err)
		}

		if err = s.order.UpdateOrderStatus(ctx, tx, orderModel.ID, enum.OrderStatusFailed, orderModel.UpdatedAt); err != nil {
			return fmt.Errorf("更新訂單狀態失敗: %w", err)
		}

		adjustParams := make([]stock.AdjustStockParams, 0, len(orderModel.Items))
		for _, item := range orderModel.Items {
			stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
			if err != nil {
				return fmt.Errorf("failed to get stock for item %s: %w", item.ProductID, err)
			}

			adjustParams = append(adjustParams, stock.AdjustStockParams{
				StockID:     item.StockID,
				Quantity:    item.Quantity,
				LastUpdated: stockModel.UpdatedAt,
			})
		}
		if err = s.stock.AdjustStock(ctx, tx, adjustParams); err != nil {
			return fmt.Errorf("failed to adjust stock: %w", err)
		}

		return err
	})
}

func (s *service) handlePaymentIntentCanceled(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling PaymentIntent canceled event", zap.String("event_id", event.ID))

	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		s.logger.Error("Failed to unmarshal PaymentIntent", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		order, err := s.order.GetOrderByPaymentIntentID(ctx, tx, paymentIntent.ID)
		if err != nil {
			s.logger.Error("Order not found for PaymentIntent", zap.String("payment_intent_id", paymentIntent.ID), zap.Error(err))
			return err
		}

		if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, enum.OrderStatusCancelled, order.UpdatedAt); err != nil {
			s.logger.Error("Failed to update order status to 'cancelled'", zap.Error(err))
			return err
		}

		// 恢復庫存
		adjustParams := make([]stock.AdjustStockParams, 0, len(order.Items))
		for _, item := range order.Items {
			stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
			if err != nil {
				return fmt.Errorf("failed to get stock for item %s: %w", item.ProductID, err)
			}

			adjustParams = append(adjustParams, stock.AdjustStockParams{
				StockID:     item.StockID,
				Quantity:    item.Quantity,
				LastUpdated: stockModel.UpdatedAt,
			})
		}
		if err = s.stock.AdjustStock(ctx, tx, adjustParams); err != nil {
			return fmt.Errorf("failed to adjust stock: %w", err)
		}

		s.logger.Info("Order status updated to 'cancelled' and stock restored", zap.Uint64("order_id", order.ID))
		return err
	})
}

func (s *service) handleRefundCreated(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Refund created event", zap.String("event_id", event.ID))

	var refund stripe.Refund
	if err := json.Unmarshal(event.Data.Raw, &refund); err != nil {
		s.logger.Error("Failed to unmarshal Refund", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 獲取相關訂單
		order, err := s.order.GetOrderByPaymentIntentID(ctx, tx, refund.PaymentIntent.ID)
		if err != nil {
			return fmt.Errorf("failed to get order by payment intent ID: %w", err)
		}

		// 更新訂單狀態
		newStatus := enum.OrderStatusRefundPending
		if refund.Amount == int64(order.Total*100) {
			newStatus = enum.OrderStatusRefunded
		}

		if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, newStatus, order.UpdatedAt); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		s.logger.Info("Refund created processed", zap.String("refund_id", refund.ID))
		return err
	})
}

func (s *service) handleRefundUpdated(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Refund updated event", zap.String("event_id", event.ID))

	var refund stripe.Refund
	if err := json.Unmarshal(event.Data.Raw, &refund); err != nil {
		s.logger.Error("Failed to unmarshal Refund", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 獲取相關訂單
		order, err := s.order.GetOrderByRefundID(ctx, tx, refund.ID)
		if err != nil {
			return fmt.Errorf("failed to get order by refund ID: %w", err)
		}

		// 如果退款狀態變為成功，更新訂單的退款狀態
		if refund.Status == stripe.RefundStatusSucceeded {
			if err := s.order.UpdateOrderStatus(ctx, tx, order.ID, enum.OrderStatusRefunded, order.UpdatedAt); err != nil {
				return fmt.Errorf("failed to update order refund status: %w", err)
			}
		}

		s.logger.Info("Refund updated processed", zap.String("refund_id", refund.ID))
		return err
	})
}

func (s *service) handleChargeRefunded(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Charge refunded event", zap.String("event_id", event.ID))

	var charge stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		s.logger.Error("Failed to unmarshal Charge", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 獲取相關訂單
		order, err := s.order.GetOrderByPaymentIntentID(ctx, tx, charge.PaymentIntent.ID)
		if err != nil {
			return fmt.Errorf("failed to get order by payment intent ID: %w", err)
		}

		// 更新訂單狀態
		newStatus := enum.OrderStatusPartiallyRefunded
		if charge.AmountRefunded == charge.Amount {
			newStatus = enum.OrderStatusRefunded
		}

		if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, newStatus, order.UpdatedAt); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		// 處理庫存
		orderItems, err := s.order.ListOrderItems(ctx, tx, order.ID)
		if err != nil {
			return fmt.Errorf("failed to list order items: %w", err)
		}

		for _, item := range orderItems {
			// 假設全額退款，所有商品都退回庫存
			if err = s.stock.AdjustStock(ctx, tx, []stock.AdjustStockParams{{
				StockID:  item.StockID,
				Quantity: item.Quantity,
			}}); err != nil {
				return fmt.Errorf("failed to adjust stock: %w", err)
			}
		}

		s.logger.Info("Charge refunded processed", zap.String("charge_id", charge.ID))
		return err
	})
}

func (s *service) handleChargeDisputeCreated(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Charge dispute created event", zap.String("event_id", event.ID))

	var dispute stripe.Dispute
	if err := json.Unmarshal(event.Data.Raw, &dispute); err != nil {
		s.logger.Error("Failed to unmarshal Dispute", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 通過 PaymentIntent ID 獲取訂單
		order, err := s.order.GetOrderByRefundID(ctx, tx, dispute.PaymentIntent.ID)
		if err != nil {
			s.logger.Error("Order not found for Charge", zap.String("charge_id", dispute.Charge.ID), zap.Error(err))
			return err
		}

		// 更新訂單狀態為爭議中
		if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, enum.OrderStatusDispute, order.UpdatedAt); err != nil {
			s.logger.Error("Failed to update order status to 'disputed'", zap.Error(err))
			return err
		}

		s.logger.Info("Order status updated to 'disputed'", zap.Uint64("order_id", order.ID))
		return err
	})
}

func (s *service) handleCheckoutSessionCompleted(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Checkout Session completed event", zap.String("event_id", event.ID))

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		s.logger.Error("Failed to unmarshal Checkout Session", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 根據 Session ID 或 PaymentIntent ID 獲取訂單
		order, err := s.order.GetOrderByPaymentIntentID(ctx, tx, session.PaymentIntent.ID)
		if err != nil {
			s.logger.Error("Order not found for PaymentIntent", zap.String("payment_intent_id", session.PaymentIntent.ID), zap.Error(err))
			return err
		}

		// 更新訂單狀態為已支付
		if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, enum.OrderStatusPaid, order.UpdatedAt); err != nil {
			s.logger.Error("Failed to update order status to 'paid'", zap.Error(err))
			return err
		}

		s.logger.Info("Order status updated to 'paid'", zap.Uint64("order_id", order.ID))
		return err
	})
}

func (s *service) handleInvoicePaymentSucceeded(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Invoice payment succeeded event", zap.String("event_id", event.ID))

	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		s.logger.Error("Failed to unmarshal Invoice", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 檢查是否存在相關訂單
		order, err := s.order.GetOrderByInvoiceID(ctx, tx, invoice.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// 如果沒有相關訂單,可能是訂閱付款,創建新訂單
				order = &models.Order{
					CustomerID: invoice.Customer.ID,
					Status:     enum.OrderStatusPaid,
					Total:      float64(invoice.Total) / 100, // 轉換為元
					Currency:   invoice.Currency,
					InvoiceID:  invoice.ID,
				}
				if _, err = s.order.CreateOrder(ctx, tx, order); err != nil {
					return fmt.Errorf("failed to create order for invoice: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get order by invoice ID: %w", err)
			}
		} else {
			// 如果訂單存在,更新狀態
			if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, enum.OrderStatusPaid, order.UpdatedAt); err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}
		}

		s.logger.Info("Invoice payment succeeded processed", zap.String("invoice_id", invoice.ID))
		return nil
	})
}

func (s *service) handleInvoicePaymentFailed(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Invoice payment failed event", zap.String("event_id", event.ID))

	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		s.logger.Error("Failed to unmarshal Invoice", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 檢查是否存在相關訂單
		order, err := s.order.GetOrderByInvoiceID(ctx, tx, invoice.ID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to get order by invoice ID: %w", err)
			}
			// 如果沒有相關訂單,可能是訂閱付款失敗,不需要創建新訂單
		} else {
			// 如果訂單存在,更新狀態
			if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, enum.OrderStatusFailed, order.UpdatedAt); err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}
		}

		s.logger.Info("Invoice payment failed processed", zap.String("invoice_id", invoice.ID))
		return nil
	})
}

func (s *service) handleSubscriptionCreated(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Subscription created event", zap.String("event_id", event.ID))

	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		s.logger.Error("Failed to unmarshal Subscription", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 創建相關的訂單
		order := &models.Order{
			CustomerID:     subscription.Customer.ID,
			Status:         enum.OrderStatusPaid,
			Total:          float64(subscription.Items.Data[0].Price.UnitAmount) / 100,
			Currency:       subscription.Items.Data[0].Price.Currency,
			SubscriptionID: subscription.ID,
		}

		if _, err := s.order.CreateOrder(ctx, tx, order); err != nil {
			return fmt.Errorf("failed to create order for subscription: %w", err)
		}

		return nil
	})
}

func (s *service) handleSubscriptionUpdated(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Subscription updated event", zap.String("event_id", event.ID))

	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		s.logger.Error("Failed to unmarshal Subscription", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 如果訂閱狀態變更為 active，可能需要創建新的訂單
		if subscription.Status == stripe.SubscriptionStatusActive {
			order := &models.Order{
				CustomerID:     subscription.Customer.ID,
				Status:         enum.OrderStatusPaid,
				Total:          float64(subscription.Items.Data[0].Price.UnitAmount) / 100,
				Currency:       subscription.Items.Data[0].Price.Currency,
				SubscriptionID: subscription.ID,
			}

			if _, err := s.order.CreateOrder(ctx, tx, order); err != nil {
				return fmt.Errorf("failed to create order for updated subscription: %w", err)
			}
		}

		return nil
	})

}

func (s *service) handleSubscriptionDeleted(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("Handling Subscription deleted event", zap.String("event_id", event.ID))

	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		s.logger.Error("Failed to unmarshal Subscription", zap.Error(err))
		return err
	}

	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {

		order, err := s.order.GetOrderByCustomerIDAndSubscriptionID(ctx, tx, subscription.Customer.ID, subscription.ID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to get order by customer ID: %w", err)
			}
			s.logger.Error("Failed to get order by customer ID", zap.Error(err))
			return err
		}

		if err = s.order.UpdateOrderStatus(ctx, tx, order.ID, enum.OrderStatusCancelled, order.UpdatedAt); err != nil {
			return fmt.Errorf("failed to update orders for cancelled subscription: %w", err)
		}

		return err
	})
}

func (s *service) ProcessEvent(ctx context.Context, event *stripe.Event) error {

	if _, err := s.event.GetByID(ctx, event.ID); err == nil {
		s.logger.Info("Event already processed", zap.String("event_id", event.ID))
		return nil
	}

	handler, exists := s.eventManager.GetHandler(event.Type)
	if !exists {
		return fmt.Errorf("no handler registered for event type: %s", event.Type)
	}

	if err := s.event.Create(ctx, &models.Event{
		ID:        event.ID,
		Type:      event.Type,
		Processed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		s.logger.Error("Failed to create event", zap.Error(err))
		return err
	}

	if err := handler(ctx, event); err != nil {
		s.logger.Error("處理事件時出錯",
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.Type)),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Stripe event processed", zap.String("event_id", event.ID))

	return nil
}
