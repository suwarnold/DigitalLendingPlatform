package appctx

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	borrowerIDKey
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func WithBorrowerID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, borrowerIDKey, id)
}

func BorrowerID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(borrowerIDKey).(uuid.UUID)
	return id, ok && id != uuid.Nil
}
