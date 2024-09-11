-- name: CreateCart :one
INSERT INTO carts (customer_id, currency, status)
VALUES ($1, $2, 'active')
RETURNING id, customer_id, status, currency, subtotal, tax, discount, total, created_at, updated_at, expires_at;

-- name: GetCart :one
SELECT id, customer_id, status, currency, subtotal, tax, discount, total, created_at, updated_at, expires_at
FROM carts
WHERE id = $1 LIMIT 1;

-- name: UpdateCartStatus :one
UPDATE carts
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, status, currency, subtotal, tax, discount, total, created_at, updated_at, expires_at;

-- name: AddCartItem :one
INSERT INTO cart_items (cart_id, product_id, price_id, stock_id, quantity, unit_price, subtotal)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, cart_id, product_id, price_id, stock_id, quantity, unit_price, subtotal, created_at, updated_at;

-- name: RemoveCartItem :exec
DELETE FROM cart_items WHERE id = $1 AND cart_id = $2;

-- name: UpdateCartItemQuantity :exec
UPDATE cart_items
SET quantity = $3, subtotal = $4, updated_at = NOW()
WHERE id = $1 AND cart_id = $2;
