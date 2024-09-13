package shop

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go"
	"github.com/stripe/stripe-go/v79"
	"go.uber.org/zap"

	"gofalre.io/shop/cart"
	"gofalre.io/shop/category"
	"gofalre.io/shop/driver"
	"gofalre.io/shop/event"
	"gofalre.io/shop/models"
	"gofalre.io/shop/models/enum"
	"gofalre.io/shop/order"
	"gofalre.io/shop/stock"
)

type Service interface {
	CreateCart(ctx context.Context, customerID string, currency stripe.Currency) (*models.Cart, error)
	GetOrCreateActiveCart(ctx context.Context, customerID string, currency stripe.Currency) (*models.Cart, error)
	AddItemsToCart(ctx context.Context, customerID string, cartID uint64, items []*models.CartItem, currency stripe.Currency) error
	RemoveItemFromCart(ctx context.Context, cartID, itemID uint64) error
	UpdateCartItemQuantity(ctx context.Context, cartID, itemID, quantity uint64) error

	ConvertCartToOrder(ctx context.Context, cartID uint64) (*models.Order, error)
	CreateOrder(ctx context.Context, order *models.Order) error
	GetOrder(ctx context.Context, orderID uint64) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID uint64, status enum.OrderStatus) error
	ListOrders(ctx context.Context, customerID string, limit, offset uint64) ([]*models.Order, error)
	CancelOrder(ctx context.Context, orderID uint64) error

	CreateCategory(ctx context.Context, category *models.Category) error
	GetCategoryByID(ctx context.Context, id uint64) (*models.Category, error)
	UpdateCategory(ctx context.Context, category *models.Category) error
	DeleteCategory(ctx context.Context, id uint64) error
	ListCategory(ctx context.Context, limit, offset uint64) ([]*models.Category, error)
	ListSubcategories(ctx context.Context, parentID uint64) ([]*models.Category, error)
	GetCategoryTree(ctx context.Context) ([]*models.CategoryTree, error)
	AssignProductToCategory(ctx context.Context, productID string, categoryID uint64) error
	RemoveProductFromCategory(ctx context.Context, productID string, categoryID uint64) error
}

type service struct {
	category category.Repository
	cart     cart.Repository
	order    order.Repository
	event    event.Repository
	stock    stock.Repository

	transactionManager *driver.TransactionManager
	eventManager       *EventManager
	workerPool         *WorkerPool

	natsConn *nats.Conn
	logger   *zap.Logger
}

func NewService(
	category category.Repository, cart cart.Repository, order order.Repository, stock stock.Repository, tm *driver.TransactionManager,
	natsConn *nats.Conn,
	logger *zap.Logger) Service {
	s := &service{
		category:           category,
		cart:               cart,
		order:              order,
		stock:              stock,
		transactionManager: tm,
		logger:             logger,
	}
	s.eventManager = NewEventManager(natsConn, logger)
	s.workerPool = NewWorkerPool(10, s, logger)
	s.registerEventHandlers()

	// 訂閱事件
	if err := s.eventManager.SubscribeToEvents(s.workerPool); err != nil {
		logger.Error("Failed to subscribe to events", zap.Error(err))
	}

	return s
}
func (s *service) CreateCart(ctx context.Context, customerID string, currency stripe.Currency) (*models.Cart, error) {

	cartModel := models.NewCart()
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {

		existingCart, err := s.cart.GetActiveCartByCustomerID(ctx, tx, customerID)
		if err == nil {
			cartModel = existingCart
			return nil
		}

		newCart := &models.Cart{
			CustomerID: customerID,
			Currency:   currency,
			Status:     enum.CartStatusActive,
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().AddDate(0, 0, 7),
		}

		if err = s.cart.CreateCart(ctx, tx, newCart); err != nil {
			return err
		}
		cartModel = newCart

		return nil
	})

	return cartModel, err
}

