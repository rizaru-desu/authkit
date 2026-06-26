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

// SessionRepository implements domain/repository.SessionRepository.
type SessionRepository struct {
	q *sqlc.Queries
}

// NewSessionRepository creates a SessionRepository backed by the given pool.
func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{q: sqlc.New(db)}
}

func (r *SessionRepository) Create(ctx context.Context, s *entity.Session) error {
	err := r.q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:             s.ID,
		UserID:         s.UserID,
		Token:          s.Token,
		ExpiresAt:      toTS(s.ExpiresAt),
		IpAddress:      s.IPAddress,
		UserAgent:      s.UserAgent,
		ImpersonatedBy: s.ImpersonatedBy,
		CreatedAt:      toTS(s.CreatedAt),
		UpdatedAt:      toTS(s.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *SessionRepository) GetByToken(ctx context.Context, token string) (*entity.Session, error) {
	row, err := r.q.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("get session by token: %w", err)
	}
	return sessionToEntity(row), nil
}

func (r *SessionRepository) ListByUserID(ctx context.Context, userID string) ([]*entity.Session, error) {
	rows, err := r.q.ListSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	sessions := make([]*entity.Session, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, sessionToEntity(row))
	}
	return sessions, nil
}

func sessionToEntity(s sqlc.Session) *entity.Session {
	return &entity.Session{
		ID:             s.ID,
		UserID:         s.UserID,
		Token:          s.Token,
		ExpiresAt:      s.ExpiresAt.Time,
		IPAddress:      s.IpAddress,
		UserAgent:      s.UserAgent,
		ImpersonatedBy: s.ImpersonatedBy,
		CreatedAt:      s.CreatedAt.Time,
		UpdatedAt:      s.UpdatedAt.Time,
	}
}

func (r *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	if err := r.q.DeleteSessionByToken(ctx, token); err != nil {
		return fmt.Errorf("delete session by token: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	if err := r.q.DeleteSessionsByUserID(ctx, userID); err != nil {
		return fmt.Errorf("delete sessions by user: %w", err)
	}
	return nil
}
