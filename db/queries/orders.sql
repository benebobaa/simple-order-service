-- name: CreateOrder :one
INSERT INTO orders (id, user_id, status, total_price)
VALUES ($1, $2, 'pending', $3)
RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (id, order_id, product_id, sku, quantity, unit_price, subtotal)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetOrderByIDForUser :one
SELECT * FROM orders
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListProductsBySKUsForUpdate :many
SELECT * FROM products
WHERE sku = ANY(sqlc.arg(skus)::text[])
ORDER BY id
FOR UPDATE;

-- name: DecreaseProductStock :execrows
UPDATE products
SET stock = stock - sqlc.arg(quantity), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: ListOrderItemsWithProductNameByOrderIDs :many
SELECT oi.*, p.name AS product_name
FROM order_items oi
JOIN products p ON p.id = oi.product_id
WHERE oi.order_id = ANY(sqlc.arg(order_ids)::uuid[])
ORDER BY oi.created_at, oi.id;

-- name: ListOrdersByUser :many
SELECT * FROM orders
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountOrdersByUser :one
SELECT COUNT(*) FROM orders
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text);

-- name: CancelOrder :one
UPDATE orders
SET status       = 'cancelled',
    cancelled_at = now(),
    updated_at   = now()
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
  AND status = 'pending'
RETURNING *;

-- name: ListOrderItemsByOrderID :many
SELECT * FROM order_items
WHERE order_id = sqlc.arg(order_id)
ORDER BY product_id;

-- name: IncreaseProductStock :execrows
UPDATE products
SET stock = stock + sqlc.arg(quantity), updated_at = now()
WHERE id = sqlc.arg(id);
