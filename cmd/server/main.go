package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pieremarie/chessmobile/internal/handler"
	"github.com/pieremarie/chessmobile/internal/repository"
	"github.com/pieremarie/chessmobile/internal/service"
	"log"
	"net/http"
	"os"
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
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err = http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
