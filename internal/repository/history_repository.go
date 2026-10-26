package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pieremarie/chessmobile/internal/model"
)

var ErrGameNotFound = errors.New("game not found")

type HistoryDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}
type HistoryRepository struct{ db HistoryDB }

func NewHistoryRepository(db HistoryDB) *HistoryRepository { return &HistoryRepository{db: db} }
func (r *HistoryRepository) Profile(ctx context.Context, user uuid.UUID) (model.Profile, error) {
	var p model.Profile
	err := r.db.QueryRow(ctx, `SELECT nickname,email,rating FROM users WHERE id=$1`, user).Scan(&p.Nickname, &p.Email, &p.Rating)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrUserNotFound
	}
	if err != nil {
		return p, fmt.Errorf("load profile: %w", err)
	}
	return p, nil
}

const summaryColumns = `id,white_id,black_id,result,ply,created_at,updated_at,white_rating_before,black_rating_before,white_rating_after,black_rating_after`

func scanSummary(row pgx.Row) (model.GameSummary, error) {
	var g model.GameSummary
	err := row.Scan(&g.ID, &g.WhiteID, &g.BlackID, &g.Result, &g.Ply, &g.CreatedAt, &g.UpdatedAt, &g.WhiteRatingBefore, &g.BlackRatingBefore, &g.WhiteRatingAfter, &g.BlackRatingAfter)
	return g, err
}
func (r *HistoryRepository) Games(ctx context.Context, user uuid.UUID, limit, offset int) ([]model.GameSummary, error) {
	rows, err := r.db.Query(ctx, `SELECT `+summaryColumns+` FROM games WHERE white_id=$1 OR black_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, user, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	defer rows.Close()
	games := make([]model.GameSummary, 0)
	for rows.Next() {
		g, err := scanSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("scan game: %w", err)
		}
		games = append(games, g)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	return games, nil
}
func (r *HistoryRepository) Game(ctx context.Context, user, id uuid.UUID) (model.GameDetail, error) {
	var d model.GameDetail
	row := r.db.QueryRow(ctx, `SELECT `+summaryColumns+`,fen FROM games WHERE id=$1 AND (white_id=$2 OR black_id=$2)`, id, user)
	err := row.Scan(&d.ID, &d.WhiteID, &d.BlackID, &d.Result, &d.Ply, &d.CreatedAt, &d.UpdatedAt, &d.WhiteRatingBefore, &d.BlackRatingBefore, &d.WhiteRatingAfter, &d.BlackRatingAfter, &d.FEN)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrGameNotFound
	}
	if err != nil {
		return d, fmt.Errorf("load game: %w", err)
	}
	rows, err := r.db.Query(ctx, `SELECT ply,uci,fen,created_at FROM game_moves WHERE game_id=$1 ORDER BY ply`, id)
	if err != nil {
		return d, fmt.Errorf("load moves: %w", err)
	}
	defer rows.Close()
	d.Moves = make([]model.RecordedMove, 0)
	for rows.Next() {
		var m model.RecordedMove
		if err = rows.Scan(&m.Ply, &m.UCI, &m.FEN, &m.CreatedAt); err != nil {
			return d, fmt.Errorf("scan move: %w", err)
		}
		d.Moves = append(d.Moves, m)
	}
	if err = rows.Err(); err != nil {
		return d, fmt.Errorf("load moves: %w", err)
	}
	return d, nil
}