func (s *service) GetOrCreateActiveCart(ctx context.Context, customerID string, currency stripe.Currency) (*models.Cart, error) {

	cartModel, err := s.cart.GetActiveCartByCustomerID(ctx, nil, customerID)
	if err == nil {
		return cartModel, nil
	}

	newCart := &models.Cart{
		CustomerID: customerID,
		Currency:   currency,
		Status:     enum.CartStatusActive,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 7),
	}

	if err = s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.cart.CreateCart(ctx, tx, newCart)
	}); err != nil {
		return nil, err
	}

	return newCart, nil
}

func (s *service) AddItemsToCart(ctx context.Context, customerID string, cartID uint64, items []*models.CartItem, currency stripe.Currency) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 1. 獲得購物車
		cartModel, err := s.cart.GetCart(ctx, tx, cartID)
		if err != nil {
			return fmt.Errorf("failed to get cart: %w", err)
		}

		// 2. 檢查購物車狀態
		if cartModel.Status != enum.CartStatusActive {
			// 如果購物車狀態不是 active，創建新的購物車
			newCart, err := s.GetOrCreateActiveCart(ctx, customerID, currency)
			if err != nil {
				return fmt.Errorf("failed to create new cart: %w", err)
			}
			cartModel = newCart
			cartID = newCart.ID
		}

		adjustParams := make([]stock.AdjustStockParams, 0, len(items))
		moveParams := make([]stock.CreateStockMovementParams, 0, len(items))

		for _, item := range items {
			// 3. 檢查庫存
			stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
			if err != nil {
				return fmt.Errorf("failed to get stock for item %s: %w", item.ProductID, err)
			}
			if stockModel.Quantity-stockModel.ReservedQuantity < item.Quantity {
				return fmt.Errorf("insufficient stock for item %s", item.ProductID)
			}

			// 4. 檢查是否已存在相同商品
			existingItem, err := s.cart.GetCartItemByProductID(ctx, tx, cartID, item.ProductID)
			if err == nil {
				// 商品已存在，更新數量和小計
				existingItem.Quantity += item.Quantity
				existingItem.Subtotal = float64(existingItem.Quantity) * existingItem.UnitPrice

				if err = s.cart.UpdateCartItem(ctx, tx, existingItem); err != nil {
					return fmt.Errorf("failed to update cart item %s: %w", item.ProductID, err)
				}
			} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to check existing cart item %s: %w", item.ProductID, err)
			} else {
				// 商品不存在，添加新的購物車項目
				if err = s.cart.AddCartItem(ctx, tx, cartID, item); err != nil {
					return fmt.Errorf("failed to add cart item %s: %w", item.ProductID, err)
				}
			}

			// 準備庫存調整參數
			adjustParams = append(adjustParams, stock.AdjustStockParams{
				StockID:     item.StockID,
				Quantity:    int64(item.Quantity),
				LastUpdated: stockModel.UpdatedAt,
			})

			// 準備庫存變動記錄參數
			moveParams = append(moveParams, stock.CreateStockMovementParams{
				StockID:       item.StockID,
				Quantity:      int64(item.Quantity),
				Type:          enum.StockMovementTypeReserve,
				ReferenceID:   cartID,
				ReferenceType: enum.StockMovementReferenceTypeCart,
			})
		}

		// 5. 批量調整庫存
		if err = s.stock.AdjustStock(ctx, tx, adjustParams); err != nil {
			return fmt.Errorf("failed to adjust stock: %w", err)
		}

		// 6. 批量創建庫存變動記錄
		if err = s.stock.CreateStockMovements(ctx, tx, moveParams); err != nil {
			return fmt.Errorf("failed to create stock movements: %w", err)
		}

		return nil
	})
}

