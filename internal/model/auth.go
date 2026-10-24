package model

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID           uuid.UUID
	Nickname     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
