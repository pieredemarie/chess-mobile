package model

import (
	"github.com/google/uuid"
	"time"
)

type Profile struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Rating   int    `json:"rating"`
}
type GameSummary struct {
	ID                uuid.UUID `json:"id"`
	WhiteID           uuid.UUID `json:"white_id"`
	BlackID           uuid.UUID `json:"black_id"`
	Result            *string   `json:"result"`
	Ply               int       `json:"ply"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	WhiteRatingBefore *int      `json:"white_rating_before"`
	BlackRatingBefore *int      `json:"black_rating_before"`
	WhiteRatingAfter  *int      `json:"white_rating_after"`
	BlackRatingAfter  *int      `json:"black_rating_after"`
}
type RecordedMove struct {
	Ply       int       `json:"ply"`
	UCI       string    `json:"uci"`
	FEN       string    `json:"fen"`
	CreatedAt time.Time `json:"created_at"`
}
type GameDetail struct {
	GameSummary
	FEN   string         `json:"fen"`
	Moves []RecordedMove `json:"moves"`
}
