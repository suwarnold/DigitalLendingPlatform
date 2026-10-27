package outbox

import (
	"context"
	"log/slog"
)

// NoopPublisher acknowledges events without sending them anywhere.
// Later phases can replace this with a durable queue publisher.
type NoopPublisher struct {
	Logger *slog.Logger
}

func (p NoopPublisher) Publish(ctx context.Context, events []Event) error {
	log := p.Logger
	if log == nil {
		log = slog.Default()
	}
	for _, event := range events {
		log.InfoContext(ctx, "outbox event published",
			"event_id", event.ID,
			"event_type", event.EventType,
			"aggregate_type", event.AggregateType,
			"aggregate_id", event.AggregateID,
		)
	}
	return nil
}
