package hub

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/chesslogic"
)

type fakeGame struct{ moves []string }

func (g *fakeGame) Move(m string) error {
	if m != "e2e4" {
		return ErrForbidden
	}
	g.moves = append(g.moves, m)
	return nil
}
func (g *fakeGame) FEN() string       { return "fen" }
func (g *fakeGame) History() []string { return g.moves }
func (g *fakeGame) Status() chesslogic.Status {
	turn := "w"
	if len(g.moves) > 0 {
		turn = "b"
	}
	return chesslogic.Status{Turn: turn, Outcome: "*"}
}
func (g *fakeGame) ClaimThreefoldRepetition() error { return nil }

type fakeStore struct {
	players Players
	moves   []string
	result  string
}

func (s *fakeStore) Players(context.Context, uuid.UUID) (Players, error) { return s.players, nil }
func (s *fakeStore) SaveMove(_ context.Context, _ uuid.UUID, m, _ string) error {
	s.moves = append(s.moves, m)
	return nil
}
func (s *fakeStore) SaveResult(_ context.Context, _ uuid.UUID, r string) error {
	s.result = r
	return nil
}
func receive(t *testing.T, c *Client) Message {
	t.Helper()
	select {
	case m := <-c.Send:
		return m
	case <-time.After(time.Second):
		t.Fatal("no message")
		return Message{}
	}
}
func TestGameFlow(t *testing.T) {
	white, black, id := uuid.New(), uuid.New(), uuid.New()
	store := &fakeStore{players: Players{white, black}}
	game := &fakeGame{}
	h := New(store, func(uuid.UUID) (chesslogic.Game, error) { return game, nil })
	ctx := context.Background()
	if _, err := h.Join(ctx, id, uuid.New()); err != ErrForbidden {
		t.Fatalf("join outsider: %v", err)
	}
	w, err := h.Join(ctx, id, white)
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.Join(ctx, id, black)
	if err != nil {
		t.Fatal(err)
	}
	receive(t, w)
	receive(t, b)
	cases := []struct {
		name   string
		client *Client
		raw    string
		code   string
		close  bool
	}{
		{"wrong turn", b, `{"type":"move","payload":{"from":"e2","to":"e4"}}`, "not_your_turn", false},
		{"illegal", w, `{"type":"move","payload":{"from":"e2","to":"e5"}}`, "invalid_move", false},
		{"malformed", w, `{`, "", true},
		{"unknown", w, `{"type":"oops"}`, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeNow, err := h.Handle(ctx, id, tc.client, []byte(tc.raw))
			if closeNow != tc.close {
				t.Fatalf("close=%v err=%v", closeNow, err)
			}
			if tc.code != "" {
				m := receive(t, tc.client)
				e := m.Payload.(Error)
				if e.Code != tc.code {
					t.Fatalf("code=%s", e.Code)
				}
			}
		})
	}
	if len(store.moves) != 0 {
		t.Fatal("invalid moves changed state")
	}
	if closeNow, err := h.Handle(ctx, id, w, []byte(`{"type":"move","payload":{"from":"e2","to":"e4"}}`)); closeNow || err != nil {
		t.Fatalf("move: %v %v", closeNow, err)
	}
	for _, c := range []*Client{w, b} {
		m := receive(t, c)
		if m.Type != "game_state" || m.Payload.(State).LastMove != "e2e4" {
			t.Fatalf("state: %+v", m)
		}
	}
	if len(store.moves) != 1 {
		t.Fatalf("saved moves: %v", store.moves)
	}
	_, err = h.Handle(ctx, id, b, []byte(`{"type":"draw_offer"}`))
	if err != nil {
		t.Fatal(err)
	}
	if receive(t, w).Type != "draw_offer" {
		t.Fatal("offer not forwarded")
	}
	_, err = h.Handle(ctx, id, w, []byte(`{"type":"draw_accept"}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []*Client{w, b} {
		if receive(t, c).Payload.(State).Status != "1/2-1/2" {
			t.Fatal("draw status")
		}
	}
	if store.result != "1/2-1/2" {
		t.Fatal("draw not saved")
	}
}
func TestMessageShape(t *testing.T) {
	raw, _ := json.Marshal(Message{Type: "error", Payload: Error{"code", "message"}})
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil || m["type"] == nil || m["payload"] == nil {
		t.Fatalf("wire shape: %s", raw)
	}
}
