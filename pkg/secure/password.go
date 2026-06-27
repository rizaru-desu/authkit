package secure

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2Params holds the cost parameters used when hashing new passwords.
// They are also encoded into every hash, so verification reads the parameters
// from the stored hash rather than these defaults — changing them only affects
// passwords hashed afterwards. Defaults follow OWASP's argon2id guidance.
type argon2Params struct {
	memory      uint32 // KiB
	iterations  uint32
	parallelism uint8
	saltLen     uint32
	keyLen      uint32
}

var defaultArgon2Params = argon2Params{
	memory:      64 * 1024, // 64 MiB
	iterations:  3,
	parallelism: 2,
	saltLen:     16,
	keyLen:      32,
}

// ErrInvalidHash is returned when a stored hash is not a valid PHC-encoded
// argon2id string.
var ErrInvalidHash = errors.New("invalid argon2 hash")

// ErrMismatchedPassword is returned by VerifyPassword when the password does
// not match the hash.
var ErrMismatchedPassword = errors.New("password does not match")

// HashPassword hashes a plaintext password with argon2id and returns it in the
// standard PHC string format ($argon2id$v=19$m=...,t=...,p=...$salt$hash).
func HashPassword(password string) (string, error) {
	p := defaultArgon2Params
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLen)

	b64 := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.iterations, p.parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches the PHC-encoded argon2id
// hash. It returns nil on a match, ErrMismatchedPassword on a mismatch, or
// ErrInvalidHash if the hash cannot be parsed.
func VerifyPassword(encodedHash, password string) error {
	p, salt, key, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return err
	}

	other := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, uint32(len(key)))
	if subtle.ConstantTimeCompare(key, other) == 1 {
		return nil
	}
	return ErrMismatchedPassword
}

func decodeArgon2Hash(encoded string) (p argon2Params, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return p, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return p, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return p, nil, nil, fmt.Errorf("%w: unsupported version %d", ErrInvalidHash, version)
	}

	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism); err != nil {
		return p, nil, nil, ErrInvalidHash
	}

	b64 := base64.RawStdEncoding
	if salt, err = b64.DecodeString(parts[4]); err != nil {
		return p, nil, nil, ErrInvalidHash
	}
	if key, err = b64.DecodeString(parts[5]); err != nil {
		return p, nil, nil, ErrInvalidHash
	}
	return p, salt, key, nil
}
