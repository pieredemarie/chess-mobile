package timedhub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pieremarie/chessmobile/internal/hub"
	"github.com/pieremarie/chessmobile/internal/service"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/chesslogic"
)

var ErrNotFound = hub.ErrNotFound
var ErrForbidden = hub.ErrForbidden
var ErrFinished = hub.ErrFinished

type Players = hub.Players

// GameStore is implemented by persistent storage in a separate task.
type GameStore interface {
	Players(context.Context, uuid.UUID) (Players, error)
	SaveMove(context.Context, uuid.UUID, string, string) error
	SaveResult(context.Context, uuid.UUID, string) error
}
type GameFactory func(uuid.UUID) (chesslogic.Game, error)
type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}
type incoming struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
type Move struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
}
type State struct {
	FEN      string `json:"fen"`
	Turn     string `json:"turn"`
	Status   string `json:"status"`
	LastMove string `json:"last_move,omitempty"`
}
type TimedState struct {
	State
	WhiteMS int64 `json:"white_ms"`
	BlackMS int64 `json:"black_ms"`
}
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Client struct {
	User uuid.UUID
	Send chan Message
}
type command struct {
	client *Client
	kind   string
	move   Move
	reply  chan error
}
type party struct {
	id       uuid.UUID
	players  Players
	game     chesslogic.Game
	store    GameStore
	commands chan command
	done     chan struct{}
	onDone   func()
	clients  map[*Client]bool
	offered  uuid.UUID
	result   string
	lastMove string
	clock    service.Clock
}
type Hub struct {
	mu        sync.Mutex
	parties   map[uuid.UUID]*party
	store     GameStore
	factory   GameFactory
	base      time.Duration
	increment time.Duration
}

