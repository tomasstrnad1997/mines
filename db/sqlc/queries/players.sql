-- name: CreatePlayer :one
INSERT INTO players (username, password_hash, created_at)
VALUES (?, ?, NOW())
RETURNING id, username, password_hash, created_at;

-- name: GetPlayerByUsername :one
SELECT id, username, password_hash, created_at
FROM players
WHERE username = ?;

-- name: GetPlayerByID :one
SELECT id, username, password_hash, created_at
FROM players
WHERE id = ?;
