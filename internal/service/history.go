package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/model"
)

type HistoryReader interface {
	Profile(context.Context, uuid.UUID) (model.Profile, error)
	Games(context.Context, uuid.UUID, int, int) ([]model.GameSummary, error)
	Game(context.Context, uuid.UUID, uuid.UUID) (model.GameDetail, error)
}
type HistoryService struct{ repo HistoryReader }

func NewHistoryService(r HistoryReader) *HistoryService { return &HistoryService{repo: r} }
func (s *HistoryService) Profile(ctx context.Context, user uuid.UUID) (model.Profile, error) {
	return s.repo.Profile(ctx, user)
}
func (s *HistoryService) Games(ctx context.Context, user uuid.UUID, limit, offset int) ([]model.GameSummary, error) {
	return s.repo.Games(ctx, user, limit, offset)
}
func (s *HistoryService) Game(ctx context.Context, user, id uuid.UUID) (model.GameDetail, error) {
	return s.repo.Game(ctx, user, id)
}
