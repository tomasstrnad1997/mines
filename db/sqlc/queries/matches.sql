
-- name: CreateMatch :one
INSERT INTO matches (gamemode_id, created_at)
VALUES (?, date('now'))
RETURNING id, created_at;
