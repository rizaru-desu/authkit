package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"mns/backend/internal/delivery/http/middleware"
	"mns/backend/internal/domain/entity"
	"mns/backend/internal/usecase"
	"mns/backend/pkg/config"
)

// AdminHandler implements the Better Auth admin plugin endpoints.
type AdminHandler struct {
	uc     *usecase.AdminUsecase
	cookie config.SessionConfig
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(uc *usecase.AdminUsecase, cookie config.SessionConfig) *AdminHandler {
	return &AdminHandler{uc: uc, cookie: cookie}
}

// --- requests ---

type adminCreateUserRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Name     string `json:"name"     binding:"required"`
	Role     string `json:"role"`
}

type userIDRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type setRoleRequest struct {
	UserID string `json:"userId" binding:"required"`
	Role   string `json:"role"   binding:"required"`
}

type setPasswordRequest struct {
	UserID      string `json:"userId"      binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
}

type banUserRequest struct {
	UserID       string  `json:"userId"       binding:"required"`
	BanReason    *string `json:"banReason"`
	BanExpiresIn int64   `json:"banExpiresIn"`
}

type revokeSessionRequest struct {
	SessionToken string `json:"sessionToken" binding:"required"`
}

type hasPermissionRequest struct {
	UserID      string              `json:"userId"`
	Role        string              `json:"role"`
	Permission  map[string][]string `json:"permission"`
	Permissions map[string][]string `json:"permissions"`
}

// CreateUser handles POST /admin/create-user.
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req adminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	user, err := h.uc.CreateUser(c.Request.Context(), usecase.CreateUserInput{
		Name: req.Name, Email: req.Email, Password: req.Password, Role: entity.Role(req.Role),
	})
	if err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": toAuthUser(user)})
}

// ListUsers handles GET /admin/list-users.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	search := c.Query("searchValue")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	res, err := h.uc.ListUsers(c.Request.Context(), search, limit, offset)
	if err != nil {
		authError(c, err)
		return
	}
	users := make([]gin.H, 0, len(res.Users))
	for _, u := range res.Users {
		users = append(users, toAuthUser(u))
	}
	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"total":  res.Total,
		"limit":  res.Limit,
		"offset": res.Offset,
	})
}

// SetRole handles POST /admin/set-role.
func (h *AdminHandler) SetRole(c *gin.Context) {
	var req setRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	user, err := h.uc.SetRole(c.Request.Context(), req.UserID, entity.Role(req.Role))
	if err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": toAuthUser(user)})
}

// SetUserPassword handles POST /admin/set-user-password.
func (h *AdminHandler) SetUserPassword(c *gin.Context) {
	var req setPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	if err := h.uc.SetUserPassword(c.Request.Context(), req.UserID, req.NewPassword); err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// BanUser handles POST /admin/ban-user.
func (h *AdminHandler) BanUser(c *gin.Context) {
	var req banUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	user, err := h.uc.BanUser(c.Request.Context(), req.UserID, req.BanReason, req.BanExpiresIn)
	if err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": toAuthUser(user)})
}

// UnbanUser handles POST /admin/unban-user.
func (h *AdminHandler) UnbanUser(c *gin.Context) {
	var req userIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	user, err := h.uc.UnbanUser(c.Request.Context(), req.UserID)
	if err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": toAuthUser(user)})
}

// ListUserSessions handles POST /admin/list-user-sessions.
func (h *AdminHandler) ListUserSessions(c *gin.Context) {
	var req userIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	sessions, err := h.uc.ListUserSessions(c.Request.Context(), req.UserID)
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

// RevokeUserSession handles POST /admin/revoke-user-session.
func (h *AdminHandler) RevokeUserSession(c *gin.Context) {
	var req revokeSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	if err := h.uc.RevokeUserSession(c.Request.Context(), req.SessionToken); err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RevokeUserSessions handles POST /admin/revoke-user-sessions.
func (h *AdminHandler) RevokeUserSessions(c *gin.Context) {
	var req userIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	if err := h.uc.RevokeUserSessions(c.Request.Context(), req.UserID); err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RemoveUser handles POST /admin/remove-user.
func (h *AdminHandler) RemoveUser(c *gin.Context) {
	var req userIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	if err := h.uc.RemoveUser(c.Request.Context(), req.UserID); err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Impersonate handles POST /admin/impersonate-user.
func (h *AdminHandler) Impersonate(c *gin.Context) {
	admin, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	var req userIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}
	ip, ua := clientMeta(c)
	session, user, err := h.uc.Impersonate(c.Request.Context(), admin.ID, req.UserID, ip, ua)
	if err != nil {
		authError(c, err)
		return
	}
	// Switch the browser session to the impersonation token.
	writeSessionCookie(c, h.cookie, session.Token, false)
	c.JSON(http.StatusOK, gin.H{
		"token":   session.Token,
		"session": toSessionDTO(session),
		"user":    toAuthUser(user),
	})
}

// StopImpersonating handles POST /admin/stop-impersonating.
func (h *AdminHandler) StopImpersonating(c *gin.Context) {
	session, ok := middleware.CurrentSession(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	if err := h.uc.StopImpersonating(c.Request.Context(), session); err != nil {
		authError(c, err)
		return
	}
	clearSessionCookie(c, h.cookie)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HasPermission handles POST /admin/has-permission. Any authenticated user may
// check their own role's permissions; checking another user/role requires the
// caller to be able to list users (admin-level).
func (h *AdminHandler) HasPermission(c *gin.Context) {
	caller, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "not authenticated", "code": "UNAUTHORIZED"})
		return
	}
	var req hasPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		authBadRequest(c, err)
		return
	}

	perms := req.Permissions
	if perms == nil {
		perms = req.Permission
	}

	// Checking someone else's permissions is privileged.
	checkingOther := req.UserID != "" || req.Role != ""
	if checkingOther && caller.Role != entity.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"message": "forbidden", "code": "FORBIDDEN"})
		return
	}

	userID, role := req.UserID, req.Role
	if !checkingOther {
		role = string(caller.Role) // check the caller's own permissions
	}

	granted, err := h.uc.HasPermission(c.Request.Context(), userID, role, perms)
	if err != nil {
		authError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": granted})
}

// toSessionDTO builds a Better Auth-shaped session object.
func toSessionDTO(s *entity.Session) gin.H {
	return gin.H{
		"id":             s.ID,
		"token":          s.Token,
		"userId":         s.UserID,
		"expiresAt":      s.ExpiresAt.Format(time.RFC3339),
		"ipAddress":      s.IPAddress,
		"userAgent":      s.UserAgent,
		"impersonatedBy": s.ImpersonatedBy,
		"createdAt":      s.CreatedAt.Format(time.RFC3339),
	}
}
