package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/model"

	"net/http"
	"net/http/httptest"
	"testing"
)

type historyFake struct {
	user uuid.UUID
	game model.GameDetail
	err  error
}

func (f historyFake) Profile(_ context.Context, u uuid.UUID) (model.Profile, error) {
	if u != f.user {
		return model.Profile{}, errors.New("wrong user")
	}
	return model.Profile{Nickname: "a", Email: "a@example.com", Rating: 1200}, f.err
}
func (f historyFake) Games(_ context.Context, u uuid.UUID, limit, offset int) ([]model.GameSummary, error) {
	if u != f.user || limit != 20 || offset != 0 {
		return nil, errors.New("bad arguments")
	}
	return []model.GameSummary{}, f.err
}
func (f historyFake) Game(_ context.Context, u, id uuid.UUID) (model.GameDetail, error) {
	if u != f.user || id != f.game.ID {
		return model.GameDetail{}, errors.New("bad arguments")
	}
	return f.game, f.err
}

type verifierFake struct{ user uuid.UUID }

func (v verifierFake) VerifyToken(token string) (uuid.UUID, error) {
	if token != "valid" {
		return uuid.Nil, errors.New("bad token")
	}
	return v.user, nil
}
func TestHistoryRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user, id := uuid.New(), uuid.New()
	h := NewHistoryHandler(historyFake{user: user, game: model.GameDetail{GameSummary: model.GameSummary{ID: id}, Moves: []model.RecordedMove{}}}, verifierFake{user})
	r := gin.New()
	r.GET("/me", h.Profile)
	r.GET("/me/games", h.Games)
	r.GET("/me/games/:id", h.Game)
	cases := []struct {
		name, path, token string
		code              int
		field             string
	}{
		{"profile", "/me", "valid", 200, "nickname"},
		{"unauthorized", "/me", "bad", 401, "error"},
		{"list", "/me/games", "valid", 200, "games"},
		{"invalid limit", "/me/games?limit=0", "valid", 400, "error"},
		{"game", "/me/games/" + id.String(), "valid", 200, "moves"},
		{"invalid game", "/me/games/nope", "valid", 400, "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("status %d body %s", w.Code, w.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if _, ok := body[tc.field]; !ok {
				t.Fatalf("missing %s: %v", tc.field, body)
			}
		})
	}

}
