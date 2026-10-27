package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	BorrowerID  uuid.UUID `json:"borrower_id"`
	PhoneNumber string    `json:"phone_number"`
	jwt.RegisteredClaims
}

type Tokens struct {
	Issuer string
	Secret []byte
	TTL    time.Duration
}

func NewTokens(secret string, ttl time.Duration) *Tokens {
	return &Tokens{
		Issuer: "digital-lending-platform",
		Secret: []byte(secret),
		TTL:    ttl,
	}
}

func (t *Tokens) Issue(borrowerID uuid.UUID, phone string, now time.Time) (string, error) {
	claims := Claims{
		BorrowerID:  borrowerID,
		PhoneNumber: phone,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.Issuer,
			Subject:   borrowerID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.TTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.Secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (t *Tokens) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.BorrowerID == uuid.Nil {
		return nil, fmt.Errorf("invalid token subject")
	}
	return claims, nil
}

func (t *Tokens) ExpiresInSeconds() int {
	return int(t.TTL.Seconds())
}
