package service

import (
	"fmt"

	"github.com/pieremarie/chessmobile/internal/model"
)

type AuthRepository interface {
	CreateUser(user model.User) error
	Login(email, passwordhash string) error
}

type AuthService struct {
	repo AuthRepository
}

func (a *AuthService) CreateUser(user model.User) error {
	if user.Email == "" || user.Name == "" {
		// TODO: change the name of error and make it a custom error
		return fmt.Errorf("incorrect data")
	}

	// The defualt estimation of ELO of the new player will be around 800
	user.Rating = 800

	err := a.repo.CreateUser(user)

	return err
}

func (a *AuthService) Login(email, passwordhash string) (string, error) {
	// TODO: make jwt function

	// jwt, err := ...(...)

	return "", nil
}
