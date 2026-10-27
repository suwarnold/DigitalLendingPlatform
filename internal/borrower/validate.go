package borrower

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/apperr"
)

var nonDigit = regexp.MustCompile(`[\s\-()]`)

func normalizeAndValidateRegister(in RegisterInput) (RegisterInput, error) {
	in.FullName = strings.TrimSpace(in.FullName)
	if in.FullName == "" || utf8.RuneCountInString(in.FullName) > 200 {
		return RegisterInput{}, apperr.Validation("full_name must be between 1 and 200 characters")
	}

	phone, err := normalizePhone(in.PhoneNumber)
	if err != nil {
		return RegisterInput{}, err
	}
	in.PhoneNumber = phone

	if in.Email != nil {
		email := strings.TrimSpace(*in.Email)
		if email == "" {
			in.Email = nil
		} else {
			if _, err := mail.ParseAddress(email); err != nil {
				return RegisterInput{}, apperr.Validation("email is invalid")
			}
			in.Email = &email
		}
	}

	if err := validatePassword(in.Password); err != nil {
		return RegisterInput{}, err
	}
	return in, nil
}

func normalizePhone(raw string) (string, error) {
	phone := nonDigit.ReplaceAllString(strings.TrimSpace(raw), "")
	switch {
	case strings.HasPrefix(phone, "+62"):
		phone = "+62" + strings.TrimPrefix(phone, "+62")
	case strings.HasPrefix(phone, "62"):
		phone = "+" + phone
	case strings.HasPrefix(phone, "0"):
		phone = "+62" + strings.TrimPrefix(phone, "0")
	default:
		return "", apperr.Validation("phone_number must be an Indonesian number")
	}

	digits := strings.TrimPrefix(phone, "+62")
	if len(digits) < 8 || len(digits) > 13 || !allDigits(digits) {
		return "", apperr.Validation("phone_number is invalid")
	}
	return "+62" + digits, nil
}

func validatePassword(password string) error {
	if n := utf8.RuneCountInString(password); n < 8 || n > 72 {
		return apperr.Validation("password must be between 8 and 72 characters")
	}
	return nil
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
