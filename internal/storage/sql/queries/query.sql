-- name: CreateUser :one
INSERT INTO doodle.users (user_id, name)
VALUES ($1, $2)
RETURNING user_id, name, created_at;

