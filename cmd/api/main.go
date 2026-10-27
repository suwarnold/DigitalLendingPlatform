package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/suwarnold/DigitalLendingPlatform/internal/app"
	"github.com/suwarnold/DigitalLendingPlatform/internal/borrower"
	"github.com/suwarnold/DigitalLendingPlatform/internal/outbox"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/auth"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/config"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/database"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/httpserver"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/idempotency"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	if err := cfg.RequireJWTSecret(); err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Error("run migrations", "error", err)
		os.Exit(1)
	}

	tokens := auth.NewTokens(cfg.JWTSecret, cfg.JWTTTL)
	svc := &borrower.Service{
		DB:          db,
		Borrowers:   borrower.NewRepository(),
		Outbox:      outbox.NewRepository(db),
		Idempotency: idempotency.NewRepository(),
		Tokens:      tokens,
	}
	handler := app.NewRouter(app.RouterDeps{
		DB:              db,
		Auth:            tokens,
		BorrowerHandler: &borrower.Handler{Service: svc},
	})

	srv := httpserver.New(cfg.HTTPAddr, handler)
	errCh := make(chan error, 1)
	go func() {
		log.Info("api listening", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("api shutdown", "error", err)
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("api server", "error", err)
			os.Exit(1)
		}
	}
}
