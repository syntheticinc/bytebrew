-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, password_hash, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, google_id, created_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, created_at
FROM users
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: DeleteUserByID :exec
DELETE FROM users WHERE id = $1;

-- name: SetPasswordResetToken :exec
UPDATE users
SET password_reset_token = $2, password_reset_expires_at = $3
WHERE id = $1;

-- name: GetUserByResetToken :one
SELECT id, email, password_hash, created_at, password_reset_token, password_reset_expires_at
FROM users
WHERE password_reset_token = $1
  AND password_reset_expires_at > NOW();

-- name: ClearPasswordResetToken :exec
UPDATE users
SET password_reset_token = NULL, password_reset_expires_at = NULL
WHERE id = $1;

-- name: UpdatePasswordAndClearResetToken :exec
UPDATE users
SET password_hash = $2, password_reset_token = NULL, password_reset_expires_at = NULL
WHERE id = $1;

-- name: GetUserByGoogleID :one
SELECT id, email, password_hash, google_id, created_at
FROM users
WHERE google_id = $1;

-- name: CreateGoogleUser :one
INSERT INTO users (email, password_hash, google_id)
VALUES ($1, '', $2)
RETURNING id, email, password_hash, google_id, created_at;

-- name: LinkGoogleID :exec
UPDATE users SET google_id = $2 WHERE id = $1;
