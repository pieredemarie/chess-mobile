package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/hub"
)

type matchStore struct {
	calls   int
	err     error
	players hub.Players
}

func (s *matchStore) CreateGame(_ context.Context, _ uuid.UUID, p hub.Players, _ string) error {
	s.calls++
	s.players = p
	return s.err
}

func TestMatchmaking(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	cases := []struct {
		name   string
		cancel bool
		fail   bool
	}{
		{"pair", false, false}, {"cancel", true, false}, {"create failure", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &matchStore{}
			m := NewMatchmaker(store)
			if match, err := m.Search(context.Background(), a); err != nil || match != nil {
				t.Fatalf("first: %v %v", match, err)
			}
			if _, err := m.Search(context.Background(), a); !errors.Is(err, ErrAlreadySearching) {
				t.Fatalf("duplicate: %v", err)
			}
			if tc.cancel {
				if err := m.Cancel(a); err != nil {
					t.Fatal(err)
				}
				if _, waiting := m.Status(a); waiting {
					t.Fatal("still waiting")
				}
				return
			}
			if tc.fail {
				store.err = errors.New("db down")
			}
			match, err := m.Search(context.Background(), b)
			if tc.fail {
				if err == nil || match != nil {
					t.Fatalf("create failure: %v %v", match, err)
				}
				if _, waiting := m.Status(a); !waiting {
					t.Fatal("first lost")
				}
				return
			}
			if err != nil || match == nil || match.White != a || match.Black != b || store.calls != 1 {
				t.Fatalf("match: %v %v", match, err)
			}
			found, waiting := m.Status(a)
			if waiting || found == nil || found.GameID != match.GameID {
				t.Fatal("first not notified")
			}
		})
	}
}
