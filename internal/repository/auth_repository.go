package repository

import (
	"database/sql"

	"github.com/pieremarie/chessmobile/internal/model"
)

type PostgresRepository struct {
	db *sql.DB
}

// TODO: write here some code
func (p *PostgresRepository) CreateUser(user model.User) error {
	return nil
}

// TODO: change the name and write some code there too
func (p *PostgresRepository) Login(email, passwordhash string) error {
	return nil
}
