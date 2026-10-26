package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/hub"
)

type RatedStore struct {
	Games interface {
		Players(context.Context, uuid.UUID) (hub.Players, error)
		SaveMove(context.Context, uuid.UUID, string, string) error
	}
	Ratings interface {
		Finish(context.Context, uuid.UUID, string, func(int, int, string) (int, int)) error
	}
}

func (s RatedStore) Players(ctx context.Context, id uuid.UUID) (hub.Players, error) {
	return s.Games.Players(ctx, id)
}
func (s RatedStore) SaveMove(ctx context.Context, id uuid.UUID, uci, fen string) error {
	return s.Games.SaveMove(ctx, id, uci, fen)
}
func (s RatedStore) SaveResult(ctx context.Context, id uuid.UUID, result string) error {
	return s.Ratings.Finish(ctx, id, result, EloRatings)
}
