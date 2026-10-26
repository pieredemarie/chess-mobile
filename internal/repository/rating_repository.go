package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pieremarie/chessmobile/internal/hub"
)

type RatingRepository struct{ db GameDB }

func NewRatingRepository(db GameDB) *RatingRepository { return &RatingRepository{db: db} }

// Finish locks the game and both users, then saves the result and ratings atomically.
func (r *RatingRepository) Finish(ctx context.Context, id uuid.UUID, result string, calculate func(int, int, string) (int, int)) error {
	if result != "1-0" && result != "0-1" && result != "1/2-1/2" {
		return errors.New("invalid result")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin result transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var white, black uuid.UUID
	var prior *string
	err = tx.QueryRow(ctx, `SELECT white_id,black_id,result FROM games WHERE id=$1 FOR UPDATE`, id).Scan(&white, &black, &prior)
	if errors.Is(err, pgx.ErrNoRows) {
		return hub.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock game: %w", err)
	}
	if prior != nil {
		return hub.ErrFinished
	}
	rows, err := tx.Query(ctx, `SELECT id,rating FROM users WHERE id=$1 OR id=$2 ORDER BY id FOR UPDATE`, white, black)
	if err != nil {
		return fmt.Errorf("lock ratings: %w", err)
	}
	ratings := map[uuid.UUID]int{}
	for rows.Next() {
		var user uuid.UUID
		var rating int
		if err = rows.Scan(&user, &rating); err != nil {
			rows.Close()
			return fmt.Errorf("read rating: %w", err)
		}
		ratings[user] = rating
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return fmt.Errorf("read ratings: %w", err)
	}
	if len(ratings) != 2 {
		return errors.New("game player missing")
	}
	oldW, oldB := ratings[white], ratings[black]
	newW, newB := calculate(oldW, oldB, result)
	if _, err = tx.Exec(ctx, `UPDATE users SET rating=CASE id WHEN $1 THEN $3 ELSE $4 END WHERE id IN ($1,$2)`, white, black, newW, newB); err != nil {
		return fmt.Errorf("save ratings: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE games SET result=$2,white_rating_before=$3,black_rating_before=$4,white_rating_after=$5,black_rating_after=$6,updated_at=now() WHERE id=$1`, id, result, oldW, oldB, newW, newB); err != nil {
		return fmt.Errorf("save result: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit result: %w", err)
	}
	return nil
}
