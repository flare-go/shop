-- name: CreateCategory :one
INSERT INTO categories (name, description, parent_id)
VALUES ($1, $2, $3)
RETURNING id, name, description, parent_id, created_at, updated_at;

-- name: GetCategory :one
SELECT id, name, description, parent_id, created_at, updated_at
FROM categories
WHERE id = $1 LIMIT 1;

-- name: UpdateCategory :one
UPDATE categories
SET name = $2, description = $3, parent_id = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, name, description, parent_id, created_at, updated_at;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;

-- name: ListCategories :many
SELECT id, name, description, parent_id, created_at, updated_at
FROM categories
ORDER BY name
LIMIT $1 OFFSET $2;
