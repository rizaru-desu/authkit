package repository

import (
	"context"

	"mns/backend/internal/domain/entity"
)

// AccountRepository defines persistence operations for the `accounts` table.
type AccountRepository interface {
	Create(ctx context.Context, account *entity.Account) error
	GetByProvider(ctx context.Context, providerID, accountID string) (*entity.Account, error)
	GetCredentialByUserID(ctx context.Context, userID string) (*entity.Account, error)
	ListByUserID(ctx context.Context, userID string) ([]*entity.Account, error)
	UpdatePassword(ctx context.Context, accountID, hashedPassword string) error
	Delete(ctx context.Context, id string) error
}
