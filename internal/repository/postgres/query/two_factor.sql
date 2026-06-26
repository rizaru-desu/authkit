-- name: CreateTwoFactor :exec
INSERT INTO two_factors (id, user_id, secret, backup_codes, verified, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetTwoFactorByUserID :one
SELECT * FROM two_factors
WHERE user_id = $1;

-- name: SetTwoFactorVerified :exec
UPDATE two_factors
SET verified = $2, updated_at = $3
WHERE user_id = $1;

-- name: UpdateTwoFactorBackupCodes :exec
UPDATE two_factors
SET backup_codes = $2, updated_at = $3
WHERE user_id = $1;

-- name: DeleteTwoFactorByUserID :exec
DELETE FROM two_factors
WHERE user_id = $1;
