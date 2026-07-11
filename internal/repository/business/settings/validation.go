package settings

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const maxBusinessSettingsLogoBytes = 5 * 1024 * 1024

func ValidateBusinessSettingsLogoDataURL(imageURL string) error {
	trimmed := strings.TrimSpace(imageURL)
	if trimmed == "" {
		return nil
	}

	parts := strings.SplitN(trimmed, ",", 2)
	if len(parts) != 2 {
		return ErrInvalidBusinessSettingsLogo
	}

	meta := strings.ToLower(strings.TrimSpace(parts[0]))
	if !strings.HasPrefix(meta, "data:image/") || !strings.HasSuffix(meta, ";base64") {
		return ErrBusinessSettingsLogoTypeNotAllowed
	}

	if !strings.HasPrefix(meta, "data:image/png;base64") &&
		!strings.HasPrefix(meta, "data:image/jpeg;base64") &&
		!strings.HasPrefix(meta, "data:image/jpg;base64") {
		return ErrBusinessSettingsLogoTypeNotAllowed
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrInvalidBusinessSettingsLogo
	}

	if len(decoded) > maxBusinessSettingsLogoBytes {
		return ErrBusinessSettingsLogoTooLarge
	}

	return nil
}

func normalizeBusinessSettingsLogoDataURL(imageURL string) (string, error) {
	trimmed := strings.TrimSpace(imageURL)
	if trimmed == "" {
		return "", nil
	}

	if err := ValidateBusinessSettingsLogoDataURL(trimmed); err != nil {
		return "", err
	}

	return trimmed, nil
}

func BusinessSettingsLogoValidationMessage(err error) string {
	switch err {
	case ErrBusinessSettingsLogoTooLarge:
		return fmt.Sprintf("Logo must be smaller than %d MB.", maxBusinessSettingsLogoBytes/(1024*1024))
	case ErrBusinessSettingsLogoTypeNotAllowed:
		return "Only PNG and JPEG images are allowed."
	case ErrInvalidBusinessSettingsLogo:
		return "The uploaded logo data is invalid."
	default:
		return "The uploaded logo is invalid."
	}
}
