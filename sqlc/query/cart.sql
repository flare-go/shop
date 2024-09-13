-- name: CreateCart :exec
INSERT INTO carts (customer_id, status, currency, subtotal, tax, discount, total, expires_at, created_at, updated_at)
VALUES ($1, $2, $3, 0, 0, 0, 0, $4, NOW(), NOW());

-- name: GetCart :one
SELECT id, customer_id, status, currency, subtotal, tax, discount, total, expires_at, created_at, updated_at
FROM carts
WHERE id = $1;

-- name: FindActiveCartByCustomerID :one
SELECT id, customer_id, status, currency, subtotal, tax, discount, total, expires_at, created_at, updated_at
FROM carts
WHERE customer_id = $1 AND status = 'active' LIMIT 1;

-- name: AddCartItem :exec
INSERT INTO cart_items (cart_id, product_id, price_id, stock_id, quantity, unit_price, subtotal, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW());

-- name: ListCartItems :many
SELECT id, cart_id, product_id, price_id, stock_id, quantity, unit_price, subtotal, created_at, updated_at
FROM cart_items
WHERE cart_id = $1;

-- name: GetCartItem :one
SELECT id, cart_id, product_id, price_id, stock_id, quantity, unit_price, subtotal, created_at, updated_at
FROM cart_items
WHERE id = $1;

-- name: FindCartItemByProductID :one
SELECT id, cart_id, product_id, price_id, stock_id, quantity, unit_price, subtotal, created_at, updated_at
FROM cart_items
WHERE cart_id = $1 AND product_id = $2;

-- name: UpdateCartItem :exec
UPDATE cart_items
SET quantity = $2, subtotal = $3, updated_at = NOW()
WHERE id = $1 AND updated_at = $4;


-- name: RemoveCartItem :exec
DELETE FROM cart_items WHERE id = $1;

-- name: ClearCartItems :exec
DELETE FROM cart_items WHERE cart_id = $1;

-- name: UpdateCartTotals :exec
UPDATE carts
SET subtotal = (SELECT COALESCE(SUM(subtotal), 0) FROM cart_items WHERE cart_id = $1),
    total = subtotal + tax - discount,
    updated_at = NOW()
WHERE id = $1 AND updated_at = $2;


-- name: UpdateCartStatus :exec
UPDATE carts
SET status = $2, updated_at = NOW()
WHERE id = $1 AND updated_at = $3;

-- name: UpdateCartItemQuantity :exec
UPDATE cart_items
SET quantity = $2, subtotal = $3, updated_at = NOW()
WHERE id = $1;
