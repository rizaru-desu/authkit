package entity

import "time"

// TwoFactor mirrors the Better Auth twoFactor plugin table. Secret is the TOTP
// secret; BackupCodes holds comma-joined SHA-256 hashes of one-time codes.
type TwoFactor struct {
	ID          string
	UserID      string
	Secret      string
	BackupCodes string
	Verified    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
