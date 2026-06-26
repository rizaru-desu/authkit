// Package usecase contains application business logic.
package usecase

import (
	"context"
	"fmt"

	"mns/backend/internal/domain/entity"
	"mns/backend/internal/domain/repository"
)

// UserUsecase handles user read/management operations. Registration lives in
// AuthUsecase (sign-up).
type UserUsecase struct {
	users repository.UserRepository
}

// NewUserUsecase creates a UserUsecase.
func NewUserUsecase(users repository.UserRepository) *UserUsecase {
	return &UserUsecase{users: users}
}

// GetUser retrieves a user by ID.
func (uc *UserUsecase) GetUser(ctx context.Context, id string) (*entity.User, error) {
	user, err := uc.users.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// ListUsers returns a paginated list of users.
func (uc *UserUsecase) ListUsers(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	users, err := uc.users.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// DeleteUser removes a user by ID (sessions/accounts cascade in the DB).
func (uc *UserUsecase) DeleteUser(ctx context.Context, id string) error {
	if err := uc.users.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
