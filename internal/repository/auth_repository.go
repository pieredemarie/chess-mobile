package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pieremarie/chessmobile/internal/model"
)

var ErrUserNotFound = errors.New("user not found")
var ErrEmailExists = errors.New("email already exists")

type DB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type UserRepository struct{ db DB }

func NewUserRepository(db DB) *UserRepository { return &UserRepository{db} }
func (r *UserRepository) Create(ctx context.Context, u model.User) error {
	_, err := r.db.Exec(ctx, "INSERT INTO users (id,nickname,email,password_hash) VALUES ($1,$2,$3,$4)", u.ID, u.Nickname, u.Email, u.PasswordHash)
	if err != nil {
		var p *pgconn.PgError
		if errors.As(err, &p) && p.Code == "23505" && p.ConstraintName == "users_email_key" {
			return ErrEmailExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx, "SELECT id,nickname,email,password_hash,created_at FROM users WHERE email=$1", email).Scan(&u.ID, &u.Nickname, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}
