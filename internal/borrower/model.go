package borrower

import (
	"time"

	"github.com/google/uuid"
)

const (
	KYCPending  = "PENDING"
	KYCInReview = "IN_REVIEW"
	KYCVerified = "VERIFIED"
	KYCRejected = "REJECTED"
)

type Borrower struct {
	ID           uuid.UUID
	FullName     string
	PhoneNumber  string
	Email        *string
	PasswordHash string
	KYCStatus    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RegisterInput struct {
	FullName    string
	PhoneNumber string
	Email       *string
	Password    string
}

type LoginInput struct {
	PhoneNumber string
	Password    string
}

type PublicBorrower struct {
	ID          uuid.UUID `json:"id"`
	FullName    string    `json:"full_name"`
	PhoneNumber string    `json:"phone_number"`
	Email       *string   `json:"email"`
	KYCStatus   string    `json:"kyc_status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (b Borrower) Public() PublicBorrower {
	return PublicBorrower{
		ID:          b.ID,
		FullName:    b.FullName,
		PhoneNumber: b.PhoneNumber,
		Email:       b.Email,
		KYCStatus:   b.KYCStatus,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
