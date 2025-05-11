-- name: InsertGamemodes :exec
INSERT INTO gamemodes (id, name)
VALUES (?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name
