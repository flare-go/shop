-- name: CreateCategory :exec
INSERT INTO categories (name, description, parent_id, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW());

-- name: GetCategoryByID :one
SELECT id, name, description, parent_id, created_at, updated_at
FROM categories
WHERE id = $1;

-- name: UpdateCategory :exec
UPDATE categories
SET name = $2, description = $3, parent_id = $4, updated_at = NOW()
WHERE id = $1 AND updated_at = $5;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;

-- name: ListCategories :many
SELECT id, name, description, parent_id, created_at, updated_at
FROM categories
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListSubcategories :many
SELECT id, name, description, parent_id, created_at, updated_at
FROM categories
WHERE parent_id = $1
ORDER BY created_at DESC;

-- name: AssignProductToCategory :exec
INSERT INTO product_categories (product_id, category_id)
VALUES ($1, $2)
ON CONFLICT (product_id, category_id) DO NOTHING;

-- name: RemoveProductFromCategory :exec
DELETE FROM product_categories
WHERE product_id = $1 AND category_id = $2;