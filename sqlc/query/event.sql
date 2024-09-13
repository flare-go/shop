-- name: CreateEvent :exec
INSERT INTO events (
    id, type, processed, created_at, updated_at
) VALUES (
             $1, $2, $3, $4, $5
         );

-- name: GetEventByID :one
SELECT id, type, processed, created_at, updated_at
FROM events
WHERE id = $1;

-- name: MarkEventAsProcessed :exec
UPDATE events
SET processed = true, updated_at = $2
WHERE id = $1;