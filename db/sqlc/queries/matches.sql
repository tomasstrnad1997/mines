
-- name: CreateMatch :one
INSERT INTO matches (created_at)
VALUES (NOW())
RETURNING id, created_at;
