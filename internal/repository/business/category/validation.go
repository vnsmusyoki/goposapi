package category

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const maxCategoryImageBytes = 5 * 1024 * 1024

func ValidateCategoryImageDataURL(imageURL string) error {
	trimmed := strings.TrimSpace(imageURL)
	if trimmed == "" {
		return nil
	}

	parts := strings.SplitN(trimmed, ",", 2)
	if len(parts) != 2 {
		return ErrInvalidCategoryImage
	}

	meta := strings.ToLower(strings.TrimSpace(parts[0]))
	if !strings.HasPrefix(meta, "data:image/") || !strings.HasSuffix(meta, ";base64") {
		return ErrCategoryImageTypeNotAllowed
	}

	if !strings.HasPrefix(meta, "data:image/png;base64") && !strings.HasPrefix(meta, "data:image/jpeg;base64") && !strings.HasPrefix(meta, "data:image/jpg;base64") {
		return ErrCategoryImageTypeNotAllowed
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrInvalidCategoryImage
	}

	if len(decoded) > maxCategoryImageBytes {
		return ErrCategoryImageTooLarge
	}

	return nil
}

func normalizeCategoryImageDataURL(imageURL string) (string, error) {
	trimmed := strings.TrimSpace(imageURL)
	if trimmed == "" {
		return "", nil
	}

	if err := ValidateCategoryImageDataURL(trimmed); err != nil {
		return "", err
	}

	return trimmed, nil
}

func CategoryImageValidationMessage(err error) string {
	switch err {
	case ErrCategoryImageTooLarge:
		return fmt.Sprintf("Image must be smaller than %d MB.", maxCategoryImageBytes/(1024*1024))
	case ErrCategoryImageTypeNotAllowed:
		return "Only PNG and JPEG images are allowed."
	case ErrInvalidCategoryImage:
		return "The uploaded image data is invalid."
	default:
		return "The uploaded image is invalid."
	}
}
