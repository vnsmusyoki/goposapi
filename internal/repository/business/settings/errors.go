package settings

import "errors"

var ErrBusinessNotResolved = errors.New("business not resolved")
var ErrInvalidBusinessSettingsInput = errors.New("invalid business settings input")
var ErrInvalidBusinessSettingsLogo = errors.New("invalid business settings logo")
var ErrBusinessSettingsLogoTooLarge = errors.New("business settings logo too large")
var ErrBusinessSettingsLogoTypeNotAllowed = errors.New("business settings logo type not allowed")
var ErrInvalidBusinessInvoiceSettingsInput = errors.New("invalid business invoice settings input")
var ErrBusinessInvoiceSettingsLogoInvalid = errors.New("business invoice settings logo invalid")
var ErrBusinessInvoiceSettingsLogoTypeNotAllowed = errors.New("business invoice settings logo type not allowed")
var ErrBusinessInvoiceSettingsLogoTooLarge = errors.New("business invoice settings logo too large")
var ErrBusinessInvoiceSettingsDuplicateCode = errors.New("business invoice settings duplicate code")
var ErrBusinessInvoiceSettingsNotFound = errors.New("business invoice settings not found")
