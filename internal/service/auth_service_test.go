package service

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pieremarie/chessmobile/internal/model"
	"github.com/pieremarie/chessmobile/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"testing"
	"time"
)

type fakeRepo struct {
	user              model.User
	createErr, getErr error
	created           model.User
}

func (f *fakeRepo) Create(_ context.Context, u model.User) error { f.created = u; return f.createErr }
func (f *fakeRepo) GetByEmail(_ context.Context, _ string) (model.User, error) {
	return f.user, f.getErr
}
func TestRegister(t *testing.T) {
	cases := []struct {
		name, nick, email, password string
		createErr, wantErr          error
	}{
		{"success", "player", "a@example.com", "password123", nil, nil},
		{"duplicate", "player", "a@example.com", "password123", repository.ErrEmailExists, ErrEmailExists},
		{"bad email", "player", "bad", "password123", nil, ErrInvalidInput},
		{"short password", "player", "a@example.com", "short", nil, ErrInvalidInput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeRepo{createErr: tc.createErr}
			s, _ := NewAuthService(f, "test-secret")
			id, err := s.Register(context.Background(), tc.nick, tc.email, tc.password)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error: got %v want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				if id == uuid.Nil || f.created.ID != id {
					t.Fatal("invalid user ID")
				}
				if f.created.PasswordHash == tc.password || bcrypt.CompareHashAndPassword([]byte(f.created.PasswordHash), []byte(tc.password)) != nil {
					t.Fatal("password not hashed")
				}
			}
		})
	}
}
func TestLogin(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	id := uuid.New()
	cases := []struct {
		name, password  string
		getErr, wantErr error
	}{
		{"success", "password123", nil, nil},
		{"wrong password", "wrongpass", nil, ErrInvalidCredentials},
		{"unknown email", "password123", pgx.ErrNoRows, ErrInvalidCredentials},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeRepo{user: model.User{ID: id, PasswordHash: string(hash)}, getErr: tc.getErr}
			s, _ := NewAuthService(f, "test-secret")
			token, err := s.Login(context.Background(), "a@example.com", tc.password)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error: got %v want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				parsed, e := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
					if t.Method.Alg() != "HS256" {
						return nil, errors.New("wrong algorithm")
					}
					return []byte("test-secret"), nil
				})
				if e != nil || !parsed.Valid {
					t.Fatalf("invalid token: %v", e)
				}
				claims := parsed.Claims.(*jwt.RegisteredClaims)
				if claims.Subject != id.String() {
					t.Fatal("wrong subject")
				}
				exp, e := claims.GetExpirationTime()
				if e != nil || time.Until(exp.Time) < 14*time.Minute || time.Until(exp.Time) > 15*time.Minute {
					t.Fatal("wrong expiry")
				}
			}
		})
	}
}
