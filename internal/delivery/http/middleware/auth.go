package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"mns/backend/internal/domain/entity"
	"mns/backend/internal/usecase"
	"mns/backend/pkg/response"
)

const (
	ctxUserKey    = "auth_user"
	ctxSessionKey = "auth_session"
)

// RequireAuth validates the session token (Bearer header first, then the
// session cookie) and loads the user into the request context.
func RequireAuth(auth *usecase.AuthUsecase, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c, cookieName)
		if token == "" {
			response.Unauthorized(c, "missing session token")
			c.Abort()
			return
		}
		user, session, err := auth.ValidateSession(c.Request.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, entity.ErrUnauthorized), errors.Is(err, entity.ErrSessionExpired):
				response.Unauthorized(c, err.Error())
			default:
				response.InternalError(c, "authentication error")
			}
			c.Abort()
			return
		}
		c.Set(ctxUserKey, user)
		c.Set(ctxSessionKey, session)
		c.Next()
	}
}

// CurrentUser returns the user set by RequireAuth.
func CurrentUser(c *gin.Context) (*entity.User, bool) {
	v, ok := c.Get(ctxUserKey)
	if !ok {
		return nil, false
	}
	u, ok := v.(*entity.User)
	return u, ok
}

// CurrentSession returns the session set by RequireAuth.
func CurrentSession(c *gin.Context) (*entity.Session, bool) {
	v, ok := c.Get(ctxSessionKey)
	if !ok {
		return nil, false
	}
	s, ok := v.(*entity.Session)
	return s, ok
}

// extractToken reads the session token from the Authorization: Bearer header,
// falling back to the session cookie (web clients).
func extractToken(c *gin.Context, cookieName string) string {
	if h := c.GetHeader("Authorization"); h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	if token, err := c.Cookie(cookieName); err == nil {
		return token
	}
	return ""
}
