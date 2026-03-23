-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, password_hash, email_verified, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, google_id, email_verified, created_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, email_verified, created_at
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
SELECT id, email, password_hash, google_id, email_verified, created_at
FROM users
WHERE google_id = $1;

-- name: CreateGoogleUser :one
INSERT INTO users (email, password_hash, google_id, email_verified)
VALUES ($1, '', $2, true)
RETURNING id, email, password_hash, google_id, email_verified, created_at;

-- name: LinkGoogleID :exec
UPDATE users SET google_id = $2 WHERE id = $1;

-- name: SetVerificationToken :exec
UPDATE users
SET verification_token = $2, verification_expires_at = $3
WHERE id = $1;

-- name: GetUserByVerificationToken :one
SELECT id, email, password_hash, email_verified, created_at
FROM users
WHERE verification_token = $1
  AND verification_expires_at > NOW();

-- name: SetEmailVerified :exec
UPDATE users
SET email_verified = true, verification_token = NULL, verification_expires_at = NULL
WHERE id = $1;
