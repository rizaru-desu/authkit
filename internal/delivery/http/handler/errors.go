package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"authkit/internal/domain/entity"
	"authkit/pkg/response"
)

// respondError maps domain sentinel errors to HTTP responses.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrInvalidCredential):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, entity.ErrTwoFactorRequired):
		c.JSON(http.StatusUnauthorized, gin.H{
			"success":             false,
			"error":               err.Error(),
			"two_factor_required": true,
		})
	case errors.Is(err, entity.ErrUnauthorized),
		errors.Is(err, entity.ErrSessionExpired),
		errors.Is(err, entity.ErrInvalidCode),
		errors.Is(err, entity.ErrTokenExpired):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, entity.ErrNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, entity.ErrEmailTaken):
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, entity.ErrTwoFactorNotSet), errors.Is(err, entity.ErrInvalidInput):
		response.BadRequest(c, err.Error())
	default:
		response.InternalError(c, err.Error())
	}
}

// clientMeta extracts IP address and user agent as optional strings.
func clientMeta(c *gin.Context) (ip, ua *string) {
	if v := c.ClientIP(); v != "" {
		ip = &v
	}
	if v := c.Request.UserAgent(); v != "" {
		ua = &v
	}
	return ip, ua
}
