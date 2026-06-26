-- name: CreateAccount :exec
INSERT INTO accounts (
    id, user_id, account_id, provider_id,
    access_token, refresh_token, id_token,
    access_token_expires_at, refresh_token_expires_at,
    scope, password, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: GetAccountByProvider :one
SELECT * FROM accounts
WHERE provider_id = $1 AND account_id = $2;

-- name: GetCredentialByUserID :one
SELECT * FROM accounts
WHERE user_id = $1 AND provider_id = 'credential';

-- name: ListAccountsByUserID :many
SELECT * FROM accounts
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateAccountPassword :exec
UPDATE accounts
SET password = $2, updated_at = $3
WHERE id = $1;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE id = $1;
