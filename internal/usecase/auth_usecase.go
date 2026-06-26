package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"authkit/internal/domain/entity"
	"authkit/internal/domain/repository"
	"authkit/pkg/id"
	"authkit/pkg/secure"
)

// TwoFactorVerifier verifies a 2FA code (TOTP or backup) for a user. Implemented
// by TwoFactorUsecase; injected here to avoid a hard usecase dependency.
type TwoFactorVerifier interface {
	Verify(ctx context.Context, userID, code string) error
}

// VerificationSender (re)sends an email-verification link. Implemented by
// VerificationUsecase; used to auto-send on a blocked, unverified sign-in.
type VerificationSender interface {
	RequestEmailVerification(ctx context.Context, email string) error
}

// AuthUsecase handles sign-up, login, logout and session validation.
type AuthUsecase struct {
	users                    repository.UserRepository
	accounts                 repository.AccountRepository
	sessions                 repository.SessionRepository
	authRepo                 repository.AuthRepository
	twoFactor                TwoFactorVerifier
	verification             VerificationSender
	requireEmailVerification bool
	expiry                   time.Duration
}

// NewAuthUsecase wires the auth usecase.
func NewAuthUsecase(
	users repository.UserRepository,
	accounts repository.AccountRepository,
	sessions repository.SessionRepository,
	authRepo repository.AuthRepository,
	twoFactor TwoFactorVerifier,
	verification VerificationSender,
	requireEmailVerification bool,
	expiry time.Duration,
) *AuthUsecase {
	return &AuthUsecase{
		users:                    users,
		accounts:                 accounts,
		sessions:                 sessions,
		authRepo:                 authRepo,
		twoFactor:                twoFactor,
		verification:             verification,
		requireEmailVerification: requireEmailVerification,
		expiry:                   expiry,
	}
}

// SignUpInput carries registration data and request metadata.
type SignUpInput struct {
	Name      string
	Email     string
	Password  string
	Image     *string
	Role      entity.Role
	IPAddress *string
	UserAgent *string
}

// SignUp registers a credential user and immediately issues a session
// (Better Auth auto sign-in on sign-up).
func (uc *AuthUsecase) SignUp(ctx context.Context, in SignUpInput) (*entity.Session, *entity.User, error) {
	existing, err := uc.users.GetByEmail(ctx, in.Email)
	if err != nil && !errors.Is(err, entity.ErrNotFound) {
		return nil, nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, nil, entity.ErrEmailTaken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	role := in.Role
	if role == "" {
		role = entity.RoleUser
	}
	now := time.Now().UTC()
	userID := id.New()
	hashedStr := string(hashed)

	user := &entity.User{
		ID:            userID,
		Name:          in.Name,
		Email:         in.Email,
		Image:         in.Image,
		EmailVerified: false,
		Role:          role,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	account := &entity.Account{
		ID:         id.New(),
		UserID:     userID,
		AccountID:  userID,
		ProviderID: entity.ProviderCredential,
		Password:   &hashedStr,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := uc.authRepo.RegisterCredentialUser(ctx, user, account); err != nil {
		return nil, nil, fmt.Errorf("register user: %w", err)
	}

	// When verification is required, don't auto sign-in; send the email instead.
	if uc.requireEmailVerification {
		_ = uc.verification.RequestEmailVerification(ctx, user.Email)
		return nil, user, nil
	}

	session, err := uc.issueSession(ctx, userID, in.IPAddress, in.UserAgent)
	if err != nil {
		return nil, nil, err
	}
	return session, user, nil
}

// LoginInput carries credentials and request metadata.
type LoginInput struct {
	Email     string
	Password  string
	Code      string // 2FA code (TOTP or backup), required when 2FA is enabled
	IPAddress *string
	UserAgent *string
}

// Login verifies credentials (and 2FA when enabled) and issues a session.
func (uc *AuthUsecase) Login(ctx context.Context, in LoginInput) (*entity.Session, *entity.User, error) {
	user, err := uc.users.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, nil, entity.ErrInvalidCredential
		}
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	account, err := uc.accounts.GetCredentialByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, nil, entity.ErrInvalidCredential
		}
		return nil, nil, fmt.Errorf("get credential: %w", err)
	}
	if err := verifyAccountPassword(account, in.Password); err != nil {
		return nil, nil, err
	}

	if user.IsBanned(time.Now().UTC()) {
		return nil, nil, entity.ErrBanned
	}

	if uc.requireEmailVerification && !user.EmailVerified {
		_ = uc.verification.RequestEmailVerification(ctx, user.Email) // resend on attempt
		return nil, nil, entity.ErrEmailNotVerified
	}

	if user.TwoFactorEnabled {
		if in.Code == "" {
			return nil, nil, entity.ErrTwoFactorRequired
		}
		if err := uc.twoFactor.Verify(ctx, user.ID, in.Code); err != nil {
			return nil, nil, err
		}
	}

	session, err := uc.issueSession(ctx, user.ID, in.IPAddress, in.UserAgent)
	if err != nil {
		return nil, nil, err
	}
	return session, user, nil
}

