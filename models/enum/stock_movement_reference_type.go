package enum

type StockMovementReferenceType string

const (
	StockMovementReferenceTypeCart       StockMovementReferenceType = "cart"
	StockMovementReferenceTypeOrder      StockMovementReferenceType = "order"
	StockMovementReferenceTypeReturn     StockMovementReferenceType = "return"
	StockMovementReferenceTypeAdjustment StockMovementReferenceType = "adjustment"
)
