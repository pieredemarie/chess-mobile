package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/service"
)

type MatchmakingHandler struct {
	Matchmaker *service.Matchmaker
	Auth       TokenVerifier
}

func (h *MatchmakingHandler) user(c *gin.Context) (uuid.UUID, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	user, err := h.Auth.VerifyToken(strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	return user, true
}

func (h *MatchmakingHandler) Search(c *gin.Context) {
	user, ok := h.user(c)
	if !ok {
		return
	}
	match, err := h.Matchmaker.Search(c.Request.Context(), user)
	if errors.Is(err, service.ErrAlreadySearching) {
		c.JSON(http.StatusConflict, gin.H{"error": "already searching"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if match == nil {
		c.JSON(http.StatusAccepted, gin.H{"status": "searching"})
		return
	}
	c.JSON(http.StatusCreated, match)
}

func (h *MatchmakingHandler) Status(c *gin.Context) {
	user, ok := h.user(c)
	if !ok {
		return
	}
	match, waiting := h.Matchmaker.Status(user)
	if match != nil {
		c.JSON(http.StatusOK, match)
		return
	}
	if waiting {
		c.JSON(http.StatusOK, gin.H{"status": "searching"})
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not searching"})
}

func (h *MatchmakingHandler) Cancel(c *gin.Context) {
	user, ok := h.user(c)
	if !ok {
		return
	}
	if err := h.Matchmaker.Cancel(user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not searching"})
		return
	}
	c.Status(http.StatusNoContent)
}
