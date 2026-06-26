// Package id centralises unique ID generation so the strategy is swappable
// in one place. Uses cuid2 to stay consistent with the Better Auth frontend.
package id

import "github.com/nrednav/cuid2"

// New returns a new collision-resistant cuid2 (24 chars, URL-safe).
func New() string {
	return cuid2.Generate()
}
