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

// EmailSender delivers transactional auth emails (verification, reset).
type EmailSender interface {
	SendVerificationEmail(ctx context.Context, to, token string) error
	SendPasswordReset(ctx context.Context, to, token string) error
}

// VerificationUsecase handles email verification and password reset tokens.
type VerificationUsecase struct {
	users         repository.UserRepository
	accounts      repository.AccountRepository
	sessions      repository.SessionRepository
	verifications repository.VerificationRepository
	email         EmailSender
	emailExpiry   time.Duration
	resetExpiry   time.Duration
}

// NewVerificationUsecase wires the verification usecase.
func NewVerificationUsecase(
	users repository.UserRepository,
	accounts repository.AccountRepository,
	sessions repository.SessionRepository,
	verifications repository.VerificationRepository,
	email EmailSender,
) *VerificationUsecase {
	return &VerificationUsecase{
		users:         users,
		accounts:      accounts,
		sessions:      sessions,
		verifications: verifications,
		email:         email,
		emailExpiry:   1 * time.Hour,
		resetExpiry:   1 * time.Hour,
	}
}

// RequestEmailVerification creates a verification token and emails the link.
// The token is never stored in plaintext nor returned to the caller.
func (uc *VerificationUsecase) RequestEmailVerification(ctx context.Context, email string) error {
	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	token, err := uc.createToken(ctx, entity.PurposeEmailVerify, user.ID, uc.emailExpiry)
	if err != nil {
		return err
	}
	return uc.email.SendVerificationEmail(ctx, email, token)
}

// VerifyEmail consumes an email-verification token and marks the email verified.
func (uc *VerificationUsecase) VerifyEmail(ctx context.Context, token string) error {
	v, err := uc.consumeToken(ctx, entity.PurposeEmailVerify, token)
	if err != nil {
		return err
	}
	if err := uc.users.SetEmailVerified(ctx, v.Value, true); err != nil {
		return err
	}
	return uc.verifications.Delete(ctx, v.ID)
}

// RequestPasswordReset creates a reset token and emails the link.
func (uc *VerificationUsecase) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	token, err := uc.createToken(ctx, entity.PurposePasswordReset, user.ID, uc.resetExpiry)
	if err != nil {
		return err
	}
	return uc.email.SendPasswordReset(ctx, email, token)
}

// ResetPassword consumes a reset token, updates the credential password and
// revokes all of the user's sessions.
func (uc *VerificationUsecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	v, err := uc.consumeToken(ctx, entity.PurposePasswordReset, token)
	if err != nil {
		return err
	}

	account, err := uc.accounts.GetCredentialByUserID(ctx, v.Value)
	if err != nil {
		return fmt.Errorf("get credential: %w", err)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := uc.accounts.UpdatePassword(ctx, account.ID, string(hashed)); err != nil {
		return err
	}
	if err := uc.verifications.Delete(ctx, v.ID); err != nil {
		return err
	}
	// Force re-login everywhere after a password change.
	return uc.sessions.DeleteByUserID(ctx, v.Value)
}

func (uc *VerificationUsecase) createToken(ctx context.Context, purpose, userID string, ttl time.Duration) (string, error) {
	token, err := secure.Token()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := uc.verifications.Create(ctx, &entity.Verification{
		ID:         id.New(),
		Identifier: purpose + ":" + secure.Hash(token),
		Value:      userID,
		ExpiresAt:  now.Add(ttl),
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		return "", err
	}
	return token, nil
}

func (uc *VerificationUsecase) consumeToken(ctx context.Context, purpose, token string) (*entity.Verification, error) {
	identifier := purpose + ":" + secure.Hash(token)
	v, err := uc.verifications.GetByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, entity.ErrInvalidCode
		}
		return nil, fmt.Errorf("get verification: %w", err)
	}
	if time.Now().UTC().After(v.ExpiresAt) {
		_ = uc.verifications.Delete(ctx, v.ID)
		return nil, entity.ErrTokenExpired
	}
	return v, nil
}
