package idempotency

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Record struct {
	Scope          string
	IdempotencyKey string
	RequestHash    string
	ResponseStatus *int
	ResponseBody   json.RawMessage
	CreatedAt      time.Time
	ExpiresAt      *time.Time
}

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) Get(ctx context.Context, q sqlx.QueryerContext, scope, key string) (*Record, error) {
	var row struct {
		Scope          string          `db:"scope"`
		IdempotencyKey string          `db:"idempotency_key"`
		RequestHash    string          `db:"request_hash"`
		ResponseStatus sql.NullInt64   `db:"response_status"`
		ResponseBody   json.RawMessage `db:"response_body"`
		CreatedAt      time.Time       `db:"created_at"`
		ExpiresAt      *time.Time      `db:"expires_at"`
	}

	err := sqlx.GetContext(ctx, q, &row, `
		SELECT scope, idempotency_key, request_hash, response_status, response_body, created_at, expires_at
		FROM idempotency_records
		WHERE scope = $1 AND idempotency_key = $2
	`, scope, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get idempotency record: %w", err)
	}

	rec := &Record{
		Scope:          row.Scope,
		IdempotencyKey: row.IdempotencyKey,
		RequestHash:    row.RequestHash,
		ResponseBody:   row.ResponseBody,
		CreatedAt:      row.CreatedAt,
		ExpiresAt:      row.ExpiresAt,
	}
	if row.ResponseStatus.Valid {
		status := int(row.ResponseStatus.Int64)
		rec.ResponseStatus = &status
	}
	return rec, nil
}

func (r *Repository) Save(ctx context.Context, q sqlx.ExtContext, rec Record) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO idempotency_records (
			scope, idempotency_key, request_hash, response_status, response_body, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (scope, idempotency_key) DO UPDATE
		SET request_hash = EXCLUDED.request_hash,
		    response_status = EXCLUDED.response_status,
		    response_body = EXCLUDED.response_body,
		    expires_at = EXCLUDED.expires_at
	`, rec.Scope, rec.IdempotencyKey, rec.RequestHash, rec.ResponseStatus, rec.ResponseBody, rec.ExpiresAt)
	if err != nil {
		return fmt.Errorf("save idempotency record: %w", err)
	}
	return nil
}
