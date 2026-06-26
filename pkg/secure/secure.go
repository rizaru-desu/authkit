// Package secure provides cryptographic helpers: high-entropy tokens and
// hashed one-time backup codes.
package secure

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// Token returns a URL-safe, 256-bit random token suitable for session and
// verification tokens.
func Token() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash returns the hex-encoded SHA-256 of s. Used to store tokens/backup codes
// so the database never holds the plaintext.
func Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Equal compares two strings in constant time.
func Equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateBackupCodes returns n human-friendly codes (plaintext, shown once)
// alongside their SHA-256 hashes (stored).
func GenerateBackupCodes(n int) (plain, hashed []string, err error) {
	plain = make([]string, 0, n)
	hashed = make([]string, 0, n)
	for i := 0; i < n; i++ {
		b := make([]byte, 5) // 10 hex chars
		if _, err = rand.Read(b); err != nil {
			return nil, nil, fmt.Errorf("read random: %w", err)
		}
		code := hex.EncodeToString(b)
		plain = append(plain, code)
		hashed = append(hashed, Hash(code))
	}
	return plain, hashed, nil
}
