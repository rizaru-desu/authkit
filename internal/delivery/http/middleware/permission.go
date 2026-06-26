package middleware

import (
	"github.com/gin-gonic/gin"

	"mns/backend/pkg/access"
	"mns/backend/pkg/response"
)

// RequirePermission allows the request only if the current user's role is
// granted the given action on the resource. Must run after RequireAuth.
func RequirePermission(ac *access.Controller, resource, action string) gin.HandlerFunc {
	required := map[string][]string{resource: {action}}
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			response.Unauthorized(c, "not authenticated")
			c.Abort()
			return
		}
		if !ac.Check(string(user.Role), required) {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}