func New(store GameStore, factory GameFactory) *Hub {
	return NewWithTimeControl(store, factory, 10*time.Minute, 0)
}
func NewWithTimeControl(store GameStore, factory GameFactory, base, increment time.Duration) *Hub {
	return &Hub{parties: make(map[uuid.UUID]*party), store: store, factory: factory, base: base, increment: increment}
}
func (h *Hub) Join(ctx context.Context, id, user uuid.UUID) (*Client, error) {
	h.mu.Lock()
	p := h.parties[id]
	h.mu.Unlock()
	if p == nil {
		players, err := h.store.Players(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("load players: %w", err)
		}
		if user != players.White && user != players.Black {
			return nil, ErrForbidden
		}
		game, err := h.factory(id)
		if err != nil {
			return nil, fmt.Errorf("create game: %w", err)
		}
		clock, err := service.NewClock(h.base, h.increment, time.Now())
		if err != nil {
			return nil, err
		}
		candidate := &party{clock: clock, id: id, players: players, game: game, store: h.store, commands: make(chan command), done: make(chan struct{}), clients: make(map[*Client]bool)}
		candidate.onDone = func() {
			h.mu.Lock()
			if h.parties[id] == candidate {
				delete(h.parties, id)
			}
			h.mu.Unlock()
		}
		h.mu.Lock()
		p = h.parties[id]
		if p == nil {
			p = candidate
			h.parties[id] = p
			go p.run()
		}
		h.mu.Unlock()
	}
	if user != p.players.White && user != p.players.Black {
		return nil, ErrForbidden
	}
	c := &Client{User: user, Send: make(chan Message, 32)}
	if err := p.submit(ctx, command{client: c, kind: "join"}); err != nil {
		return nil, err
	}
	return c, nil
}
func (h *Hub) Leave(id uuid.UUID, c *Client) {
	h.mu.Lock()
	p := h.parties[id]
	h.mu.Unlock()
	if p != nil {
		_ = p.submit(context.Background(), command{client: c, kind: "leave"})
	}
}
func (h *Hub) Handle(ctx context.Context, id uuid.UUID, c *Client, data []byte) (bool, error) {
	var msg incoming
	if err := json.Unmarshal(data, &msg); err != nil {
		return true, err
	}
	switch msg.Type {
	case "move":
		var m Move
		if len(msg.Payload) == 0 || json.Unmarshal(msg.Payload, &m) != nil {
			return true, errors.New("invalid move payload")
		}
		return false, h.dispatch(ctx, id, c, "move", m)
	case "resign", "draw_offer", "draw_accept", "draw_decline":
		return false, h.dispatch(ctx, id, c, msg.Type, Move{})
	default:
		return true, errors.New("unknown message type")
	}
}
func (h *Hub) dispatch(ctx context.Context, id uuid.UUID, c *Client, kind string, m Move) error {
	h.mu.Lock()
	p := h.parties[id]
	h.mu.Unlock()
	if p == nil {
		return ErrFinished
	}
	return p.submit(ctx, command{client: c, kind: kind, move: m})
}
func (p *party) submit(ctx context.Context, c command) error {
	c.reply = make(chan error, 1)
	select {
	case p.commands <- c:
	case <-p.done:
		return ErrFinished
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-c.reply:
		return err
	case <-p.done:
		return ErrFinished
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (p *party) run() {
	defer close(p.done)
	defer p.onDone()
	for {
		wait := time.Until(p.clock.Deadline())
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		var c command
		select {
		case c = <-p.commands:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
			result := "0-1"
			if p.clock.Turn == "b" {
				result = "1-0"
			}
			if err := p.store.SaveResult(context.Background(), p.id, result); err != nil {
				log.Printf("save timeout result: %v", err)
				if errors.Is(err, ErrFinished) {
					return
				}
				time.Sleep(time.Second)
				continue
			}
			p.result = result
			p.broadcast(p.state())
			for client := range p.clients {
				close(client.Send)
			}
			return
		}
		err := p.apply(c)
		c.reply <- err
		if p.result != "" {
			for client := range p.clients {
				close(client.Send)
			}
			return
		}
	}
}
func (p *party) send(c *Client, m Message) {
	select {
	case c.Send <- m:
	default:
		delete(p.clients, c)
		close(c.Send)
	}
}
func (p *party) broadcast(m Message) {
	for c := range p.clients {
		p.send(c, m)
	}
}
func (p *party) state() Message {
	s := p.game.Status()
	status := s.Outcome
	if p.result != "" {
		status = p.result
	}
	w, b := p.clock.Remaining(time.Now())
	return Message{"game_state", TimedState{State: State{p.game.FEN(), s.Turn, status, p.lastMove}, WhiteMS: w.Milliseconds(), BlackMS: b.Milliseconds()}}
}
func (p *party) apply(c command) error {
	if c.kind == "join" {
		p.clients[c.client] = true
		p.send(c.client, p.state())
		return nil
	}
	if c.kind == "leave" {
		if p.clients[c.client] {
			delete(p.clients, c.client)
			close(c.client.Send)
		}
		return nil
	}
	if !p.clients[c.client] {
		return ErrForbidden
	}
	fail := func(code, msg string) error { p.send(c.client, Message{"error", Error{code, msg}}); return nil }
	switch c.kind {
	case "move":
		m := c.move
		if len(m.From) != 2 || len(m.To) != 2 || len(m.Promotion) > 1 {
			return fail("invalid_move", "invalid move")
		}
		turn := p.game.Status().Turn
		if (turn == "w" && c.client.User != p.players.White) || (turn == "b" && c.client.User != p.players.Black) {
			return fail("not_your_turn", "not your turn")
		}
		moveAt := time.Now()
		if w, b := p.clock.Remaining(moveAt); (turn == "w" && w == 0) || (turn == "b" && b == 0) {
			return fail("time_expired", "time expired")
		}
		uci := strings.ToLower(m.From + m.To + m.Promotion)
		if err := p.game.Move(uci); err != nil {
			return fail("invalid_move", "invalid move")
		}
		if err := p.store.SaveMove(context.Background(), p.id, uci, p.game.FEN()); err != nil {
			return fmt.Errorf("save move: %w", err)
		}
		if err := p.clock.Move(moveAt); err != nil {
			return fmt.Errorf("update clock: %w", err)
		}
		p.lastMove = uci
		p.offered = uuid.Nil
		if s := p.game.Status(); s.Outcome != "*" {
			p.result = s.Outcome
			if err := p.store.SaveResult(context.Background(), p.id, p.result); err != nil {
				return fmt.Errorf("save result: %w", err)
			}
		}
		p.broadcast(p.state())
	case "resign":
		result := "1-0"
		if c.client.User == p.players.White {
			result = "0-1"
		}
		if err := p.store.SaveResult(context.Background(), p.id, result); err != nil {
			return fmt.Errorf("save result: %w", err)
		}
		p.result = result
		p.broadcast(p.state())
	case "draw_offer":
		p.offered = c.client.User
		for client := range p.clients {
			if client.User != c.client.User {
				p.send(client, Message{"draw_offer", struct{}{}})
			}
		}
	case "draw_accept":
		if p.offered == uuid.Nil || p.offered == c.client.User {
			return fail("no_draw_offer", "no opponent draw offer")
		}
		if err := p.store.SaveResult(context.Background(), p.id, "1/2-1/2"); err != nil {
			return fmt.Errorf("save result: %w", err)
		}
		p.result = "1/2-1/2"
		p.broadcast(p.state())
	case "draw_decline":
		if p.offered == uuid.Nil || p.offered == c.client.User {
			return fail("no_draw_offer", "no opponent draw offer")
		}
		p.offered = uuid.Nil
	}
	return nil
}