func (uc *AuthUsecase) issueSession(ctx context.Context, userID string, ip, ua *string) (*entity.Session, error) {
	token, err := secure.Token()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	now := time.Now().UTC()
	session := &entity.Session{
		ID:        id.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: now.Add(uc.expiry),
		IPAddress: ip,
		UserAgent: ua,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}

// Logout revokes the session identified by token.
func (uc *AuthUsecase) Logout(ctx context.Context, token string) error {
	if err := uc.sessions.DeleteByToken(ctx, token); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

// ListSessions returns all sessions belonging to the user.
func (uc *AuthUsecase) ListSessions(ctx context.Context, userID string) ([]*entity.Session, error) {
	return uc.sessions.ListByUserID(ctx, userID)
}

// RevokeSession revokes one of the user's sessions by token. A user may only
// revoke their own sessions.
func (uc *AuthUsecase) RevokeSession(ctx context.Context, userID, token string) error {
	s, err := uc.sessions.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return entity.ErrNotFound
		}
		return fmt.Errorf("get session: %w", err)
	}
	if s.UserID != userID {
		return entity.ErrForbidden
	}
	return uc.sessions.DeleteByToken(ctx, token)
}

// RevokeAllSessions revokes every session of the user (including the current one).
func (uc *AuthUsecase) RevokeAllSessions(ctx context.Context, userID string) error {
	return uc.sessions.DeleteByUserID(ctx, userID)
}

// RevokeOtherSessions revokes all of the user's sessions except the current one.
func (uc *AuthUsecase) RevokeOtherSessions(ctx context.Context, userID, currentToken string) error {
	sessions, err := uc.sessions.ListByUserID(ctx, userID)
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if s.Token == currentToken {
			continue
		}
		if err := uc.sessions.DeleteByToken(ctx, s.Token); err != nil {
			return err
		}
	}
	return nil
}

// ValidateSession resolves a bearer token to its user, rejecting expired ones.
func (uc *AuthUsecase) ValidateSession(ctx context.Context, token string) (*entity.User, *entity.Session, error) {
	session, err := uc.sessions.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, nil, entity.ErrUnauthorized
		}
		return nil, nil, fmt.Errorf("get session: %w", err)
	}
	if session.IsExpired(time.Now().UTC()) {
		_ = uc.sessions.DeleteByToken(ctx, token)
		return nil, nil, entity.ErrSessionExpired
	}
	user, err := uc.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}
	return user, session, nil
}

// verifyAccountPassword checks password against a credential account hash.
func verifyAccountPassword(account *entity.Account, password string) error {
	if account.Password == nil {
		return entity.ErrInvalidCredential
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*account.Password), []byte(password)); err != nil {
		return entity.ErrInvalidCredential
	}
	return nil
}
