// Package handler contains HTTP handlers for the delivery layer.
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"mns/backend/internal/domain/entity"
	"mns/backend/internal/usecase"
	"mns/backend/pkg/response"
)

// UserHandler handles HTTP requests for user resources.
type UserHandler struct {
	uc *usecase.UserUsecase
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(uc *usecase.UserUsecase) *UserHandler {
	return &UserHandler{uc: uc}
}

// GetByID handles GET /users/:id.
func (h *UserHandler) GetByID(c *gin.Context) {
	user, err := h.uc.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "ok", toUserResponse(user))
}

// List handles GET /users.
func (h *UserHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, err := h.uc.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	out := make([]userResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toUserResponse(u))
	}
	response.OK(c, "ok", out)
}

// Delete handles DELETE /users/:id.
func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.uc.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, "user deleted", nil)
}

// userResponse is the public representation of a user (no password).
type userResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"email_verified"`
	Image         *string `json:"image"`
	Role          string  `json:"role"`
	CreatedAt     string  `json:"created_at"`
}

func toUserResponse(u *entity.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Name:          u.Name,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Image:         u.Image,
		Role:          string(u.Role),
		CreatedAt:     u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
