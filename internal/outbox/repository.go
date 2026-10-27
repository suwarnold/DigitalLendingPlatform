package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct{}

func NewRepository(_ *sqlx.DB) *Repository {
	return &Repository{}
}

func (r *Repository) Append(ctx context.Context, q sqlx.ExtContext, aggregateType string, aggregateID uuid.UUID, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	_, err = q.ExecContext(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
		VALUES ($1, $2, $3, $4)
	`, aggregateType, aggregateID, eventType, body)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

func (r *Repository) ClaimUnpublished(ctx context.Context, tx *sqlx.Tx, limit int) ([]Event, error) {
	var rows []struct {
		ID            uuid.UUID       `db:"id"`
		AggregateType string          `db:"aggregate_type"`
		AggregateID   uuid.UUID       `db:"aggregate_id"`
		EventType     string          `db:"event_type"`
		Payload       json.RawMessage `db:"payload"`
		CreatedAt     time.Time       `db:"created_at"`
		PublishedAt   *time.Time      `db:"published_at"`
		Attempts      int             `db:"attempts"`
	}

	err := tx.SelectContext(ctx, &rows, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, published_at, attempts
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}

	events := make([]Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, Event{
			ID:            row.ID,
			AggregateType: row.AggregateType,
			AggregateID:   row.AggregateID,
			EventType:     row.EventType,
			Payload:       row.Payload,
			CreatedAt:     row.CreatedAt,
			PublishedAt:   row.PublishedAt,
			Attempts:      row.Attempts,
		})
	}
	return events, nil
}

func (r *Repository) MarkPublished(ctx context.Context, tx *sqlx.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`
		UPDATE outbox_events
		SET published_at = now(), attempts = attempts + 1
		WHERE id IN (?)
	`, ids)
	if err != nil {
		return fmt.Errorf("bind published ids: %w", err)
	}
	query = tx.Rebind(query)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	return nil
}

func (r *Repository) IncrementAttempts(ctx context.Context, tx *sqlx.Tx, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`
		UPDATE outbox_events
		SET attempts = attempts + 1
		WHERE id IN (?)
	`, ids)
	if err != nil {
		return fmt.Errorf("bind attempt ids: %w", err)
	}
	query = tx.Rebind(query)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("increment outbox attempts: %w", err)
	}
	return nil
}
