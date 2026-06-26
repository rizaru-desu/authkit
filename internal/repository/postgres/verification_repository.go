package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"authkit/internal/domain/entity"
	"authkit/internal/repository/postgres/sqlc"
)

// VerificationRepository implements domain/repository.VerificationRepository.
type VerificationRepository struct {
	q *sqlc.Queries
}

// NewVerificationRepository creates a VerificationRepository backed by the pool.
func NewVerificationRepository(db *pgxpool.Pool) *VerificationRepository {
	return &VerificationRepository{q: sqlc.New(db)}
}

func (r *VerificationRepository) Create(ctx context.Context, v *entity.Verification) error {
	err := r.q.CreateVerification(ctx, sqlc.CreateVerificationParams{
		ID:         v.ID,
		Identifier: v.Identifier,
		Value:      v.Value,
		ExpiresAt:  toTS(v.ExpiresAt),
		CreatedAt:  toTS(v.CreatedAt),
		UpdatedAt:  toTS(v.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("create verification: %w", err)
	}
	return nil
}

func (r *VerificationRepository) GetByIdentifier(ctx context.Context, identifier string) (*entity.Verification, error) {
	row, err := r.q.GetVerificationByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("get verification: %w", err)
	}
	return &entity.Verification{
		ID:         row.ID,
		Identifier: row.Identifier,
		Value:      row.Value,
		ExpiresAt:  row.ExpiresAt.Time,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}, nil
}

func (r *VerificationRepository) Delete(ctx context.Context, id string) error {
	if err := r.q.DeleteVerification(ctx, id); err != nil {
		return fmt.Errorf("delete verification: %w", err)
	}
	return nil
}
