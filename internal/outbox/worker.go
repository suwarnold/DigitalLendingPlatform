package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/database"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/metrics"
)

type Publisher interface {
	Publish(ctx context.Context, events []Event) error
}

type Worker struct {
	DB        *sqlx.DB
	Repo      *Repository
	Publisher Publisher
	Logger    *slog.Logger
	Interval  time.Duration
	BatchSize int
}

func (w *Worker) Run(ctx context.Context) error {
	interval := w.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	batch := w.batchSize()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if err := w.process(ctx, batch); err != nil && ctx.Err() == nil {
		w.Logger.ErrorContext(ctx, "outbox poll failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.process(ctx, batch); err != nil && ctx.Err() == nil {
				w.Logger.ErrorContext(ctx, "outbox poll failed", "error", err)
			}
		}
	}
}

func (w *Worker) ProcessOnce(ctx context.Context) (int, error) {
	return w.processCount(ctx, w.batchSize())
}

func (w *Worker) batchSize() int {
	if w.BatchSize <= 0 {
		return 50
	}
	return w.BatchSize
}

func (w *Worker) process(ctx context.Context, batch int) error {
	_, err := w.processCount(ctx, batch)
	return err
}

func (w *Worker) processCount(ctx context.Context, batch int) (int, error) {
	var published int
	err := database.Transact(ctx, w.DB, func(tx *sqlx.Tx) error {
		events, err := w.Repo.ClaimUnpublished(ctx, tx, batch)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}

		if err := w.Publisher.Publish(ctx, events); err != nil {
			if incErr := w.Repo.IncrementAttempts(ctx, tx, eventIDs(events)); incErr != nil {
				return incErr
			}
			return err
		}
		if err := w.Repo.MarkPublished(ctx, tx, eventIDs(events)); err != nil {
			return err
		}
		published = len(events)
		return nil
	})
	if err != nil {
		return 0, err
	}
	if published > 0 {
		metrics.IncOutboxPublished(published)
		w.Logger.InfoContext(ctx, "outbox batch published", "count", published)
	}
	return published, nil
}

func eventIDs(events []Event) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}
