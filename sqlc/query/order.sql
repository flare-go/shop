-- name: CreateOrder :exec
INSERT INTO orders (customer_id, cart_id, status, currency, subtotal, tax, discount, total, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetOrder :one
SELECT id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, created_at, updated_at
FROM orders
WHERE id = $1;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = $2, updated_at = NOW()
WHERE id = $1 AND updated_at = $3;

-- name: UpdateOrderTotals :exec
UPDATE orders
SET subtotal = $2, tax = $3, discount = $4, total = $5, updated_at = NOW()
WHERE id = $1 AND updated_at = $6;

-- name: ListOrders :many
SELECT id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, created_at, updated_at
FROM orders
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteOrder :exec
DELETE FROM orders WHERE id = $1;

-- name: AddOrderItems :batchexec
INSERT INTO order_items (order_id, product_id, price_id, stock_id, quantity, unit_price, subtotal)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetOrderItem :one
SELECT id, order_id, product_id, price_id, stock_id, quantity, unit_price, subtotal
FROM order_items
WHERE id = $1;

-- name: ListOrderItems :many
SELECT id, order_id, product_id, price_id, stock_id, quantity, unit_price, subtotal
FROM order_items
WHERE order_id = $1;

-- name: UpdateOrderItem :exec
UPDATE order_items
SET quantity = $2, unit_price = $3, subtotal = $4
WHERE id = $1;

-- name: DeleteOrderItem :exec
DELETE FROM order_items WHERE id = $1;

-- name: GetOrderByPaymentIntentID :one
SELECT id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, created_at, updated_at
FROM orders
WHERE payment_intent_id = $1;

-- name: GetOrderByChargeID :one
SELECT id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, created_at, updated_at
FROM orders
WHERE charge_id = $1;

-- name: GetOrderByInvoiceID :one
SELECT id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, created_at, updated_at
FROM orders
WHERE invoice_id = $1;

-- name: UpdateOrderStatusBySubscriptionID :exec
UPDATE orders
SET status = $2, updated_at = NOW()
WHERE subscription_id = $1;

-- name: ListOrdersByStatus :many
SELECT id, customer_id, cart_id, status, currency, subtotal, tax, discount, total, created_at, updated_at
FROM orders
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;