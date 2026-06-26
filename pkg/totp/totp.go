// Package totp wraps pquerna/otp for TOTP secret generation and validation.
package totp

import (
	"fmt"

	"github.com/pquerna/otp/totp"
)

// Generate creates a new TOTP secret for accountName under issuer and returns
// the raw secret plus the otpauth:// URI (encode as QR for authenticator apps).
func Generate(issuer, accountName string) (secret, uri string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate totp: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// Validate reports whether passcode is currently valid for secret.
func Validate(passcode, secret string) bool {
	return totp.Validate(passcode, secret)
}
