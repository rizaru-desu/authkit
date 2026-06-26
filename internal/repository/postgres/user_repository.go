// Package postgres adapts sqlc-generated queries to the domain repository
// contracts. It maps between sqlc row models and domain entities, and
// translates driver errors into domain sentinel errors.
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

// UserRepository implements domain/repository.UserRepository using sqlc + pgx.
type UserRepository struct {
	q *sqlc.Queries
}

// NewUserRepository creates a UserRepository backed by the given pool.
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{q: sqlc.New(db)}
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return userToEntity(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return userToEntity(row), nil
}

func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	err := r.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Image:         user.Image,
		Role:          string(user.Role),
		UpdatedAt:     toTS(user.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserRepository) SetEmailVerified(ctx context.Context, id string, verified bool) error {
	err := r.q.SetUserEmailVerified(ctx, sqlc.SetUserEmailVerifiedParams{
		ID:            id,
		EmailVerified: verified,
		UpdatedAt:     toTS(time.Now().UTC()),
	})
	if err != nil {
		return fmt.Errorf("set email verified: %w", err)
	}
	return nil
}

func (r *UserRepository) SetTwoFactorEnabled(ctx context.Context, id string, enabled bool) error {
	err := r.q.SetUserTwoFactorEnabled(ctx, sqlc.SetUserTwoFactorEnabledParams{
		ID:               id,
		TwoFactorEnabled: enabled,
		UpdatedAt:        toTS(time.Now().UTC()),
	})
	if err != nil {
		return fmt.Errorf("set two factor enabled: %w", err)
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	if err := r.q.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *UserRepository) SetRole(ctx context.Context, id string, role entity.Role) error {
	err := r.q.SetUserRole(ctx, sqlc.SetUserRoleParams{
		ID:        id,
		Role:      string(role),
		UpdatedAt: toTS(time.Now().UTC()),
	})
	if err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	return nil
}

func (r *UserRepository) Ban(ctx context.Context, id string, reason *string, expires *time.Time) error {
	err := r.q.BanUser(ctx, sqlc.BanUserParams{
		ID:         id,
		BanReason:  reason,
		BanExpires: toTSPtr(expires),
		UpdatedAt:  toTS(time.Now().UTC()),
	})
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}
	return nil
}

func (r *UserRepository) Unban(ctx context.Context, id string) error {
	err := r.q.UnbanUser(ctx, sqlc.UnbanUserParams{
		ID:        id,
		UpdatedAt: toTS(time.Now().UTC()),
	})
	if err != nil {
		return fmt.Errorf("unban user: %w", err)
	}
	return nil
}

func (r *UserRepository) Search(ctx context.Context, query string, limit, offset int) ([]*entity.User, error) {
	rows, err := r.q.SearchUsers(ctx, sqlc.SearchUsersParams{
		Search: query,
		Lim:    int32(limit),
		Off:    int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	users := make([]*entity.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, userToEntity(row))
	}
	return users, nil
}

func (r *UserRepository) Count(ctx context.Context, query string) (int, error) {
	n, err := r.q.CountUsers(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return int(n), nil
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	rows, err := r.q.ListUsers(ctx, sqlc.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]*entity.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, userToEntity(row))
	}
	return users, nil
}

func userToEntity(u sqlc.User) *entity.User {
	return &entity.User{
		ID:               u.ID,
		Name:             u.Name,
		Email:            u.Email,
		EmailVerified:    u.EmailVerified,
		Image:            u.Image,
		Role:             entity.Role(u.Role),
		TwoFactorEnabled: u.TwoFactorEnabled,
		Banned:           u.Banned,
		BanReason:        u.BanReason,
		BanExpires:       fromTSPtr(u.BanExpires),
		CreatedAt:        u.CreatedAt.Time,
		UpdatedAt:        u.UpdatedAt.Time,
	}
}
