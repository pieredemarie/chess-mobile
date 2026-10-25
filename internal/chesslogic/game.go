package chesslogic

import (
	"fmt"

	"github.com/notnil/chess"
)

// Game exposes chess rules using UCI moves (for example, e2e4 or e7e8q).
type Game interface {
	Move(uci string) error
	FEN() string
	History() []string
	Status() Status
	ClaimThreefoldRepetition() error
}

// Status describes the side to move, game result, and repetition claim.
type Status struct {
	Turn                  string
	Outcome               string
	Method                string
	CanClaimThreefoldDraw bool
}

type game struct{ engine *chess.Game }

// New creates a game in the standard initial position.
func New() Game { return &game{engine: chess.NewGame()} }

// FromFEN creates a game from a position. FEN does not contain move history.
func FromFEN(value string) (Game, error) {
	option, err := chess.FEN(value)
	if err != nil {
		return nil, fmt.Errorf("parse FEN: %w", err)
	}
	return &game{engine: chess.NewGame(option)}, nil
}

func (g *game) Move(uci string) error {
	move, err := (chess.UCINotation{}).Decode(g.engine.Position(), uci)
	if err != nil {
		return fmt.Errorf("decode UCI move %q: %w", uci, err)
	}
	if err := g.engine.Move(move); err != nil {
		return fmt.Errorf("apply UCI move %q: %w", uci, err)
	}
	return nil
}

func (g *game) FEN() string { return g.engine.FEN() }

func (g *game) History() []string {
	moves, positions := g.engine.Moves(), g.engine.Positions()
	history := make([]string, len(moves))
	for i, move := range moves {
		history[i] = (chess.UCINotation{}).Encode(positions[i], move)
	}
	return history
}

func (g *game) Status() Status {
	status := Status{
		Turn:    g.engine.Position().Turn().String(),
		Outcome: g.engine.Outcome().String(),
		Method:  g.engine.Method().String(),
	}
	for _, method := range g.engine.EligibleDraws() {
		if method == chess.ThreefoldRepetition {
			status.CanClaimThreefoldDraw = true
		}
	}
	return status
}

func (g *game) ClaimThreefoldRepetition() error {
	if err := g.engine.Draw(chess.ThreefoldRepetition); err != nil {
		return fmt.Errorf("claim threefold repetition: %w", err)
	}
	return nil
}
