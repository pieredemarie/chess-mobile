package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pieremarie/chessmobile/internal/chesslogic"
	"github.com/pieremarie/chessmobile/internal/config"
	"github.com/pieremarie/chessmobile/internal/handler"
	"github.com/pieremarie/chessmobile/internal/repository"
	"github.com/pieremarie/chessmobile/internal/service"
	"github.com/pieremarie/chessmobile/internal/timedhub"
	"github.com/pieremarie/chessmobile/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := run(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	root, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(root, 30*time.Second)
	defer cancel()
	db, err := pgxpool.New(startup, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer db.Close()
	if err = db.Ping(startup); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if err = migrations.Apply(startup, db); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	auth, err := service.NewAuthService(repository.NewUserRepository(db), cfg.JWTSecret)
	if err != nil {
		return fmt.Errorf("create auth service: %w", err)
	}
	router := routes(db, auth)
	server := &http.Server{Addr: cfg.Address(), Handler: router, ReadHeaderTimeout: 5 * time.Second}
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	slog.Info("server listening", "address", cfg.Address())
	select {
	case err = <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-root.Done():
		slog.Info("shutdown requested")
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = server.Shutdown(shutdown); err != nil {
			return fmt.Errorf("shutdown HTTP: %w", err)
		}
		if err = <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		slog.Info("server stopped")
		return nil
	}
}

func routes(db *pgxpool.Pool, auth *service.AuthService) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery(), func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http request", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds())
	})
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			slog.Error("health check failed", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	h := handler.NewAuthHandler(auth)
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	gameStore := repository.NewGameRepository(db)
	ratedStore := service.RatedStore{Games: gameStore, Ratings: repository.NewRatingRepository(db)}
	games := timedhub.NewWithTimeControl(ratedStore, func(id uuid.UUID) (chesslogic.Game, error) {
		fen, err := gameStore.FEN(context.Background(), id)
		if err != nil {
			return nil, err
		}
		return chesslogic.FromFEN(fen)
	}, 10*time.Minute, 0)
	r.GET("/ws/games/:id", handler.NewTimedGameHandler(games, auth).Serve)
	history := handler.NewHistoryHandler(service.NewHistoryService(repository.NewHistoryRepository(db)), auth)
	r.GET("/me", history.Profile)
	r.GET("/me/games", history.Games)
	r.GET("/me/games/:id", history.Game)
	matchmaking := &handler.MatchmakingHandler{Matchmaker: service.NewMatchmaker(gameStore), Auth: auth}
	r.POST("/matchmaking/search", matchmaking.Search)
	r.GET("/matchmaking/status", matchmaking.Status)
	r.DELETE("/matchmaking/search", matchmaking.Cancel)
	return r
}
