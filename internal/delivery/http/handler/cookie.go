package handler

import (
	"github.com/gin-gonic/gin"

	"mns/backend/pkg/config"
)

// writeSessionCookie sets the session token as an httpOnly cookie. When
// persistent is false it becomes a session cookie (cleared on browser close).
func writeSessionCookie(c *gin.Context, cfg config.SessionConfig, token string, persistent bool) {
	maxAge := int(cfg.Expiry.Seconds())
	if !persistent {
		maxAge = 0
	}
	c.SetSameSite(cfg.CookieSameSite)
	c.SetCookie(cfg.CookieName, token, maxAge, "/", cfg.CookieDomain, cfg.CookieSecure, true)
}

// clearSessionCookie expires the session cookie.
func clearSessionCookie(c *gin.Context, cfg config.SessionConfig) {
	c.SetSameSite(cfg.CookieSameSite)
	c.SetCookie(cfg.CookieName, "", -1, "/", cfg.CookieDomain, cfg.CookieSecure, true)
}
