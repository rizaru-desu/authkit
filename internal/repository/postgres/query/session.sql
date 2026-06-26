-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token, expires_at, ip_address, user_agent, impersonated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetSessionByToken :one
SELECT * FROM sessions
WHERE token = $1;

-- name: ListSessionsByUserID :many
SELECT * FROM sessions
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;

-- name: DeleteSessionByToken :exec
DELETE FROM sessions
WHERE token = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions
WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < $1;