func (s *service) RemoveItemFromCart(ctx context.Context, cartID, itemID uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		item, err := s.cart.GetCartItem(ctx, tx, itemID)
		if err != nil {
			return err
		}

		stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
		if err != nil {
			return fmt.Errorf("failed to get stock: %w", err)
		}
		if stockModel.Quantity-stockModel.ReservedQuantity < item.Quantity {
			return errors.New("insufficient stock")
		}

		if err = s.cart.RemoveCartItem(ctx, tx, itemID); err != nil {
			return err
		}

		adjustParams := []stock.AdjustStockParams{
			{
				StockID:     item.StockID,
				Quantity:    int64(item.Quantity),
				LastUpdated: stockModel.UpdatedAt,
			},
		}
		if err = s.stock.AdjustStock(ctx, tx, adjustParams); err != nil {
			return fmt.Errorf("failed to adjust stock: %w", err)
		}

		moveParams := []stock.CreateStockMovementParams{
			{
				StockID:       item.StockID,
				Quantity:      int64(item.Quantity),
				Type:          enum.StockMovementTypeReserve,
				ReferenceID:   cartID,
				ReferenceType: enum.StockMovementReferenceTypeCart,
			},
		}
		if err = s.stock.CreateStockMovements(ctx, tx, moveParams); err != nil {
			return fmt.Errorf("failed to create stock movement: %w", err)
		}

		return nil
	})
}

func (s *service) ClearCart(ctx context.Context, cartID uint64, status enum.CartStatus) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 1. 獲取購物車
		if _, err := s.cart.GetCart(ctx, tx, cartID); err != nil {
			return fmt.Errorf("failed to get cart: %w", err)
		}

		// 2. 獲取購物車項目
		items, err := s.cart.ListCartItems(ctx, tx, cartID)
		if err != nil {
			return fmt.Errorf("failed to list cart items: %w", err)
		}

		if len(items) > 0 {
			// 3. 準備庫存釋放參數
			releaseParams := make([]stock.ReleaseStockParams, len(items))
			moveParams := make([]stock.CreateStockMovementParams, len(items))

			for i, item := range items {
				stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
				if err != nil {
					return fmt.Errorf("failed to get stock for item %s: %w", item.ProductID, err)
				}

				releaseParams[i] = stock.ReleaseStockParams{
					StockID:     item.StockID,
					Quantity:    int64(item.Quantity),
					LastUpdated: stockModel.UpdatedAt,
				}

				moveParams[i] = stock.CreateStockMovementParams{
					StockID:       item.StockID,
					Quantity:      int64(item.Quantity),
					Type:          enum.StockMovementTypeRelease,
					ReferenceID:   cartID,
					ReferenceType: enum.StockMovementReferenceTypeCart,
				}
			}

			// 4. 批量釋放庫存
			if err = s.stock.ReleaseStock(ctx, tx, releaseParams); err != nil {
				return fmt.Errorf("failed to release stock: %w", err)
			}

			// 5. 批量創建庫存變動記錄
			if err = s.stock.CreateStockMovements(ctx, tx, moveParams); err != nil {
				return fmt.Errorf("failed to create stock movements: %w", err)
			}
		}

		// 6. 清空購物車項目
		if err = s.cart.ClearCartItems(ctx, tx, cartID); err != nil {
			return fmt.Errorf("failed to clear cart items: %w", err)
		}

		// 7. 更新購物車狀態
		if err = s.cart.UpdateCartStatus(ctx, tx, cartID, status); err != nil {
			return fmt.Errorf("failed to update cart status: %w", err)
		}

		return nil
	})
}

