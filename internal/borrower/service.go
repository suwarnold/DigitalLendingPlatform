package borrower

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/suwarnold/DigitalLendingPlatform/internal/outbox"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/apperr"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/auth"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/database"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/idempotency"
)

const registerScope = "borrowers.register"

type Service struct {
	DB          *sqlx.DB
	Borrowers   *Repository
	Outbox      *outbox.Repository
	Idempotency *idempotency.Repository
	Tokens      *auth.Tokens
}

func (s *Service) Register(ctx context.Context, in RegisterInput, idempotencyKey, requestHash string) (*Borrower, int, []byte, error) {
	normalized, err := normalizeAndValidateRegister(in)
	if err != nil {
		return nil, 0, nil, err
	}

	var (
		created *Borrower
		status  int
		body    []byte
	)

	err = database.Transact(ctx, s.DB, func(tx *sqlx.Tx) error {
		if idempotencyKey != "" {
			existing, err := s.Idempotency.Get(ctx, tx, registerScope, idempotencyKey)
			if err != nil {
				return err
			}
			if existing != nil {
				if existing.RequestHash != requestHash {
					return apperr.ErrIdempotencyConflict
				}
				if existing.ResponseStatus != nil && len(existing.ResponseBody) > 0 {
					status = *existing.ResponseStatus
					body = existing.ResponseBody
					var pub PublicBorrower
					if err := json.Unmarshal(existing.ResponseBody, &pub); err == nil {
						created = &Borrower{
							ID:          pub.ID,
							FullName:    pub.FullName,
							PhoneNumber: pub.PhoneNumber,
							Email:       pub.Email,
							KYCStatus:   pub.KYCStatus,
							CreatedAt:   pub.CreatedAt,
							UpdatedAt:   pub.UpdatedAt,
						}
					}
					return nil
				}
			}
		}

		hash, err := auth.HashPassword(normalized.Password)
		if err != nil {
			return err
		}

		created, err = s.Borrowers.Create(ctx, tx, Borrower{
			FullName:     normalized.FullName,
			PhoneNumber:  normalized.PhoneNumber,
			Email:        normalized.Email,
			PasswordHash: hash,
			KYCStatus:    KYCPending,
		})
		if err != nil {
			return err
		}

		if err := s.Outbox.Append(ctx, tx, "borrower", created.ID, "BorrowerRegistered", map[string]any{
			"borrower_id": created.ID,
		}); err != nil {
			return err
		}

		status = 201
		body, err = json.Marshal(created.Public())
		if err != nil {
			return err
		}

		if idempotencyKey != "" {
			expires := time.Now().Add(24 * time.Hour)
			return s.Idempotency.Save(ctx, tx, idempotency.Record{
				Scope:          registerScope,
				IdempotencyKey: idempotencyKey,
				RequestHash:    requestHash,
				ResponseStatus: &status,
				ResponseBody:   body,
				ExpiresAt:      &expires,
			})
		}
		return nil
	})
	if err != nil {
		return nil, 0, nil, err
	}
	return created, status, body, nil
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*TokenResponse, error) {
	phone, err := normalizePhone(in.PhoneNumber)
	if err != nil {
		return nil, err
	}
	if in.Password == "" {
		return nil, apperr.Unauthorized("invalid phone_number or password")
	}

	b, err := s.Borrowers.GetByPhone(ctx, s.DB, phone)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, apperr.Unauthorized("invalid phone_number or password")
		}
		return nil, err
	}
	if err := auth.ComparePassword(b.PasswordHash, in.Password); err != nil {
		return nil, apperr.Unauthorized("invalid phone_number or password")
	}

	token, err := s.Tokens.Issue(b.ID, b.PhoneNumber, time.Now())
	if err != nil {
		return nil, err
	}
	return &TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   s.Tokens.ExpiresInSeconds(),
	}, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Borrower, error) {
	return s.Borrowers.GetByID(ctx, s.DB, id)
}

func (s *Service) VerifyIdentity(ctx context.Context, id uuid.UUID) (*Borrower, error) {
	var updated *Borrower
	err := database.Transact(ctx, s.DB, func(tx *sqlx.Tx) error {
		current, err := s.Borrowers.GetByID(ctx, tx, id)
		if err != nil {
			return err
		}

		switch current.KYCStatus {
		case KYCVerified:
			return apperr.Conflict("identity is already verified")
		case KYCInReview:
			updated = current
			return nil
		}

		updated, err = s.Borrowers.UpdateKYCStatus(ctx, tx, id, KYCInReview)
		if err != nil {
			return err
		}
		return s.Outbox.Append(ctx, tx, "borrower", updated.ID, "IdentityVerificationRequested", map[string]any{
			"borrower_id": updated.ID,
			"kyc_status":  updated.KYCStatus,
		})
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}
