package repository

import (
	"context"

	"authkit/internal/domain/entity"
)

// VerificationRepository defines persistence operations for the
// `verifications` table.
type VerificationRepository interface {
	Create(ctx context.Context, v *entity.Verification) error
	GetByIdentifier(ctx context.Context, identifier string) (*entity.Verification, error)
	Delete(ctx context.Context, id string) error
}
