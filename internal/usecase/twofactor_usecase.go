package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"mns/backend/internal/domain/entity"
	"mns/backend/internal/domain/repository"
	"mns/backend/pkg/id"
	"mns/backend/pkg/secure"
	"mns/backend/pkg/totp"
)

const backupCodeCount = 10

// TwoFactorUsecase manages TOTP-based two-factor authentication.
type TwoFactorUsecase struct {
	users     repository.UserRepository
	accounts  repository.AccountRepository
	twoFactor repository.TwoFactorRepository
	issuer    string
}

// NewTwoFactorUsecase wires the two-factor usecase.
func NewTwoFactorUsecase(
	users repository.UserRepository,
	accounts repository.AccountRepository,
	twoFactor repository.TwoFactorRepository,
	issuer string,
) *TwoFactorUsecase {
	return &TwoFactorUsecase{users: users, accounts: accounts, twoFactor: twoFactor, issuer: issuer}
}

// EnableResult is returned to the user once during enrolment.
type EnableResult struct {
	TOTPURI     string
	BackupCodes []string
}

// Enable starts 2FA enrolment: verifies the password, stores an unverified
// secret + backup codes, and returns the otpauth URI and plaintext codes.
func (uc *TwoFactorUsecase) Enable(ctx context.Context, userID, password string) (*EnableResult, error) {
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if err := uc.checkPassword(ctx, userID, password); err != nil {
		return nil, err
	}

	secret, uri, err := totp.Generate(uc.issuer, user.Email)
	if err != nil {
		return nil, err
	}
	plain, hashed, err := secure.GenerateBackupCodes(backupCodeCount)
	if err != nil {
		return nil, err
	}

	// Replace any prior (re-)enrolment.
	if err := uc.twoFactor.DeleteByUserID(ctx, userID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := uc.twoFactor.Create(ctx, &entity.TwoFactor{
		ID:          id.New(),
		UserID:      userID,
		Secret:      secret,
		BackupCodes: strings.Join(hashed, ","),
		Verified:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return nil, err
	}

	return &EnableResult{TOTPURI: uri, BackupCodes: plain}, nil
}

// ConfirmEnable verifies the first TOTP code and activates 2FA for the user.
func (uc *TwoFactorUsecase) ConfirmEnable(ctx context.Context, userID, code string) error {
	tf, err := uc.twoFactor.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return entity.ErrTwoFactorNotSet
		}
		return fmt.Errorf("get two factor: %w", err)
	}
	if !totp.Validate(code, tf.Secret) {
		return entity.ErrInvalidCode
	}
	if err := uc.twoFactor.SetVerified(ctx, userID, true); err != nil {
		return err
	}
	return uc.users.SetTwoFactorEnabled(ctx, userID, true)
}

// Disable turns off 2FA after a password check.
func (uc *TwoFactorUsecase) Disable(ctx context.Context, userID, password string) error {
	if err := uc.checkPassword(ctx, userID, password); err != nil {
		return err
	}
	if err := uc.twoFactor.DeleteByUserID(ctx, userID); err != nil {
		return err
	}
	return uc.users.SetTwoFactorEnabled(ctx, userID, false)
}

// Verify validates a TOTP code or consumes a one-time backup code.
// Implements usecase.TwoFactorVerifier for the login flow.
func (uc *TwoFactorUsecase) Verify(ctx context.Context, userID, code string) error {
	tf, err := uc.twoFactor.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return entity.ErrInvalidCode
		}
		return fmt.Errorf("get two factor: %w", err)
	}

	if totp.Validate(code, tf.Secret) {
		return nil
	}
	return uc.consumeBackupCode(ctx, userID, tf, code)
}

func (uc *TwoFactorUsecase) consumeBackupCode(ctx context.Context, userID string, tf *entity.TwoFactor, code string) error {
	want := secure.Hash(code)
	if tf.BackupCodes == "" {
		return entity.ErrInvalidCode
	}
	codes := strings.Split(tf.BackupCodes, ",")
	remaining := make([]string, 0, len(codes))
	matched := false
	for _, h := range codes {
		if !matched && secure.Equal(h, want) {
			matched = true
			continue // consume it
		}
		remaining = append(remaining, h)
	}
	if !matched {
		return entity.ErrInvalidCode
	}
	if err := uc.twoFactor.UpdateBackupCodes(ctx, userID, strings.Join(remaining, ",")); err != nil {
		return err
	}
	return nil
}

func (uc *TwoFactorUsecase) checkPassword(ctx context.Context, userID, password string) error {
	account, err := uc.accounts.GetCredentialByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return entity.ErrInvalidCredential
		}
		return fmt.Errorf("get credential: %w", err)
	}
	return verifyAccountPassword(account, password)
}
