package borrower

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/apperr"
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

type row struct {
	ID           uuid.UUID      `db:"id"`
	FullName     string         `db:"full_name"`
	PhoneNumber  string         `db:"phone_number"`
	Email        sql.NullString `db:"email"`
	PasswordHash string         `db:"password_hash"`
	KYCStatus    string         `db:"kyc_status"`
	CreatedAt    sql.NullTime   `db:"created_at"`
	UpdatedAt    sql.NullTime   `db:"updated_at"`
}

func (r *Repository) Create(ctx context.Context, q sqlx.QueryerContext, b Borrower) (*Borrower, error) {
	var out row
	err := sqlx.GetContext(ctx, q, &out, `
		INSERT INTO borrowers (full_name, phone_number, email, password_hash, kyc_status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, full_name, phone_number, email, password_hash, kyc_status, created_at, updated_at
	`, b.FullName, b.PhoneNumber, b.Email, b.PasswordHash, b.KYCStatus)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apperr.Conflict("phone_number is already registered")
		}
		return nil, fmt.Errorf("create borrower: %w", err)
	}
	return mapRow(out), nil
}

func (r *Repository) GetByID(ctx context.Context, q sqlx.QueryerContext, id uuid.UUID) (*Borrower, error) {
	var out row
	err := sqlx.GetContext(ctx, q, &out, `
		SELECT id, full_name, phone_number, email, password_hash, kyc_status, created_at, updated_at
		FROM borrowers
		WHERE id = $1
	`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("borrower not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get borrower: %w", err)
	}
	return mapRow(out), nil
}

func (r *Repository) GetByPhone(ctx context.Context, q sqlx.QueryerContext, phone string) (*Borrower, error) {
	var out row
	err := sqlx.GetContext(ctx, q, &out, `
		SELECT id, full_name, phone_number, email, password_hash, kyc_status, created_at, updated_at
		FROM borrowers
		WHERE phone_number = $1
	`, phone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("borrower not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get borrower by phone: %w", err)
	}
	return mapRow(out), nil
}

func (r *Repository) UpdateKYCStatus(ctx context.Context, q sqlx.QueryerContext, id uuid.UUID, status string) (*Borrower, error) {
	var out row
	err := sqlx.GetContext(ctx, q, &out, `
		UPDATE borrowers
		SET kyc_status = $2
		WHERE id = $1
		RETURNING id, full_name, phone_number, email, password_hash, kyc_status, created_at, updated_at
	`, id, status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("borrower not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update kyc status: %w", err)
	}
	return mapRow(out), nil
}

func mapRow(in row) *Borrower {
	b := &Borrower{
		ID:           in.ID,
		FullName:     in.FullName,
		PhoneNumber:  in.PhoneNumber,
		PasswordHash: in.PasswordHash,
		KYCStatus:    in.KYCStatus,
	}
	if in.Email.Valid {
		email := in.Email.String
		b.Email = &email
	}
	if in.CreatedAt.Valid {
		b.CreatedAt = in.CreatedAt.Time
	}
	if in.UpdatedAt.Valid {
		b.UpdatedAt = in.UpdatedAt.Time
	}
	return b
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
