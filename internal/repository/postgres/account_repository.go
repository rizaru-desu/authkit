package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mns/backend/internal/domain/entity"
	"mns/backend/internal/repository/postgres/sqlc"
)

// AccountRepository implements domain/repository.AccountRepository.
type AccountRepository struct {
	q *sqlc.Queries
}

// NewAccountRepository creates an AccountRepository backed by the given pool.
func NewAccountRepository(db *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{q: sqlc.New(db)}
}

func (r *AccountRepository) Create(ctx context.Context, a *entity.Account) error {
	err := r.q.CreateAccount(ctx, sqlc.CreateAccountParams{
		ID:                    a.ID,
		UserID:                a.UserID,
		AccountID:             a.AccountID,
		ProviderID:            a.ProviderID,
		AccessToken:           a.AccessToken,
		RefreshToken:          a.RefreshToken,
		IDToken:               a.IDToken,
		AccessTokenExpiresAt:  toTSPtr(a.AccessTokenExpiresAt),
		RefreshTokenExpiresAt: toTSPtr(a.RefreshTokenExpiresAt),
		Scope:                 a.Scope,
		Password:              a.Password,
		CreatedAt:             toTS(a.CreatedAt),
		UpdatedAt:             toTS(a.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

func (r *AccountRepository) GetByProvider(ctx context.Context, providerID, accountID string) (*entity.Account, error) {
	row, err := r.q.GetAccountByProvider(ctx, sqlc.GetAccountByProviderParams{
		ProviderID: providerID,
		AccountID:  accountID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("get account by provider: %w", err)
	}
	return accountToEntity(row), nil
}

func (r *AccountRepository) GetCredentialByUserID(ctx context.Context, userID string) (*entity.Account, error) {
	row, err := r.q.GetCredentialByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("get credential account: %w", err)
	}
	return accountToEntity(row), nil
}

func (r *AccountRepository) ListByUserID(ctx context.Context, userID string) ([]*entity.Account, error) {
	rows, err := r.q.ListAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	accounts := make([]*entity.Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, accountToEntity(row))
	}
	return accounts, nil
}

func (r *AccountRepository) UpdatePassword(ctx context.Context, accountID, hashedPassword string) error {
	err := r.q.UpdateAccountPassword(ctx, sqlc.UpdateAccountPasswordParams{
		ID:        accountID,
		Password:  &hashedPassword,
		UpdatedAt: toTS(time.Now().UTC()),
	})
	if err != nil {
		return fmt.Errorf("update account password: %w", err)
	}
	return nil
}

func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	if err := r.q.DeleteAccount(ctx, id); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

func accountToEntity(a sqlc.Account) *entity.Account {
	return &entity.Account{
		ID:                    a.ID,
		UserID:                a.UserID,
		AccountID:             a.AccountID,
		ProviderID:            a.ProviderID,
		AccessToken:           a.AccessToken,
		RefreshToken:          a.RefreshToken,
		IDToken:               a.IDToken,
		AccessTokenExpiresAt:  fromTSPtr(a.AccessTokenExpiresAt),
		RefreshTokenExpiresAt: fromTSPtr(a.RefreshTokenExpiresAt),
		Scope:                 a.Scope,
		Password:              a.Password,
		CreatedAt:             a.CreatedAt.Time,
		UpdatedAt:             a.UpdatedAt.Time,
	}
}