func (s *service) UpdateCartItemQuantity(ctx context.Context, cartID, itemID, newQuantity uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 1. 獲取購物車項目
		item, err := s.cart.GetCartItem(ctx, tx, itemID)
		if err != nil {
			return fmt.Errorf("failed to get cart item: %w", err)
		}

		if item.CartID != cartID {
			return fmt.Errorf("cart item does not belong to the specified cart")
		}

		// 2. 計算數量差異
		quantityDiff := int64(newQuantity) - int64(item.Quantity)

		// 3. 獲取庫存信息
		stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
		if err != nil {
			return fmt.Errorf("failed to get stock: %w", err)
		}

		// 4. 檢查庫存是否足夠（如果是增加數量）
		if quantityDiff > 0 && stockModel.Quantity-stockModel.ReservedQuantity < uint64(quantityDiff) {
			return fmt.Errorf("insufficient stock")
		}

		// 5. 更新購物車項目
		item.Quantity = newQuantity
		item.Subtotal = float64(newQuantity) * item.UnitPrice

		if err = s.cart.UpdateCartItem(ctx, tx, item); err != nil {
			return fmt.Errorf("failed to update cart item: %w", err)
		}

		// 6. 調整庫存
		var adjustParams []stock.AdjustStockParams
		var moveParams []stock.CreateStockMovementParams

		if quantityDiff > 0 {
			adjustParams = []stock.AdjustStockParams{
				{
					StockID:     item.StockID,
					Quantity:    quantityDiff,
					LastUpdated: stockModel.UpdatedAt,
				},
			}
			moveParams = []stock.CreateStockMovementParams{
				{
					StockID:       item.StockID,
					Quantity:      quantityDiff,
					Type:          enum.StockMovementTypeReserve,
					ReferenceID:   cartID,
					ReferenceType: enum.StockMovementReferenceTypeCart,
				},
			}
			if err = s.stock.AdjustStock(ctx, tx, adjustParams); err != nil {
				return fmt.Errorf("failed to adjust stock: %w", err)
			}
		} else if quantityDiff < 0 {
			releaseParams := []stock.ReleaseStockParams{
				{
					StockID:     item.StockID,
					Quantity:    -quantityDiff,
					LastUpdated: stockModel.UpdatedAt,
				},
			}
			moveParams = []stock.CreateStockMovementParams{
				{
					StockID:       item.StockID,
					Quantity:      -quantityDiff,
					Type:          enum.StockMovementTypeRelease,
					ReferenceID:   cartID,
					ReferenceType: enum.StockMovementReferenceTypeCart,
				},
			}
			if err = s.stock.ReleaseStock(ctx, tx, releaseParams); err != nil {
				return fmt.Errorf("failed to release stock: %w", err)
			}
		}

		// 7. 創建庫存變動記錄（如果數量有變化）
		if quantityDiff != 0 {
			if err = s.stock.CreateStockMovements(ctx, tx, moveParams); err != nil {
				return fmt.Errorf("failed to create stock movement: %w", err)
			}
		}

		return nil
	})
}

