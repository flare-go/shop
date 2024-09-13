-- name: AdjustStock :batchexec
UPDATE stocks
SET reserved_quantity = reserved_quantity + $2, updated_at = NOW()
WHERE id = $1 AND updated_at = $3;

-- name: ReleaseStock :batchexec
UPDATE stocks
SET reserved_quantity = reserved_quantity - $2, updated_at = NOW()
WHERE id = $1 AND updated_at = $3;

-- name: ReduceStock :batchexec
UPDATE stocks
SET quantity = quantity - $2, reserved_quantity = reserved_quantity - $2, updated_at = NOW()
WHERE id = $1 AND updated_at = $3;

-- name: GetStock :one
SELECT id, product_id, quantity, reserved_quantity, location, created_at, updated_at
FROM stocks
WHERE id = $1;

-- name: CreateStockMovement :batchexec
INSERT INTO stock_movements (stock_id, quantity, type, reference_id, reference_type, created_at)
VALUES ($1, $2, $3, $4, $5, NOW());

-- name: ListStockMovements :many
SELECT id, stock_id, quantity, type, reference_id, reference_type, created_at
FROM stock_movements
WHERE stock_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetStockMovementsByReference :many
SELECT id, stock_id, quantity, type, reference_id, reference_type, created_at
FROM stock_movements
WHERE reference_type = $1 AND reference_id = $2
ORDER BY created_at DESC;