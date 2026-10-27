package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIssueAndParse(t *testing.T) {
	t.Parallel()

	tokens := NewTokens("test-secret", time.Hour)
	id := uuid.New()
	signed, err := tokens.Issue(id, "+6281234567890", time.Now())
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	claims, err := tokens.Parse(signed)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.BorrowerID != id {
		t.Fatalf("borrower id = %s want %s", claims.BorrowerID, id)
	}
}

func TestParseRejectsWrongSecretAndExpired(t *testing.T) {
	t.Parallel()

	tokens := NewTokens("test-secret", time.Hour)
	id := uuid.New()
	signed, err := tokens.Issue(id, "+6281234567890", time.Now())
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	other := NewTokens("other-secret", time.Hour)
	if _, err := other.Parse(signed); err == nil {
		t.Fatal("expected wrong-secret token to fail")
	}

	expired, err := tokens.Issue(id, "+6281234567890", time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("issue expired: %v", err)
	}
	if _, err := tokens.Parse(expired); err == nil {
		t.Fatal("expected expired token to fail")
	}
}

func TestHashAndComparePassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("supersecret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := ComparePassword(hash, "supersecret"); err != nil {
		t.Fatalf("compare matching password: %v", err)
	}
	if err := ComparePassword(hash, "wrong"); err == nil {
		t.Fatal("expected mismatch")
	}
}