// ConvertCartToOrder 這個功能將會從購物車生成訂單，並且扣減庫存
func (s *service) ConvertCartToOrder(ctx context.Context, cartID uint64) (*models.Order, error) {
	var newOrder *models.Order

	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		var err error

		// 1. 獲取購物車
		cartModel, err := s.cart.GetCart(ctx, tx, cartID)
		if err != nil {
			return fmt.Errorf("failed to get cart: %w", err)
		}

		if cartModel.Status != enum.CartStatusActive {
			return fmt.Errorf("cart is not active")
		}

		// 2. 獲取購物車項目
		cartItems, err := s.cart.ListCartItems(ctx, tx, cartID)
		if err != nil {
			return fmt.Errorf("failed to list cart items: %w", err)
		}

		if len(cartItems) == 0 {
			return fmt.Errorf("cart is empty")
		}

		// 3. 創建訂單
		newOrder = &models.Order{
			CustomerID: cartModel.CustomerID,
			CartID:     &cartID,
			Status:     enum.OrderStatusPending,
			Currency:   cartModel.Currency,
			Subtotal:   cartModel.Subtotal,
			Tax:        cartModel.Tax,
			Discount:   cartModel.Discount,
			Total:      cartModel.Total,
		}

		if err = s.order.CreateOrder(ctx, tx, newOrder); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		// 4. 創建訂單項目並調整庫存
		orderItems := make([]*models.OrderItem, len(cartItems))
		reduceStockParams := make([]stock.ReduceStockParams, len(cartItems))
		stockMoveParams := make([]stock.CreateStockMovementParams, len(cartItems))

		for i, item := range cartItems {
			orderItems[i] = &models.OrderItem{
				OrderID:   newOrder.ID,
				ProductID: item.ProductID,
				PriceID:   item.PriceID,
				StockID:   item.StockID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				Subtotal:  item.Subtotal,
			}

			stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
			if err != nil {
				return fmt.Errorf("failed to get stock for item %s: %w", item.ProductID, err)
			}

			reduceStockParams[i] = stock.ReduceStockParams{
				StockID:     item.StockID,
				Quantity:    int64(item.Quantity),
				LastUpdated: stockModel.UpdatedAt,
			}

			stockMoveParams[i] = stock.CreateStockMovementParams{
				StockID:       item.StockID,
				Quantity:      int64(item.Quantity),
				Type:          enum.StockMovementTypeOut,
				ReferenceID:   newOrder.ID,
				ReferenceType: enum.StockMovementReferenceTypeOrder,
			}
		}

		// 5. 批量創建訂單項目
		if err = s.order.AddOrderItems(ctx, tx, orderItems); err != nil {
			return fmt.Errorf("failed to add order items: %w", err)
		}

		// 6. 批量減少庫存
		if err = s.stock.ReduceStock(ctx, tx, reduceStockParams); err != nil {
			return fmt.Errorf("failed to reduce stock: %w", err)
		}

		// 7. 批量創建庫存變動記錄
		if err = s.stock.CreateStockMovements(ctx, tx, stockMoveParams); err != nil {
			return fmt.Errorf("failed to create stock movements: %w", err)
		}

		// 8. 更新購物車狀態
		if err = s.cart.UpdateCartStatus(ctx, tx, cartID, enum.CartStatusConverted); err != nil {
			return fmt.Errorf("failed to update cart status: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newOrder, nil
}

// CreateOrder 手動創建訂單，這可能適用於後台或特殊業務需求
func (s *service) CreateOrder(ctx context.Context, order *models.Order) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 1. 驗證訂單數據
		if err := order.Validate(); err != nil {
			return fmt.Errorf("invalid order data: %w", err)
		}

		// 2. 創建訂單
		if err := s.order.CreateOrder(ctx, tx, order); err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		// 3. 準備訂單項目、庫存調整和庫存變動記錄的參數
		orderItems := make([]*models.OrderItem, len(order.Items))
		reduceStockParams := make([]stock.ReduceStockParams, len(order.Items))
		stockMoveParams := make([]stock.CreateStockMovementParams, len(order.Items))

		for i, item := range order.Items {
			// 設置訂單項目
			orderItems[i] = &models.OrderItem{
				OrderID:   order.ID,
				ProductID: item.ProductID,
				PriceID:   item.PriceID,
				StockID:   item.StockID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				Subtotal:  item.Subtotal,
			}

			// 獲取當前庫存信息
			stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
			if err != nil {
				return fmt.Errorf("failed to get stock for item %s: %w", item.ProductID, err)
			}

			// 檢查庫存是否足夠
			if stockModel.Quantity < item.Quantity {
				return fmt.Errorf("insufficient stock for product %s: available %d, required %d", item.ProductID, stockModel.Quantity, item.Quantity)
			}

			// 準備庫存調整參數
			reduceStockParams[i] = stock.ReduceStockParams{
				StockID:     item.StockID,
				Quantity:    int64(item.Quantity),
				LastUpdated: stockModel.UpdatedAt,
			}

			// 準備庫存變動記錄參數
			stockMoveParams[i] = stock.CreateStockMovementParams{
				StockID:       item.StockID,
				Quantity:      int64(item.Quantity),
				Type:          enum.StockMovementTypeOut,
				ReferenceID:   order.ID,
				ReferenceType: enum.StockMovementReferenceTypeOrder,
			}
		}

		// 4. 批量創建訂單項目
		if err := s.order.AddOrderItems(ctx, tx, orderItems); err != nil {
			return fmt.Errorf("failed to add order items: %w", err)
		}

		// 5. 批量減少庫存
		if err := s.stock.ReduceStock(ctx, tx, reduceStockParams); err != nil {
			return fmt.Errorf("failed to reduce stock: %w", err)
		}

		// 6. 批量創建庫存變動記錄
		if err := s.stock.CreateStockMovements(ctx, tx, stockMoveParams); err != nil {
			return fmt.Errorf("failed to create stock movements: %w", err)
		}

		// 7. 更新訂單總計
		if err := s.order.UpdateOrderTotals(ctx, tx, order.ID); err != nil {
			return fmt.Errorf("failed to update order totals: %w", err)
		}

		return nil
	})
}

