-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: ResetUsers :exec
DELETE FROM users;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: LoginUser :one
SELECT id, created_at, updated_at, email, hashed_password FROM users
WHERE email = $1;