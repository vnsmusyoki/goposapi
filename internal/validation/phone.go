package validation

import (
	"errors"
	"regexp"
	"strings"
)

const PhoneNumberMaxDigits = 10

var (
	ErrPhoneNumberRequired = errors.New("phone number is required")
	ErrInvalidPhoneNumber  = errors.New("phone number must start with 0 and contain 10 digits")
)

var nonDigitPattern = regexp.MustCompile(`\D+`)

func NormalizePhoneNumber(value string) string {
	digits := nonDigitPattern.ReplaceAllString(strings.TrimSpace(value), "")
	if digits == "" {
		return ""
	}

	if !strings.HasPrefix(digits, "0") {
		digits = "0" + digits
	}

	if len(digits) > PhoneNumberMaxDigits {
		digits = digits[:PhoneNumberMaxDigits]
	}

	return digits
}

func ValidatePhoneNumber(value string, required bool) error {
	normalized := NormalizePhoneNumber(value)
	if normalized == "" {
		if required {
			return ErrPhoneNumberRequired
		}
		return nil
	}

	if len(normalized) != PhoneNumberMaxDigits || !strings.HasPrefix(normalized, "0") {
		return ErrInvalidPhoneNumber
	}

	return nil
}
