package repository

import (
	"context"

	"authkit/internal/domain/entity"
)

// AuthRepository groups atomic auth writes that must span multiple tables.
type AuthRepository interface {
	// RegisterCredentialUser atomically inserts a user and its credential
	// account (Better Auth email/password sign-up) in a single transaction.
	RegisterCredentialUser(ctx context.Context, user *entity.User, account *entity.Account) error
}
