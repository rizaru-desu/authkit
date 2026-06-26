-- name: CreateUser :exec
INSERT INTO users (id, name, email, email_verified, image, role, two_factor_enabled, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetUserByID :one
SELECT id, name, email, email_verified, image, role, created_at, updated_at, two_factor_enabled, banned, ban_reason, ban_expires
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, email_verified, image, role, created_at, updated_at, two_factor_enabled, banned, ban_reason, ban_expires
FROM users
WHERE email = $1;

-- name: UpdateUser :exec
UPDATE users
SET name = $2, email = $3, email_verified = $4, image = $5, role = $6, updated_at = $7
WHERE id = $1;

-- name: SetUserEmailVerified :exec
UPDATE users
SET email_verified = $2, updated_at = $3
WHERE id = $1;

-- name: SetUserTwoFactorEnabled :exec
UPDATE users
SET two_factor_enabled = $2, updated_at = $3
WHERE id = $1;

-- name: SetUserRole :exec
UPDATE users
SET role = $2, updated_at = $3
WHERE id = $1;

-- name: BanUser :exec
UPDATE users
SET banned = true, ban_reason = $2, ban_expires = $3, updated_at = $4
WHERE id = $1;

-- name: UnbanUser :exec
UPDATE users
SET banned = false, ban_reason = NULL, ban_expires = NULL, updated_at = $2
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, name, email, email_verified, image, role, created_at, updated_at, two_factor_enabled, banned, ban_reason, ban_expires
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: SearchUsers :many
SELECT id, name, email, email_verified, image, role, created_at, updated_at, two_factor_enabled, banned, ban_reason, ban_expires
FROM users
WHERE (sqlc.arg(search)::text = '' OR email ILIKE '%' || sqlc.arg(search)::text || '%' OR name ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountUsers :one
SELECT count(*)
FROM users
WHERE (sqlc.arg(search)::text = '' OR email ILIKE '%' || sqlc.arg(search)::text || '%' OR name ILIKE '%' || sqlc.arg(search)::text || '%');
