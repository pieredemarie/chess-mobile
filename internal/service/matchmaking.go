package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/chesslogic"
	"github.com/pieremarie/chessmobile/internal/hub"
)

var ErrAlreadySearching = errors.New("already searching")
var ErrNotSearching = errors.New("not searching")

type MatchStore interface {
	CreateGame(context.Context, uuid.UUID, hub.Players, string) error
}
type Match struct {
	GameID uuid.UUID `json:"game_id"`
	White  uuid.UUID `json:"white_id"`
	Black  uuid.UUID `json:"black_id"`
}

type Matchmaker struct {
	mu      sync.Mutex
	queue   []uuid.UUID
	waiting map[uuid.UUID]bool
	matched map[uuid.UUID]Match
	store   MatchStore
}

func NewMatchmaker(store MatchStore) *Matchmaker {
	return &Matchmaker{store: store, waiting: make(map[uuid.UUID]bool), matched: make(map[uuid.UUID]Match)}
}

func (m *Matchmaker) Search(ctx context.Context, user uuid.UUID) (*Match, error) {
	if user == uuid.Nil {
		return nil, errors.New("invalid user")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if match, ok := m.matched[user]; ok {
		return &match, nil
	}
	if m.waiting[user] {
		return nil, ErrAlreadySearching
	}
	if len(m.queue) == 0 {
		m.queue = append(m.queue, user)
		m.waiting[user] = true
		return nil, nil
	}
	first := m.queue[0]
	match := Match{GameID: uuid.New(), White: first, Black: user}
	if err := m.store.CreateGame(ctx, match.GameID, hub.Players{White: first, Black: user}, chesslogic.New().FEN()); err != nil {
		return nil, fmt.Errorf("create matched game: %w", err)
	}
	m.queue = m.queue[1:]
	delete(m.waiting, first)
	m.matched[first] = match
	m.matched[user] = match
	return &match, nil
}

func (m *Matchmaker) Status(user uuid.UUID) (*Match, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if match, ok := m.matched[user]; ok {
		delete(m.matched, user)
		return &match, false
	}
	return nil, m.waiting[user]
}

func (m *Matchmaker) Cancel(user uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.waiting[user] {
		return ErrNotSearching
	}
	delete(m.waiting, user)
	for i, id := range m.queue {
		if id == user {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			break
		}
	}
	return nil
}
