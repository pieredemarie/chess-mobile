package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pieremarie/chessmobile/internal/hub"
)

func TestGameRepositoryPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL integration test")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	schema := "game_test_" + uuid.NewString()[:8]
	if _, err = tx.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `SET LOCAL search_path TO `+schema); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE users (id uuid PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000002_create_games.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range strings.Split(string(migration), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err = tx.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	white, black, id := uuid.New(), uuid.New(), uuid.New()
	if _, err = tx.Exec(ctx, `INSERT INTO users(id) VALUES ($1),($2)`, white, black); err != nil {
		t.Fatal(err)
	}
	r := NewGameRepository(tx)
	if err = r.CreateGame(ctx, id, hub.Players{White: white, Black: black}, "initial fen"); err != nil {
		t.Fatal(err)
	}
	players, err := r.Players(ctx, id)
	if err != nil || players.White != white || players.Black != black {
		t.Fatalf("players=%+v err=%v", players, err)
	}
	fen, err := r.FEN(ctx, id)
	if err != nil || fen != "initial fen" {
		t.Fatalf("fen=%q err=%v", fen, err)
	}
	if err = r.SaveMove(ctx, id, "e2e4", "after e2e4"); err != nil {
		t.Fatal(err)
	}
	if err = r.SaveMove(ctx, id, "e7e5", "after e7e5"); err != nil {
		t.Fatal(err)
	}
	var count, maxPly int
	if err = tx.QueryRow(ctx, `SELECT count(*),max(ply) FROM game_moves WHERE game_id=$1`, id).Scan(&count, &maxPly); err != nil {
		t.Fatal(err)
	}
	if count != 2 || maxPly != 2 {
		t.Fatalf("moves=%d max ply=%d", count, maxPly)
	}
	fen, err = r.FEN(ctx, id)
	if err != nil || fen != "after e7e5" {
		t.Fatalf("fen=%q err=%v", fen, err)
	}
	if err = r.SaveResult(ctx, id, "1-0"); err != nil {
		t.Fatal(err)
	}
	if _, err = r.FEN(ctx, id); !errors.Is(err, hub.ErrFinished) {
		t.Fatalf("finished game: %v", err)
	}
	if err = r.SaveMove(ctx, id, "g1f3", "later"); !errors.Is(err, hub.ErrFinished) {
		t.Fatalf("move after finish: %v", err)
	}
}