// GetOrder 根據 orderID 獲取訂單的詳細信息，包括所有訂單項
func (s *service) GetOrder(ctx context.Context, orderID uint64) (*models.Order, error) {

	orderModel, err := s.order.GetOrder(ctx, nil, orderID)
	if err != nil {
		return nil, fmt.Errorf("獲取訂單失敗: %w", err)
	}

	items, err := s.order.ListOrderItems(ctx, nil, orderID)
	if err != nil {
		return nil, fmt.Errorf("獲取訂單項目失敗: %w", err)
	}

	orderModel.Items = items
	return orderModel, nil
}

// UpdateOrderStatus 用於更新訂單狀態，如 pending、paid、cancelled、completed 等
func (s *service) UpdateOrderStatus(ctx context.Context, orderID uint64, newStatus enum.OrderStatus) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 1. 獲取訂單
		orderModel, err := s.order.GetOrder(ctx, tx, orderID)
		if err != nil {
			return fmt.Errorf("failed to get order: %w", err)
		}

		// 2. 檢查狀態轉換是否有效
		if !orderModel.AllowChangeStatus(newStatus) {
			return fmt.Errorf("invalid status transition from %s to %s", orderModel.Status, newStatus)
		}

		// 3. 更新訂單狀態
		if err = s.order.UpdateOrderStatus(ctx, tx, orderID, newStatus); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		// 4. 處理特定狀態轉換的邏輯
		switch newStatus {
		case enum.OrderStatusCancelled, enum.OrderStatusRefunded:
			// 獲取訂單項目
			items, err := s.order.ListOrderItems(ctx, tx, orderID)
			if err != nil {
				return fmt.Errorf("failed to list order items: %w", err)
			}

			// 準備庫存調整參數
			adjustParams := make([]stock.AdjustStockParams, len(items))
			moveParams := make([]stock.CreateStockMovementParams, len(items))

			for i, item := range items {
				stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
				if err != nil {
					return fmt.Errorf("failed to get stock for item %s: %w", item.ProductID, err)
				}

				adjustParams[i] = stock.AdjustStockParams{
					StockID:     item.StockID,
					Quantity:    int64(item.Quantity),
					LastUpdated: stockModel.UpdatedAt,
				}

				moveParams[i] = stock.CreateStockMovementParams{
					StockID:       item.StockID,
					Quantity:      int64(item.Quantity),
					Type:          enum.StockMovementTypeIn,
					ReferenceID:   orderID,
					ReferenceType: enum.StockMovementReferenceTypeOrder,
				}
			}

			// 批量調整庫存
			if err = s.stock.AdjustStock(ctx, tx, adjustParams); err != nil {
				return fmt.Errorf("failed to adjust stock: %w", err)
			}

			// 批量創建庫存變動記錄
			if err = s.stock.CreateStockMovements(ctx, tx, moveParams); err != nil {
				return fmt.Errorf("failed to create stock movements: %w", err)
			}
		}

		return nil
	})
}

// ListOrders 列出指定客戶的訂單
func (s *service) ListOrders(ctx context.Context, customerID string, limit, offset uint64) ([]*models.Order, error) {
	orders, err := s.order.ListOrders(ctx, nil, customerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("列出訂單失敗: %w", err)
	}
	return orders, nil
}

// DeleteOrder 刪除訂單，這適用於測試或後台操作
func (s *service) DeleteOrder(ctx context.Context, orderID uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 刪除訂單
		return s.order.DeleteOrder(ctx, tx, orderID)
	})
}

