package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pieremarie/chessmobile/internal/model"
	"github.com/pieremarie/chessmobile/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailExists = repository.ErrEmailExists
var ErrUserNotFound = repository.ErrUserNotFound

type UserRepository interface {
	Create(context.Context, model.User) error
	GetByEmail(context.Context, string) (model.User, error)
}
type AuthService struct {
	repo   UserRepository
	secret []byte
}

func NewAuthService(r UserRepository, s string) (*AuthService, error) {
	if s == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	return &AuthService{r, []byte(s)}, nil
}
func validEmail(e string) bool {
	if e == "" || strings.TrimSpace(e) != e {
		return false
	}
	a, err := mail.ParseAddress(e)
	return err == nil && a.Address == e && strings.Contains(e, ".")
}
func (a *AuthService) Register(ctx context.Context, nickname, email, password string) (uuid.UUID, error) {
	email = strings.ToLower(email)
	if strings.TrimSpace(nickname) == "" || !validEmail(email) || utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return uuid.Nil, ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash password: %w", err)
	}
	id := uuid.New()
	if err = a.repo.Create(ctx, model.User{ID: id, Nickname: nickname, Email: email, PasswordHash: string(hash)}); err != nil {
		return uuid.Nil, fmt.Errorf("register user: %w", err)
	}
	return id, nil
}
func (a *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(email)
	if !validEmail(email) || password == "" {
		return "", ErrInvalidCredentials
	}
	u, err := a.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("find user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: u.ID.String(), ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute))})
	signed, err := t.SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// VerifyToken validates an access token and returns its user ID.
func (a *AuthService) VerifyToken(token string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) { return a.secret, nil }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid {
		return uuid.Nil, ErrInvalidCredentials
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidCredentials
	}
	return id, nil
}
