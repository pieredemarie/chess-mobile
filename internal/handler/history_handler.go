package handler

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/model"
	"github.com/pieremarie/chessmobile/internal/repository"
	"net/http"
	"strconv"
	"strings"
)

type HistoryService interface {
	Profile(context.Context, uuid.UUID) (model.Profile, error)
	Games(context.Context, uuid.UUID, int, int) ([]model.GameSummary, error)
	Game(context.Context, uuid.UUID, uuid.UUID) (model.GameDetail, error)
}
type HistoryHandler struct {
	service HistoryService
	auth    TokenVerifier
}

func NewHistoryHandler(s HistoryService, a TokenVerifier) *HistoryHandler {
	return &HistoryHandler{service: s, auth: a}
}
func (h *HistoryHandler) user(c *gin.Context) (uuid.UUID, bool) {
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	id, err := h.auth.VerifyToken(parts[1])
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	return id, true
}
func (h *HistoryHandler) Profile(c *gin.Context) {
	user, ok := h.user(c)
	if !ok {
		return
	}
	p, err := h.service.Profile(c.Request.Context(), user)
	if errors.Is(err, repository.ErrUserNotFound) {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(200, p)
}
func (h *HistoryHandler) Games(c *gin.Context) {
	user, ok := h.user(c)
	if !ok {
		return
	}
	limit, offset := 20, 0
	var err error
	if raw := c.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			c.JSON(400, gin.H{"error": "invalid limit"})
			return
		}
	}
	if raw := c.Query("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			c.JSON(400, gin.H{"error": "invalid offset"})
			return
		}
	}
	games, err := h.service.Games(c.Request.Context(), user, limit, offset)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(200, gin.H{"games": games, "limit": limit, "offset": offset})
}
func (h *HistoryHandler) Game(c *gin.Context) {
	user, ok := h.user(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid game id"})
		return
	}
	game, err := h.service.Game(c.Request.Context(), user, id)
	if errors.Is(err, repository.ErrGameNotFound) {
		c.JSON(404, gin.H{"error": "game not found"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(200, game)
}