// CancelOrder 取消訂單
func (s *service) CancelOrder(ctx context.Context, orderID uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		// 1. 獲取訂單
		orderModel, err := s.order.GetOrder(ctx, tx, orderID)
		if err != nil {
			return fmt.Errorf("failed to get order: %w", err)
		}

		// 2. 檢查訂單是否可以取消
		if !orderModel.CanCancel() {
			return fmt.Errorf("order cannot be cancelled: current status is %s", orderModel.Status)
		}

		// 3. 更新訂單狀態
		if err = s.order.UpdateOrderStatus(ctx, tx, orderID, enum.OrderStatusCancelled); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		// 4. 獲取訂單項目
		items, err := s.order.ListOrderItems(ctx, tx, orderID)
		if err != nil {
			return fmt.Errorf("failed to list order items: %w", err)
		}

		// 5. 準備庫存調整參數
		adjustParams := make([]stock.AdjustStockParams, len(items))
		moveParams := make([]stock.CreateStockMovementParams, len(items))

		for i, item := range items {
			stockModel, err := s.stock.GetStock(ctx, tx, item.StockID)
			if err != nil {
				return fmt.Errorf("failed to get stock for item %s: %w", item.ProductID, err)
			}

			adjustParams[i] = stock.AdjustStockParams{
				StockID:     item.StockID,
				Quantity:    int64(item.Quantity),
				LastUpdated: stockModel.UpdatedAt,
			}

			moveParams[i] = stock.CreateStockMovementParams{
				StockID:       item.StockID,
				Quantity:      int64(item.Quantity),
				Type:          enum.StockMovementTypeIn,
				ReferenceID:   orderID,
				ReferenceType: enum.StockMovementReferenceTypeOrder,
			}
		}

		// 6. 批量調整庫存
		if err = s.stock.AdjustStock(ctx, tx, adjustParams); err != nil {
			return fmt.Errorf("failed to adjust stock: %w", err)
		}

		// 7. 批量創建庫存變動記錄
		if err = s.stock.CreateStockMovements(ctx, tx, moveParams); err != nil {
			return fmt.Errorf("failed to create stock movements: %w", err)
		}

		return nil
	})
}

func (s *service) CreateCategory(ctx context.Context, category *models.Category) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.category.Create(ctx, tx, category)
	})
}

func (s *service) GetCategoryByID(ctx context.Context, id uint64) (*models.Category, error) {

	return s.category.GetByID(ctx, nil, id)
}

func (s *service) UpdateCategory(ctx context.Context, category *models.Category) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.category.Update(ctx, tx, category)
	})
}

func (s *service) DeleteCategory(ctx context.Context, id uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.category.Delete(ctx, tx, id)
	})
}

func (s *service) ListCategory(ctx context.Context, limit, offset uint64) ([]*models.Category, error) {
	return s.category.List(ctx, nil, limit, offset)
}

func (s *service) ListSubcategories(ctx context.Context, parentID uint64) ([]*models.Category, error) {
	return s.category.ListSubcategories(ctx, nil, parentID)
}

func (s *service) GetCategoryTree(ctx context.Context) ([]*models.CategoryTree, error) {
	var categoryTree []*models.CategoryTree
	err := s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		categories, err := s.category.List(ctx, tx, 0, 0) // Get all categories
		if err != nil {
			return err
		}
		categoryTree = buildCategoryTree(categories)
		return nil
	})
	return categoryTree, err
}

func (s *service) AssignProductToCategory(ctx context.Context, productID string, categoryID uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.category.AssignProductToCategory(ctx, tx, productID, categoryID)
	})
}

func (s *service) RemoveProductFromCategory(ctx context.Context, productID string, categoryID uint64) error {
	return s.transactionManager.ExecuteTransaction(ctx, func(tx pgx.Tx) error {
		return s.category.RemoveProductFromCategory(ctx, tx, productID, categoryID)
	})
}

func buildCategoryTree(categories []*models.Category) []*models.CategoryTree {
	categoryMap := make(map[uint64]*models.CategoryTree)
	var roots []*models.CategoryTree

	for _, cat := range categories {
		node := &models.CategoryTree{Category: cat}
		categoryMap[cat.ID] = node
		if cat.ParentID == nil {
			roots = append(roots, node)
		}
	}

	for _, cat := range categories {
		if cat.ParentID != nil {
			parent, exists := categoryMap[*cat.ParentID]
			if exists {
				parent.Children = append(parent.Children, categoryMap[cat.ID])
			}
		}
	}

	return roots
}
