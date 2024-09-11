-- name: CreateStock :one
INSERT INTO stocks (product_id, quantity, reserved_quantity, location)
VALUES ($1, $2, $3, $4)
RETURNING id, product_id, quantity, reserved_quantity, location, created_at, updated_at;

-- name: GetStock :one
SELECT id, product_id, quantity, reserved_quantity, location, created_at, updated_at
FROM stocks
WHERE id = $1 LIMIT 1;

-- name: UpdateStock :one
UPDATE stocks
SET quantity = $2, reserved_quantity = $3, location = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, product_id, quantity, reserved_quantity, location, created_at, updated_at;

-- name: ListStocks :many
SELECT id, product_id, quantity, reserved_quantity, location, created_at, updated_at
FROM stocks
WHERE product_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateStockMovement :one
INSERT INTO stock_movements (stock_id, quantity, type, reference_type, reference_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, stock_id, quantity, type, reference_type, reference_id, created_at;