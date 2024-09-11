-- name: CreateOrder :one
INSERT INTO orders (customer_id, cart_id, status, currency, subtotal, tax, discount, total, payment_intent_id, shipping_address, billing_address)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, payment_intent_id, shipping_address, billing_address, created_at, updated_at;

-- name: GetOrder :one
SELECT id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, payment_intent_id, shipping_address, billing_address, created_at, updated_at
FROM orders
WHERE id = $1 LIMIT 1;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, payment_intent_id, shipping_address, billing_address, created_at, updated_at;

-- name: ListOrders :many
SELECT id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, payment_intent_id, shipping_address, billing_address, created_at, updated_at
FROM orders
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;