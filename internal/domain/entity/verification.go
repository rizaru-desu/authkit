package entity

import "time"

// Verification purposes encoded into the `identifier` column.
const (
	PurposeEmailVerify   = "email-verify"
	PurposePasswordReset = "reset-password"
)

// Verification mirrors the Better Auth core `verification` table. Used for
// email verification and password reset tokens.
type Verification struct {
	ID         string
	Identifier string // "<purpose>:<sha256(token)>"
	Value      string // target user id
	ExpiresAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
