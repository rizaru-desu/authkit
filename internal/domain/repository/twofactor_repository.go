package repository

import (
	"context"

	"mns/backend/internal/domain/entity"
)

// TwoFactorRepository defines persistence operations for the `two_factors`
// table.
type TwoFactorRepository interface {
	Create(ctx context.Context, tf *entity.TwoFactor) error
	GetByUserID(ctx context.Context, userID string) (*entity.TwoFactor, error)
	SetVerified(ctx context.Context, userID string, verified bool) error
	UpdateBackupCodes(ctx context.Context, userID, backupCodes string) error
	DeleteByUserID(ctx context.Context, userID string) error
}
