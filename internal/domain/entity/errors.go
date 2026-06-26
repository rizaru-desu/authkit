package entity

import "errors"

// Domain-level sentinel errors. Outer layers map these to transport responses
// (e.g. HTTP 404 / 409) without depending on the database driver.
var (
	ErrNotFound          = errors.New("resource not found")
	ErrEmailTaken        = errors.New("email already registered")
	ErrInvalidInput      = errors.New("invalid input")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidCredential = errors.New("invalid email or password")
	ErrSessionExpired    = errors.New("session expired")
	ErrTwoFactorRequired = errors.New("two-factor authentication required")
	ErrInvalidCode       = errors.New("invalid or expired code")
	ErrTokenExpired      = errors.New("token expired")
	ErrTwoFactorNotSet   = errors.New("two-factor not configured")
	ErrBanned            = errors.New("user is banned")
	ErrEmailNotVerified  = errors.New("email not verified")
	ErrForbidden         = errors.New("forbidden")
	ErrNotImpersonating  = errors.New("not an impersonation session")
)
