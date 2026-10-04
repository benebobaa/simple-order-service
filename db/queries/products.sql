-- name: CreateProduct :one
INSERT INTO products (id, sku, name, description, price, stock)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetProductByID :one
SELECT * FROM products WHERE id = $1;

-- name: GetProductBySKU :one
SELECT * FROM products WHERE sku = $1;

-- name: ListProducts :many
SELECT * FROM products
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: CountProducts :one
SELECT COUNT(*) FROM products;

-- name: UpdateProduct :one
UPDATE products
SET name        = $2,
    description = $3,
    price       = $4,
    stock       = $5,
    updated_at  = now()
WHERE id = $1
RETURNING *;
