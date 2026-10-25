package chesslogic

import (
	"reflect"
	"strings"
	"testing"
)

func TestSpecialMoves(t *testing.T) {
	cases := []struct {
		name, fen, move, board string
	}{
		{"castle", "4k2r/8/8/8/8/8/8/4K2R w Kk - 0 1", "e1g1", "5RK1"},
		{"en passant", "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1", "e5d6", "3P4"},
		{"promotion", "4k3/P7/8/8/8/8/8/4K3 w - - 0 1", "a7a8q", "Q3k3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			game, err := FromFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			if err := game.Move(tc.move); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(game.FEN(), tc.board) {
				t.Fatalf("FEN = %q, want board segment %q", game.FEN(), tc.board)
			}
			if got := game.History(); !reflect.DeepEqual(got, []string{tc.move}) {
				t.Fatalf("history = %v", got)
			}
		})
	}
}

func TestOutcomes(t *testing.T) {
	cases := []struct {
		name, fen, move, outcome, method string
	}{
		{"checkmate", "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1", "f7g7", "1-0", "Checkmate"},
		{"stalemate", "k1K5/8/8/8/8/8/8/1Q6 w - - 0 1", "b1b6", "1/2-1/2", "Stalemate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			game, err := FromFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			if err := game.Move(tc.move); err != nil {
				t.Fatal(err)
			}
			status := game.Status()
			if status.Outcome != tc.outcome || status.Method != tc.method {
				t.Fatalf("status = %+v", status)
			}
		})
	}
}

func TestThreefoldRepetition(t *testing.T) {
	game := New()
	if err := game.ClaimThreefoldRepetition(); err == nil {
		t.Fatal("early claim accepted")
	}
	for _, move := range []string{"g1f3", "g8f6", "f3g1", "f6g8", "g1f3", "g8f6", "f3g1", "f6g8"} {
		if err := game.Move(move); err != nil {
			t.Fatal(err)
		}
	}
	if !game.Status().CanClaimThreefoldDraw {
		t.Fatal("threefold draw not available")
	}
	if err := game.ClaimThreefoldRepetition(); err != nil {
		t.Fatal(err)
	}
	status := game.Status()
	if status.Outcome != "1/2-1/2" || status.Method != "ThreefoldRepetition" {
		t.Fatalf("status = %+v", status)
	}
}

func TestInvalidInput(t *testing.T) {
	if _, err := FromFEN("invalid"); err == nil {
		t.Fatal("invalid FEN accepted")
	}
	game := New()
	before := game.FEN()
	if err := game.Move("e2e5"); err == nil {
		t.Fatal("illegal move accepted")
	}
	if game.FEN() != before || len(game.History()) != 0 {
		t.Fatal("illegal move changed game")
	}
}
