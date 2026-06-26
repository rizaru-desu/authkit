// Package repository defines data-access contracts for the domain.
// Implementations live in internal/repository/.
package repository

import (
	"context"
	"time"

	"mns/backend/internal/domain/entity"
)

// UserRepository defines persistence operations for the `users` table.
type UserRepository interface {
	GetByID(ctx context.Context, id string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	SetEmailVerified(ctx context.Context, id string, verified bool) error
	SetTwoFactorEnabled(ctx context.Context, id string, enabled bool) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*entity.User, error)

	// Admin operations.
	SetRole(ctx context.Context, id string, role entity.Role) error
	Ban(ctx context.Context, id string, reason *string, expires *time.Time) error
	Unban(ctx context.Context, id string) error
	Search(ctx context.Context, query string, limit, offset int) ([]*entity.User, error)
	Count(ctx context.Context, query string) (int, error)
}
