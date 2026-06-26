package entity

import "time"

// Session mirrors the Better Auth core `session` table. Token is an opaque,
// high-entropy bearer credential.
type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	IPAddress *string
	UserAgent *string
	// ImpersonatedBy is the admin user id when this session is an impersonation.
	ImpersonatedBy *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsExpired reports whether the session is past its expiry.
func (s *Session) IsExpired(now time.Time) bool {
	return now.After(s.ExpiresAt)
}
