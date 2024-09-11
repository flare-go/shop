package enum

type StockMovementType string

const (
	StockMovementTypeIn      StockMovementType = "in"
	StockMovementTypeOut     StockMovementType = "out"
	StockMovementTypeReserve StockMovementType = "reserve"
	StockMovementTypeRelease StockMovementType = "release"
)
