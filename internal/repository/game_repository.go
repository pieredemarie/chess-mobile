package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pieremarie/chessmobile/internal/hub"
)

// GameDB is the pgx functionality needed by GameRepository.
type GameDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}
type GameRepository struct{ db GameDB }

func NewGameRepository(db GameDB) *GameRepository { return &GameRepository{db: db} }

// CreateGame persists a new game in its standard initial position.
func (r *GameRepository) CreateGame(ctx context.Context, id uuid.UUID, players hub.Players, fen string) error {
	if players.White == uuid.Nil || players.Black == uuid.Nil || players.White == players.Black || fen == "" {
		return errors.New("invalid game")
	}
	_, err := r.db.Exec(ctx, `INSERT INTO games (id,white_id,black_id,fen) VALUES ($1,$2,$3,$4)`, id, players.White, players.Black, fen)
	if err != nil {
		return fmt.Errorf("create game: %w", err)
	}
	return nil
}
func (r *GameRepository) Players(ctx context.Context, id uuid.UUID) (hub.Players, error) {
	var p hub.Players
	err := r.db.QueryRow(ctx, `SELECT white_id,black_id FROM games WHERE id=$1`, id).Scan(&p.White, &p.Black)
	if errors.Is(err, pgx.ErrNoRows) {
		return hub.Players{}, hub.ErrNotFound
	}
	if err != nil {
		return hub.Players{}, fmt.Errorf("load game players: %w", err)
	}
	return p, nil
}
func (r *GameRepository) FEN(ctx context.Context, id uuid.UUID) (string, error) {
	var fen string
	var result *string
	err := r.db.QueryRow(ctx, `SELECT fen,result FROM games WHERE id=$1`, id).Scan(&fen, &result)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", hub.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load game FEN: %w", err)
	}
	if result != nil {
		return "", hub.ErrFinished
	}
	return fen, nil
}
func (r *GameRepository) SaveMove(ctx context.Context, id uuid.UUID, uci, fen string) error {
	if uci == "" || fen == "" {
		return errors.New("invalid move")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin move transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var next int
	err = tx.QueryRow(ctx, `UPDATE games SET ply=ply+1,fen=$2,updated_at=now() WHERE id=$1 AND result IS NULL RETURNING ply`, id, fen).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return hub.ErrFinished
	}
	if err != nil {
		return fmt.Errorf("lock game for move: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO game_moves (game_id,ply,uci,fen) VALUES ($1,$2,$3,$4)`, id, next, uci, fen); err != nil {
		return fmt.Errorf("insert move: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit move: %w", err)
	}
	return nil
}
func (r *GameRepository) SaveResult(ctx context.Context, id uuid.UUID, result string) error {
	if result != "1-0" && result != "0-1" && result != "1/2-1/2" {
		return errors.New("invalid result")
	}
	tag, err := r.db.Exec(ctx, `UPDATE games SET result=$2,updated_at=now() WHERE id=$1 AND result IS NULL`, id, result)
	if err != nil {
		return fmt.Errorf("save result: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return hub.ErrFinished
	}
	return nil
}
