package entity

import "time"

// Provider identifiers used by the `account` table.
const (
	ProviderCredential = "credential" // email/password; hash stored in Password
)

// Account mirrors the Better Auth core `account` table. One user may have many
// accounts (credential + social providers). For credential accounts the
// argon2id password hash is stored in Password.
type Account struct {
	ID                    string
	UserID                string
	AccountID             string
	ProviderID            string
	AccessToken           *string
	RefreshToken          *string
	IDToken               *string
	AccessTokenExpiresAt  *time.Time
	RefreshTokenExpiresAt *time.Time
	Scope                 *string
	Password              *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
