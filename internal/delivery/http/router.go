// Package http wires Gin routes to handlers.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"authkit/internal/delivery/http/handler"
	"authkit/internal/delivery/http/middleware"
	"authkit/internal/usecase"
	"authkit/pkg/access"
)

// Deps holds everything the router needs to register routes.
type Deps struct {
	Log                 *zap.Logger
	GlobalLimiter       *middleware.RateLimiter
	AuthLimiter         *middleware.RateLimiter
	Auth                *usecase.AuthUsecase
	CookieName          string
	TrustedOrigins      []string
	AccessControl       *access.Controller
	UserHandler         *handler.UserHandler
	AuthHandler         *handler.AuthHandler
	TwoFactorHandler    *handler.TwoFactorHandler
	VerificationHandler *handler.VerificationHandler
	AdminHandler        *handler.AdminHandler
}

// Router wraps the configured gin engine.
type Router struct {
	engine *gin.Engine
}

// NewRouter creates and configures the Gin router.
func NewRouter(d Deps) *Router {
	engine := gin.New()
	engine.Use(
		middleware.Logger(d.Log),
		middleware.Recovery(d.Log),
		middleware.CORS(d.TrustedOrigins),
		d.GlobalLimiter.Middleware(),
	)

	requireAuth := middleware.RequireAuth(d.Auth, d.CookieName)
	authLimit := d.AuthLimiter.Middleware()

	// Better Auth-compatible endpoints (default basePath "/api/auth").
	auth := engine.Group("/api/auth")
	{
		auth.POST("/sign-up/email", authLimit, d.AuthHandler.SignUpEmail)
		auth.POST("/sign-in/email", authLimit, d.AuthHandler.SignInEmail)
		auth.POST("/sign-out", requireAuth, d.AuthHandler.SignOut)
		auth.GET("/get-session", requireAuth, d.AuthHandler.GetSession)

		// Self-service session management (current user).
		auth.GET("/list-sessions", requireAuth, d.AuthHandler.ListSessions)
		auth.POST("/revoke-session", requireAuth, d.AuthHandler.RevokeSession)
		auth.POST("/revoke-sessions", requireAuth, d.AuthHandler.RevokeSessions)
		auth.POST("/revoke-other-sessions", requireAuth, d.AuthHandler.RevokeOtherSessions)

		auth.POST("/send-verification-email", authLimit, d.VerificationHandler.SendEmailVerification)
		auth.POST("/verify-email", d.VerificationHandler.VerifyEmail)
		auth.POST("/request-password-reset", authLimit, d.VerificationHandler.ForgotPassword)
		auth.POST("/reset-password", authLimit, d.VerificationHandler.ResetPassword)

		tfa := auth.Group("/two-factor", requireAuth)
		{
			tfa.POST("/enable", d.TwoFactorHandler.Enable)
			tfa.POST("/verify-totp", d.TwoFactorHandler.Verify)
			tfa.POST("/disable", d.TwoFactorHandler.Disable)
		}

		// Permission-gated admin endpoints (role must grant resource:action).
		perm := func(resource, action string) gin.HandlerFunc {
			return middleware.RequirePermission(d.AccessControl, resource, action)
		}
		admin := auth.Group("/admin", requireAuth)
		{
			admin.POST("/create-user", perm(access.ResourceUser, access.ActionCreate), d.AdminHandler.CreateUser)
			admin.GET("/list-users", perm(access.ResourceUser, access.ActionList), d.AdminHandler.ListUsers)
			admin.POST("/set-role", perm(access.ResourceUser, access.ActionSetRole), d.AdminHandler.SetRole)
			admin.POST("/set-user-password", perm(access.ResourceUser, access.ActionSetPassword), d.AdminHandler.SetUserPassword)
			admin.POST("/ban-user", perm(access.ResourceUser, access.ActionBan), d.AdminHandler.BanUser)
			admin.POST("/unban-user", perm(access.ResourceUser, access.ActionBan), d.AdminHandler.UnbanUser)
			admin.POST("/list-user-sessions", perm(access.ResourceSession, access.ActionList), d.AdminHandler.ListUserSessions)
			admin.POST("/revoke-user-session", perm(access.ResourceSession, access.ActionRevoke), d.AdminHandler.RevokeUserSession)
			admin.POST("/revoke-user-sessions", perm(access.ResourceSession, access.ActionRevoke), d.AdminHandler.RevokeUserSessions)
			admin.POST("/impersonate-user", perm(access.ResourceUser, access.ActionImpersonate), d.AdminHandler.Impersonate)
			admin.POST("/remove-user", perm(access.ResourceUser, access.ActionDelete), d.AdminHandler.RemoveUser)

			// Auth-only (no extra permission): a user checks their own perms;
			// an impersonated user (role=user) can stop impersonating.
			admin.POST("/has-permission", d.AdminHandler.HasPermission)
			admin.POST("/stop-impersonating", d.AdminHandler.StopImpersonating)
		}
	}

	// Application API (not part of Better Auth).
	v1 := engine.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		users := v1.Group("/users", requireAuth)
		{
			users.GET("", d.UserHandler.List)
			users.GET("/:id", d.UserHandler.GetByID)
			users.DELETE("/:id", d.UserHandler.Delete)
		}
	}

	return &Router{engine: engine}
}

// Engine returns the underlying gin.Engine.
func (r *Router) Engine() *gin.Engine {
	return r.engine
}
