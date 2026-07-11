package settings

import "errors"

var ErrBusinessNotResolved = errors.New("business not resolved")
var ErrInvalidBusinessSettingsInput = errors.New("invalid business settings input")
var ErrInvalidBusinessSettingsLogo = errors.New("invalid business settings logo")
var ErrBusinessSettingsLogoTooLarge = errors.New("business settings logo too large")
var ErrBusinessSettingsLogoTypeNotAllowed = errors.New("business settings logo type not allowed")
