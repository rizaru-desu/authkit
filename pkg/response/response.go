// Package response provides standardised JSON HTTP response helpers.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// OK sends a 200 JSON response.
func OK(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, envelope{Success: true, Message: message, Data: data})
}

// Created sends a 201 JSON response.
func Created(c *gin.Context, message string, data any) {
	c.JSON(http.StatusCreated, envelope{Success: true, Message: message, Data: data})
}

// BadRequest sends a 400 JSON response.
func BadRequest(c *gin.Context, err string) {
	c.JSON(http.StatusBadRequest, envelope{Success: false, Error: err})
}

// Unauthorized sends a 401 JSON response.
func Unauthorized(c *gin.Context, err string) {
	c.JSON(http.StatusUnauthorized, envelope{Success: false, Error: err})
}

// Forbidden sends a 403 JSON response.
func Forbidden(c *gin.Context, err string) {
	c.JSON(http.StatusForbidden, envelope{Success: false, Error: err})
}

// NotFound sends a 404 JSON response.
func NotFound(c *gin.Context, err string) {
	c.JSON(http.StatusNotFound, envelope{Success: false, Error: err})
}

// InternalError sends a 500 JSON response.
func InternalError(c *gin.Context, err string) {
	c.JSON(http.StatusInternalServerError, envelope{Success: false, Error: err})
}
