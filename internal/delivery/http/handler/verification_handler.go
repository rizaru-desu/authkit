package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"authkit/internal/domain/entity"
	"authkit/internal/usecase"
	"authkit/pkg/response"
)

// VerificationHandler handles email verification and password reset endpoints.
type VerificationHandler struct {
	uc *usecase.VerificationUsecase
}

// NewVerificationHandler creates a VerificationHandler.
func NewVerificationHandler(uc *usecase.VerificationUsecase) *VerificationHandler {
	return &VerificationHandler{uc: uc}
}

type emailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type tokenRequest struct {
	Token string `json:"token" binding:"required"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"    binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// SendEmailVerification handles POST /auth/send-verification-email.
func (h *VerificationHandler) SendEmailVerification(c *gin.Context) {
	var req emailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.uc.RequestEmailVerification(c.Request.Context(), req.Email); err != nil {
		// Do not reveal whether the email exists.
		if errors.Is(err, entity.ErrNotFound) {
			response.OK(c, "if the email exists, a verification link was sent", nil)
			return
		}
		respondError(c, err)
		return
	}
	response.OK(c, "verification email sent", nil)
}

// VerifyEmail handles POST /auth/verify-email.
func (h *VerificationHandler) VerifyEmail(c *gin.Context) {
	var req tokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.uc.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "email verified", nil)
}

// ForgotPassword handles POST /auth/forgot-password.
func (h *VerificationHandler) ForgotPassword(c *gin.Context) {
	var req emailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.uc.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			response.OK(c, "if the email exists, a reset link was sent", nil)
			return
		}
		respondError(c, err)
		return
	}
	response.OK(c, "password reset email sent", nil)
}

// ResetPassword handles POST /auth/reset-password.
func (h *VerificationHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.uc.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "password updated", nil)
}
