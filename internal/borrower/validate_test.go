package borrower

import (
	"errors"
	"testing"

	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/apperr"
)

func TestNormalizePhone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "local prefix", in: "081234567890", want: "+6281234567890"},
		{name: "plus country", in: "+6281234567890", want: "+6281234567890"},
		{name: "country without plus", in: "6281234567890", want: "+6281234567890"},
		{name: "spaces and dashes", in: "0812-3456-7890", want: "+6281234567890"},
		{name: "too short", in: "0812", wantErr: true},
		{name: "not indonesian", in: "+15551234567", wantErr: true},
		{name: "empty", in: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizePhone(tc.in)
			if tc.wantErr {
				if !errors.Is(err, apperr.ErrValidation) {
					t.Fatalf("expected validation error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeAndValidateRegister(t *testing.T) {
	t.Parallel()

	email := "ada@example.com"
	in, err := normalizeAndValidateRegister(RegisterInput{
		FullName:    "  Ada Lovelace  ",
		PhoneNumber: "081234567890",
		Email:       &email,
		Password:    "supersecret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.FullName != "Ada Lovelace" {
		t.Fatalf("full name not trimmed: %q", in.FullName)
	}
	if in.PhoneNumber != "+6281234567890" {
		t.Fatalf("phone not normalized: %q", in.PhoneNumber)
	}

	_, err = normalizeAndValidateRegister(RegisterInput{
		FullName:    "",
		PhoneNumber: "081234567890",
		Password:    "supersecret",
	})
	if !errors.Is(err, apperr.ErrValidation) {
		t.Fatalf("expected validation error for empty name, got %v", err)
	}

	_, err = normalizeAndValidateRegister(RegisterInput{
		FullName:    "Ada",
		PhoneNumber: "081234567890",
		Password:    "short",
	})
	if !errors.Is(err, apperr.ErrValidation) {
		t.Fatalf("expected validation error for short password, got %v", err)
	}
}
