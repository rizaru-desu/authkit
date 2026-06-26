package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"authkit/internal/domain/entity"
	"authkit/internal/repository/postgres/sqlc"
)

// AuthRepository implements domain/repository.AuthRepository using pgx
// transactions so multi-table writes stay atomic.
type AuthRepository struct {
	pool *pgxpool.Pool
}

// NewAuthRepository creates an AuthRepository backed by the given pool.
func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

// RegisterCredentialUser inserts the user and its credential account in one tx.
func (r *AuthRepository) RegisterCredentialUser(ctx context.Context, user *entity.User, account *entity.Account) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	q := sqlc.New(tx)

	if err := q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:               user.ID,
		Name:             user.Name,
		Email:            user.Email,
		EmailVerified:    user.EmailVerified,
		Image:            user.Image,
		Role:             string(user.Role),
		TwoFactorEnabled: user.TwoFactorEnabled,
		CreatedAt:        toTS(user.CreatedAt),
		UpdatedAt:        toTS(user.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if err := q.CreateAccount(ctx, sqlc.CreateAccountParams{
		ID:                    account.ID,
		UserID:                account.UserID,
		AccountID:             account.AccountID,
		ProviderID:            account.ProviderID,
		AccessToken:           account.AccessToken,
		RefreshToken:          account.RefreshToken,
		IDToken:               account.IDToken,
		AccessTokenExpiresAt:  toTSPtr(account.AccessTokenExpiresAt),
		RefreshTokenExpiresAt: toTSPtr(account.RefreshTokenExpiresAt),
		Scope:                 account.Scope,
		Password:              account.Password,
		CreatedAt:             toTS(account.CreatedAt),
		UpdatedAt:             toTS(account.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("create account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
