package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/suwarnold/DigitalLendingPlatform/internal/outbox"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/config"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/database"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
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

	w := &outbox.Worker{
		DB:        db,
		Repo:      outbox.NewRepository(db),
		Publisher: outbox.NoopPublisher{Logger: log},
		Logger:    log,
		Interval:  cfg.WorkerPollInterval,
		BatchSize: cfg.WorkerBatchSize,
	}

	log.Info("worker started", "poll_interval", cfg.WorkerPollInterval.String(), "batch_size", cfg.WorkerBatchSize)
	if err := w.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
