
-- name: CreateMatch :one
INSERT INTO matches (created_at)
VALUES (date('now'))
RETURNING id, created_at;
