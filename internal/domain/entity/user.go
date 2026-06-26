// Package entity contains domain models. No dependencies on outer layers.
package entity

import "time"

// Role defines user access level (Better Auth "additional field").
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// User mirrors the Better Auth core `user` table (snake_case in DB).
// Password and credentials live in Account, not here.
type User struct {
	ID               string
	Name             string
	Email            string
	EmailVerified    bool
	Image            *string
	Role             Role
	TwoFactorEnabled bool
	Banned           bool
	BanReason        *string
	BanExpires       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsBanned reports whether the user is currently banned (expired bans don't count).
func (u *User) IsBanned(now time.Time) bool {
	if !u.Banned {
		return false
	}
	if u.BanExpires != nil && now.After(*u.BanExpires) {
		return false
	}
	return true
}
