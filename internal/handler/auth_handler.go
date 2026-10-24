package handler

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/service"
	"net/http"
)

type AuthService interface {
	Register(context.Context, string, string, string) (uuid.UUID, error)
	Login(context.Context, string, string) (string, error)
}
type AuthHandler struct{ service AuthService }

func NewAuthHandler(s AuthService) *AuthHandler { return &AuthHandler{s} }
func (h *AuthHandler) Register(c *gin.Context) {
	var r struct {
		Nickname string
		Email    string
		Password string
	}
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	id, err := h.service.Register(c.Request.Context(), r.Nickname, r.Email, r.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			c.JSON(400, gin.H{"error": "invalid input"})
		case errors.Is(err, service.ErrEmailExists):
			c.JSON(409, gin.H{"error": "email already exists"})
		default:
			c.JSON(500, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}
func (h *AuthHandler) Login(c *gin.Context) {
	var r struct {
		Email    string
		Password string
	}
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	token, err := h.service.Login(c.Request.Context(), r.Email, r.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(401, gin.H{"error": "invalid credentials"})
		} else {
			c.JSON(500, gin.H{"error": "internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": token})
}
