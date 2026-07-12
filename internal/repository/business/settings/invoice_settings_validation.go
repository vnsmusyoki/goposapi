package settings

import (
	"net/url"
	"strings"
)

func normalizeBusinessInvoiceSettingsLogoURL(imageURL string) (string, error) {
	trimmed := strings.TrimSpace(imageURL)
	if trimmed == "" {
		return "", nil
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "data:image/") {
		if err := ValidateBusinessSettingsLogoDataURL(trimmed); err != nil {
			switch err {
			case ErrInvalidBusinessSettingsLogo:
				return "", ErrBusinessInvoiceSettingsLogoInvalid
			case ErrBusinessSettingsLogoTooLarge:
				return "", ErrBusinessInvoiceSettingsLogoTooLarge
			case ErrBusinessSettingsLogoTypeNotAllowed:
				return "", ErrBusinessInvoiceSettingsLogoTypeNotAllowed
			default:
				return "", ErrBusinessInvoiceSettingsLogoInvalid
			}
		}

		return trimmed, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrBusinessInvoiceSettingsLogoInvalid
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return trimmed, nil
	default:
		return "", ErrBusinessInvoiceSettingsLogoTypeNotAllowed
	}
}

func BusinessInvoiceSettingsLogoValidationMessage(err error) string {
	switch err {
	case ErrBusinessInvoiceSettingsLogoTooLarge:
		return "Invoice logo must be smaller than 5 MB."
	case ErrBusinessInvoiceSettingsLogoTypeNotAllowed:
		return "Only PNG and JPEG invoice logos or public URLs are allowed."
	case ErrBusinessInvoiceSettingsLogoInvalid:
		return "The invoice logo value is invalid."
	default:
		return "The invoice logo value is invalid."
	}
}
