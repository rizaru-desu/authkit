package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"authkit/internal/delivery/http/middleware"
	"authkit/internal/domain/entity"
	"authkit/internal/usecase"
	"authkit/pkg/config"
)

// AuthHandler handles Better Auth-style sign-up / sign-in / sign-out endpoints.
type AuthHandler struct {
	uc     *usecase.AuthUsecase
	cookie config.SessionConfig
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(uc *usecase.AuthUsecase, cookie config.SessionConfig) *AuthHandler {
	return &AuthHandler{uc: uc, cookie: cookie}
}

// --- requests ---

type signUpRequest struct {
	Name     string  `json:"name"     binding:"required"`
	Email    string  `json:"email"    binding:"required,email"`
	Password string  `json:"password" binding:"required,min=8,max=128"`
	Image    *string `json:"image"`
	// AutoSignIn issues a session on successful sign-up. Defaults to false
	// (the client must send true to be signed in immediately).
	AutoSignIn *bool `json:"autoSignIn"`
}

type signInRequest struct {
	Email      string `json:"email"    binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	Code       string `json:"code"`       // 2FA code when enabled
	RememberMe *bool  `json:"rememberMe"` // default true
}

// SignUpEmail handles POST /auth/sign-up/email. A session is issued only when
// the request sets autoSignIn:true; otherwise the user is created without one.
func (h *AuthHandler) SignUpEmail(c *gin.Context) {
	var req signUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}

	autoSignIn := false
	if req.AutoSignIn != nil {
		autoSignIn = *req.AutoSignIn
	}

	ip, ua := clientMeta(c)
	session, user, err := h.uc.SignUp(c.Request.Context(), usecase.SignUpInput{
		Name:       req.Name,
		Email:      req.Email,
		Password:   req.Password,
		Image:      req.Image,
		AutoSignIn: autoSignIn,
		IPAddress:  ip,
		UserAgent:  ua,
	})
	if err != nil {
		authError(c, err)
		return
	}

	// No session when autoSignIn is false or email verification is required.
	if session == nil {
		c.JSON(http.StatusOK, gin.H{
			"token": nil,
			"user":  toAuthUser(user),
		})
		return
	}

	writeSessionCookie(c, h.cookie, session.Token, true)
	c.JSON(http.StatusOK, gin.H{
		"token": session.Token,
		"user":  toAuthUser(user),
	})
}

// SignInEmail handles POST /auth/sign-in/email.
func (h *AuthHandler) SignInEmail(c *gin.Context) {
	var req signInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}

	ip, ua := clientMeta(c)
	session, user, err := h.uc.Login(c.Request.Context(), usecase.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		Code:      req.Code,
		IPAddress: ip,
		UserAgent: ua,
	})
	if err != nil {
		authError(c, err)
		return
	}

	persistent := req.RememberMe == nil || *req.RememberMe
	writeSessionCookie(c, h.cookie, session.Token, persistent)
	c.JSON(http.StatusOK, gin.H{
		"redirect": false,
		"token":    session.Token,
		"user":     toAuthUser(user),
	})
}

// SignOut handles POST /auth/sign-out.
func (h *AuthHandler) SignOut(c *gin.Context) {
	session, ok := middleware.CurrentSession(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	if err := h.uc.Logout(c.Request.Context(), session.Token); err != nil {
		authError(c, err)
		return
	}
	clearSessionCookie(c, h.cookie)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type revokeOwnSessionRequest struct {
	Token string `json:"token" binding:"required"`
}

// ListSessions handles GET /auth/list-sessions (current user's sessions).
func (h *AuthHandler) ListSessions(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	sessions, err := h.uc.ListSessions(c.Request.Context(), user.ID)
	if err != nil {
		authError(c, err)
		return
	}
	out := make([]gin.H, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, toSessionDTO(s))
	}
	c.JSON(http.StatusOK, gin.H{"sessions": out})
}

// RevokeSession handles POST /auth/revoke-session (revoke one own session).
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	var req revokeOwnSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	if err := h.uc.RevokeSession(c.Request.Context(), user.ID, req.Token); err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RevokeSessions handles POST /auth/revoke-sessions (revoke all own sessions).
func (h *AuthHandler) RevokeSessions(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	if err := h.uc.RevokeAllSessions(c.Request.Context(), user.ID); err != nil {
		authError(c, err)
		return
	}
	clearSessionCookie(c, h.cookie)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RevokeOtherSessions handles POST /auth/revoke-other-sessions.
func (h *AuthHandler) RevokeOtherSessions(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	session, _ := middleware.CurrentSession(c)
	if err := h.uc.RevokeOtherSessions(c.Request.Context(), user.ID, session.Token); err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetSession handles GET /auth/get-session (Better Auth style).
func (h *AuthHandler) GetSession(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	session, _ := middleware.CurrentSession(c)
	c.JSON(http.StatusOK, gin.H{
		"user": toAuthUser(user),
		"session": gin.H{
			"id":        session.ID,
			"expiresAt": session.ExpiresAt.Format(time.RFC3339),
		},
	})
}

// --- helpers ---

// toAuthUser builds the Better Auth-shaped (camelCase) user object, including
// admin-plugin fields (role, banned, banReason, banExpires).
func toAuthUser(u *entity.User) gin.H {
	var banExpires *string
	if u.BanExpires != nil {
		s := u.BanExpires.Format(time.RFC3339)
		banExpires = &s
	}
	return gin.H{
		"id":               u.ID,
		"email":            u.Email,
		"name":             u.Name,
		"image":            u.Image,
		"emailVerified":    u.EmailVerified,
		"role":             string(u.Role),
		"twoFactorEnabled": u.TwoFactorEnabled,
		"banned":           u.Banned,
		"banReason":        u.BanReason,
		"banExpires":       banExpires,
		"createdAt":        u.CreatedAt.Format(time.RFC3339),
		"updatedAt":        u.UpdatedAt.Format(time.RFC3339),
	}
}

func authBadRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "code": "BAD_REQUEST"})
}

// authError maps domain errors to Better Auth-style { message, code } responses.
func authError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entity.ErrInvalidCredential):
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error(), "code": "INVALID_EMAIL_OR_PASSWORD"})
	case errors.Is(err, entity.ErrEmailTaken):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error(), "code": "USER_ALREADY_EXISTS"})
	case errors.Is(err, entity.ErrTwoFactorRequired):
		c.JSON(http.StatusUnauthorized, gin.H{
			"message":           err.Error(),
			"code":              "TWO_FACTOR_REQUIRED",
			"twoFactorRequired": true,
		})
	case errors.Is(err, entity.ErrInvalidCode):
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error(), "code": "INVALID_TWO_FACTOR_CODE"})
	case errors.Is(err, entity.ErrBanned):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error(), "code": "BANNED_USER"})
	case errors.Is(err, entity.ErrEmailNotVerified):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error(), "code": "EMAIL_NOT_VERIFIED"})
	case errors.Is(err, entity.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"message": err.Error(), "code": "FORBIDDEN"})
	case errors.Is(err, entity.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error(), "code": "NOT_FOUND"})
	case errors.Is(err, entity.ErrNotImpersonating), errors.Is(err, entity.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "code": "BAD_REQUEST"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error", "code": "INTERNAL_ERROR"})
	}
}
