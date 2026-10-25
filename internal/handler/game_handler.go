package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pieremarie/chessmobile/internal/hub"
)

type TokenVerifier interface {
	VerifyToken(string) (uuid.UUID, error)
}
type GameHandler struct {
	hub      *hub.Hub
	auth     TokenVerifier
	upgrader websocket.Upgrader
}

func NewGameHandler(h *hub.Hub, a TokenVerifier) *GameHandler {
	return &GameHandler{hub: h, auth: a, upgrader: websocket.Upgrader{CheckOrigin: checkOrigin}}
}
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host && (u.Scheme == "http" || u.Scheme == "https")
}
func (h *GameHandler) Serve(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	user, err := h.auth.VerifyToken(strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid game id"})
		return
	}
	client, err := h.hub.Join(c.Request.Context(), id, user)
	if err != nil {
		if errors.Is(err, hub.ErrForbidden) {
			c.JSON(403, gin.H{"error": "forbidden"})
		} else if errors.Is(err, hub.ErrFinished) {
			c.JSON(409, gin.H{"error": "game finished"})
		} else if errors.Is(err, hub.ErrNotFound) {
			c.JSON(404, gin.H{"error": "game not found"})
		} else {
			c.JSON(500, gin.H{"error": "internal server error"})
		}
		return
	}
	if !checkOrigin(c.Request) {
		h.hub.Leave(id, client)
		c.JSON(403, gin.H{"error": "forbidden origin"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.hub.Leave(id, client)
		return
	}
	defer func() { _ = conn.Close() }()
	defer h.hub.Leave(id, client)
	const pongWait = 60 * time.Second
	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = conn.Close() }()
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteJSON(msg); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		closeNow, err := h.hub.Handle(c.Request.Context(), id, client, data)
		if closeNow {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInvalidFramePayloadData, "invalid message"), time.Now().Add(time.Second))
			break
		}
		if err != nil {
			break
		}
	}
	_ = conn.Close()
	<-done
}
