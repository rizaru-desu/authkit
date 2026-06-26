package repository

import (
	"context"

	"mns/backend/internal/domain/entity"
)

// SessionRepository defines persistence operations for the `sessions` table.
type SessionRepository interface {
	Create(ctx context.Context, session *entity.Session) error
	GetByToken(ctx context.Context, token string) (*entity.Session, error)
	ListByUserID(ctx context.Context, userID string) ([]*entity.Session, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID string) error
}
