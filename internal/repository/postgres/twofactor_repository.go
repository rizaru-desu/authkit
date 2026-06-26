package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"authkit/internal/domain/entity"
	"authkit/internal/repository/postgres/sqlc"
)

// TwoFactorRepository implements domain/repository.TwoFactorRepository.
type TwoFactorRepository struct {
	q *sqlc.Queries
}

// NewTwoFactorRepository creates a TwoFactorRepository backed by the pool.
func NewTwoFactorRepository(db *pgxpool.Pool) *TwoFactorRepository {
	return &TwoFactorRepository{q: sqlc.New(db)}
}

func (r *TwoFactorRepository) Create(ctx context.Context, tf *entity.TwoFactor) error {
	err := r.q.CreateTwoFactor(ctx, sqlc.CreateTwoFactorParams{
		ID:          tf.ID,
		UserID:      tf.UserID,
		Secret:      tf.Secret,
		BackupCodes: tf.BackupCodes,
		Verified:    tf.Verified,
		CreatedAt:   toTS(tf.CreatedAt),
		UpdatedAt:   toTS(tf.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("create two factor: %w", err)
	}
	return nil
}

func (r *TwoFactorRepository) GetByUserID(ctx context.Context, userID string) (*entity.TwoFactor, error) {
	row, err := r.q.GetTwoFactorByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("get two factor: %w", err)
	}
	return &entity.TwoFactor{
		ID:          row.ID,
		UserID:      row.UserID,
		Secret:      row.Secret,
		BackupCodes: row.BackupCodes,
		Verified:    row.Verified,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (r *TwoFactorRepository) SetVerified(ctx context.Context, userID string, verified bool) error {
	err := r.q.SetTwoFactorVerified(ctx, sqlc.SetTwoFactorVerifiedParams{
		UserID:    userID,
		Verified:  verified,
		UpdatedAt: toTS(time.Now().UTC()),
	})
	if err != nil {
		return fmt.Errorf("set two factor verified: %w", err)
	}
	return nil
}

func (r *TwoFactorRepository) UpdateBackupCodes(ctx context.Context, userID, backupCodes string) error {
	err := r.q.UpdateTwoFactorBackupCodes(ctx, sqlc.UpdateTwoFactorBackupCodesParams{
		UserID:      userID,
		BackupCodes: backupCodes,
		UpdatedAt:   toTS(time.Now().UTC()),
	})
	if err != nil {
		return fmt.Errorf("update backup codes: %w", err)
	}
	return nil
}

func (r *TwoFactorRepository) DeleteByUserID(ctx context.Context, userID string) error {
	if err := r.q.DeleteTwoFactorByUserID(ctx, userID); err != nil {
		return fmt.Errorf("delete two factor: %w", err)
	}
	return nil
}
