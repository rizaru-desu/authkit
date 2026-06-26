-- name: CreateVerification :exec
INSERT INTO verifications (id, identifier, value, expires_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetVerificationByIdentifier :one
SELECT * FROM verifications
WHERE identifier = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteVerification :exec
DELETE FROM verifications
WHERE id = $1;

-- name: DeleteExpiredVerifications :exec
DELETE FROM verifications
WHERE expires_at < $1;
