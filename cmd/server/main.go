package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pieremarie/chessmobile/internal/chesslogic"
	"github.com/pieremarie/chessmobile/internal/handler"
	"github.com/pieremarie/chessmobile/internal/hub"
	"github.com/pieremarie/chessmobile/internal/repository"
	"github.com/pieremarie/chessmobile/internal/service"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = db.Ping(context.Background()); err != nil {
		log.Fatal(err)
	}
	a, err := service.NewAuthService(repository.NewUserRepository(db), os.Getenv("JWT_SECRET"))
	if err != nil {
		log.Fatal(err)
	}
	h := handler.NewAuthHandler(a)
	r := gin.Default()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	gameStore := repository.NewGameRepository(db)
	games := hub.New(gameStore, func(id uuid.UUID) (chesslogic.Game, error) {
		fen, err := gameStore.FEN(context.Background(), id)
		if err != nil {
			return nil, err
		}
		return chesslogic.FromFEN(fen)
	})
	r.GET("/ws/games/:id", handler.NewGameHandler(games, a).Serve)
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err = http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
