package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"authkit/internal/domain/entity"
	"authkit/internal/domain/repository"
	"authkit/pkg/access"
	"authkit/pkg/id"
	"authkit/pkg/secure"
)

// impersonationDuration matches Better Auth's default impersonationSessionDuration.
const impersonationDuration = 1 * time.Hour

// AdminUsecase implements the Better Auth admin plugin operations.
type AdminUsecase struct {
	users    repository.UserRepository
	accounts repository.AccountRepository
	sessions repository.SessionRepository
	authRepo repository.AuthRepository
	ac       *access.Controller
	expiry   time.Duration
}

// NewAdminUsecase wires the admin usecase.
func NewAdminUsecase(
	users repository.UserRepository,
	accounts repository.AccountRepository,
	sessions repository.SessionRepository,
	authRepo repository.AuthRepository,
	ac *access.Controller,
	expiry time.Duration,
) *AdminUsecase {
	return &AdminUsecase{
		users: users, accounts: accounts, sessions: sessions,
		authRepo: authRepo, ac: ac, expiry: expiry,
	}
}

// HasPermission checks whether a role (explicit, or resolved from userID) is
// granted the requested permissions.
func (uc *AdminUsecase) HasPermission(ctx context.Context, userID, role string, permissions map[string][]string) (bool, error) {
	if role == "" {
		if userID == "" {
			return false, entity.ErrInvalidInput
		}
		user, err := uc.users.GetByID(ctx, userID)
		if err != nil {
			return false, fmt.Errorf("get user: %w", err)
		}
		role = string(user.Role)
	}
	return uc.ac.Check(role, permissions), nil
}

// CreateUserInput is the admin user-creation DTO.
type CreateUserInput struct {
	Name     string
	Email    string
	Password string
	Role     entity.Role
}

// CreateUser creates a user with a credential account (no auto sign-in).
func (uc *AdminUsecase) CreateUser(ctx context.Context, in CreateUserInput) (*entity.User, error) {
	existing, err := uc.users.GetByEmail(ctx, in.Email)
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, entity.ErrEmailTaken
	}

	hashedStr, err := secure.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	role := in.Role
	if role == "" {
		role = entity.RoleUser
	}
	now := time.Now().UTC()
	userID := id.New()

	user := &entity.User{
		ID: userID, Name: in.Name, Email: in.Email, Role: role,
		CreatedAt: now, UpdatedAt: now,
	}
	account := &entity.Account{
		ID: id.New(), UserID: userID, AccountID: userID,
		ProviderID: entity.ProviderCredential, Password: &hashedStr,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := uc.authRepo.RegisterCredentialUser(ctx, user, account); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// ListUsersResult is the paginated admin listing.
type ListUsersResult struct {
	Users  []*entity.User
	Total  int
	Limit  int
	Offset int
}

// ListUsers searches/paginates users.
func (uc *AdminUsecase) ListUsers(ctx context.Context, search string, limit, offset int) (*ListUsersResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	users, err := uc.users.Search(ctx, search, limit, offset)
	if err != nil {
		return nil, err
	}
	total, err := uc.users.Count(ctx, search)
	if err != nil {
		return nil, err
	}
	return &ListUsersResult{Users: users, Total: total, Limit: limit, Offset: offset}, nil
}

// SetRole changes a user's role and returns the updated user.
func (uc *AdminUsecase) SetRole(ctx context.Context, userID string, role entity.Role) (*entity.User, error) {
	if _, err := uc.users.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if err := uc.users.SetRole(ctx, userID, role); err != nil {
		return nil, err
	}
	return uc.users.GetByID(ctx, userID)
}

// SetUserPassword sets a user's credential password, creating the credential
// account if it does not exist yet (matches Better Auth behaviour).
func (uc *AdminUsecase) SetUserPassword(ctx context.Context, userID, newPassword string) error {
	hashedStr, err := secure.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	account, err := uc.accounts.GetCredentialByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			now := time.Now().UTC()
			return uc.accounts.Create(ctx, &entity.Account{
				ID: id.New(), UserID: userID, AccountID: userID,
				ProviderID: entity.ProviderCredential, Password: &hashedStr,
				CreatedAt: now, UpdatedAt: now,
			})
		}
		return fmt.Errorf("get credential: %w", err)
	}
	return uc.accounts.UpdatePassword(ctx, account.ID, hashedStr)
}

// BanUser bans a user and revokes all their sessions. banExpiresIn is seconds
// until the ban lifts (0 = permanent).
func (uc *AdminUsecase) BanUser(ctx context.Context, userID string, reason *string, banExpiresIn int64) (*entity.User, error) {
	if _, err := uc.users.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	var expires *time.Time
	if banExpiresIn > 0 {
		t := time.Now().UTC().Add(time.Duration(banExpiresIn) * time.Second)
		expires = &t
	}
	if err := uc.users.Ban(ctx, userID, reason, expires); err != nil {
		return nil, err
	}
	if err := uc.sessions.DeleteByUserID(ctx, userID); err != nil {
		return nil, err
	}
	return uc.users.GetByID(ctx, userID)
}

// UnbanUser lifts a ban and returns the updated user.
func (uc *AdminUsecase) UnbanUser(ctx context.Context, userID string) (*entity.User, error) {
	if _, err := uc.users.GetByID(ctx, userID); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if err := uc.users.Unban(ctx, userID); err != nil {
		return nil, err
	}
	return uc.users.GetByID(ctx, userID)
}

// ListUserSessions returns all sessions for a user.
func (uc *AdminUsecase) ListUserSessions(ctx context.Context, userID string) ([]*entity.Session, error) {
	return uc.sessions.ListByUserID(ctx, userID)
}

// RevokeUserSession revokes a single session by its token.
func (uc *AdminUsecase) RevokeUserSession(ctx context.Context, token string) error {
	return uc.sessions.DeleteByToken(ctx, token)
}

// RevokeUserSessions revokes all sessions of a user.
func (uc *AdminUsecase) RevokeUserSessions(ctx context.Context, userID string) error {
	return uc.sessions.DeleteByUserID(ctx, userID)
}

// RemoveUser hard-deletes a user (sessions/accounts cascade).
func (uc *AdminUsecase) RemoveUser(ctx context.Context, userID string) error {
	return uc.users.Delete(ctx, userID)
}

// Impersonate creates a short-lived session for targetUserID, tagged with the
// acting admin id.
func (uc *AdminUsecase) Impersonate(ctx context.Context, adminID, targetUserID string, ip, ua *string) (*entity.Session, *entity.User, error) {
	user, err := uc.users.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}
	token, err := secure.Token()
	if err != nil {
		return nil, nil, fmt.Errorf("generate token: %w", err)
	}
	now := time.Now().UTC()
	adminRef := adminID
	session := &entity.Session{
		ID:             id.New(),
		UserID:         targetUserID,
		Token:          token,
		ExpiresAt:      now.Add(impersonationDuration),
		IPAddress:      ip,
		UserAgent:      ua,
		ImpersonatedBy: &adminRef,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := uc.sessions.Create(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}
	return session, user, nil
}

// StopImpersonating ends an impersonation session.
func (uc *AdminUsecase) StopImpersonating(ctx context.Context, session *entity.Session) error {
	if session.ImpersonatedBy == nil {
		return entity.ErrNotImpersonating
	}
	return uc.sessions.DeleteByToken(ctx, session.Token)
}
