package warranty

import "errors"

var ErrWarrantyAlreadyExists = errors.New("warranty already exists")
var ErrWarrantyNotFound = errors.New("warranty not found")
var ErrInvalidWarrantyInput = errors.New("invalid warranty input")
var ErrBusinessNotResolved = errors.New("business not resolved")
