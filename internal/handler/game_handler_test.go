package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/chesslogic"
	"github.com/pieremarie/chessmobile/internal/hub"
)

type fakeVerifier struct {
	id  uuid.UUID
	err error
}

func (v fakeVerifier) VerifyToken(string) (uuid.UUID, error) { return v.id, v.err }

type fakeGameStore struct{ players hub.Players }

func (s fakeGameStore) Players(context.Context, uuid.UUID) (hub.Players, error) {
	return s.players, nil
}
func (fakeGameStore) SaveMove(context.Context, uuid.UUID, string, string) error { return nil }
func (fakeGameStore) SaveResult(context.Context, uuid.UUID, string) error       { return nil }
func TestWebSocketPreUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	white := uuid.New()
	id := uuid.New()
	cases := []struct {
		name, auth string
		user       uuid.UUID
		want       int
	}{
		{"missing token", "", white, 401}, {"outsider", "Bearer valid", uuid.New(), 403}, {"bad origin", "Bearer valid", white, 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hub.New(fakeGameStore{hub.Players{White: white, Black: uuid.New()}}, func(uuid.UUID) (chesslogic.Game, error) { return chesslogic.New(), nil })
			r := gin.New()
			r.GET("/ws/games/:id", NewGameHandler(h, fakeVerifier{id: tc.user}).Serve)
			req := httptest.NewRequest(http.MethodGet, "/ws/games/"+id.String(), nil)
			req.Header.Set("Authorization", tc.auth)
			if tc.name == "bad origin" {
				req.Header.Set("Origin", "https://evil.example")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d want=%d", w.Code, tc.want)
			}
		})
	}
}
