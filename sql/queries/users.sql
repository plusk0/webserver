-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, password, is_chirpy_red)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2,
    false
)
RETURNING *;

-- name: GetUsers :many
SELECT * FROM users;

-- name: GetUser :one
SELECT * FROM users WHERE $1 = email;

-- name: UpdateUser :one
UPDATE users SET 
email = $2,
password = $3,
updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, email, is_chirpy_red;

-- name: ResetUsers :many
DELETE FROM users RETURNING *;

-- name: UpgradeUser :one
UPDATE users SET is_chirpy_red = true WHERE id = $1 RETURNING id;

