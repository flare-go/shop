package enum

// OrderStatus 表示訂單的狀態
type OrderStatus string

const (
	OrderStatusPending           OrderStatus = "pending"            // 訂單已創建，等待付款
	OrderStatusProcessing        OrderStatus = "processing"         // 訂單處理中
	OrderStatusCompleted         OrderStatus = "completed"          // 訂單完成，已發貨或已交付
	OrderStatusCancelled         OrderStatus = "cancelled"          // 訂單取消
	OrderStatusRefunded          OrderStatus = "refunded"           // 訂單退款完成
	OrderStatusPartiallyRefunded OrderStatus = "partially_refunded" // 訂單部分退款完成
	OrderStatusPaid              OrderStatus = "paid"               // 訂單已支付
	OrderStatusFailed            OrderStatus = "failed"             // 訂單支付失敗
	OrderStatusAwaitingStock     OrderStatus = "awaiting_stock"     // 等待庫存補貨
	OrderStatusDispute           OrderStatus = "dispute"            // 訂單爭議
)
