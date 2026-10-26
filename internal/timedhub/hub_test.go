package timedhub

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/chesslogic"
)

type memoryStore struct {
	players Players
	result  chan string
}

func (s *memoryStore) Players(context.Context, uuid.UUID) (Players, error)       { return s.players, nil }
func (s *memoryStore) SaveMove(context.Context, uuid.UUID, string, string) error { return nil }
func (s *memoryStore) SaveResult(_ context.Context, _ uuid.UUID, r string) error {
	s.result <- r
	return nil
}

func TestTimeoutWithoutMessages(t *testing.T) {
	white, black, id := uuid.New(), uuid.New(), uuid.New()
	store := &memoryStore{players: Players{White: white, Black: black}, result: make(chan string, 1)}
	h := NewWithTimeControl(store, func(uuid.UUID) (chesslogic.Game, error) { return chesslogic.New(), nil }, 20*time.Millisecond, 0)
	client, err := h.Join(context.Background(), id, white)
	if err != nil {
		t.Fatal(err)
	}
	<-client.Send
	select {
	case result := <-store.result:
		if result != "0-1" {
			t.Fatalf("result %s", result)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout not persisted")
	}
}
