package handler

import (
	"github.com/gin-gonic/gin"

	"authkit/internal/delivery/http/middleware"
	"authkit/internal/usecase"
	"authkit/pkg/response"
)

// TwoFactorHandler handles 2FA enrolment endpoints (all require auth).
type TwoFactorHandler struct {
	uc *usecase.TwoFactorUsecase
}

// NewTwoFactorHandler creates a TwoFactorHandler.
func NewTwoFactorHandler(uc *usecase.TwoFactorUsecase) *TwoFactorHandler {
	return &TwoFactorHandler{uc: uc}
}

type passwordRequest struct {
	Password string `json:"password" binding:"required"`
}

type codeRequest struct {
	Code string `json:"code" binding:"required"`
}

// Enable handles POST /auth/2fa/enable.
func (h *TwoFactorHandler) Enable(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "not authenticated")
		return
	}
	var req passwordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	res, err := h.uc.Enable(c.Request.Context(), user.ID, req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "scan the QR / enter the URI in your authenticator, then verify", gin.H{
		"totp_uri":     res.TOTPURI,
		"backup_codes": res.BackupCodes,
	})
}

// Verify handles POST /auth/2fa/verify (confirm enrolment).
func (h *TwoFactorHandler) Verify(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "not authenticated")
		return
	}
	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.uc.ConfirmEnable(c.Request.Context(), user.ID, req.Code); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "two-factor authentication enabled", nil)
}

// Disable handles POST /auth/2fa/disable.
func (h *TwoFactorHandler) Disable(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "not authenticated")
		return
	}
	var req passwordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.uc.Disable(c.Request.Context(), user.ID, req.Password); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "two-factor authentication disabled", nil)
}
