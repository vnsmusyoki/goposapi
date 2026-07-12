package unit

import "errors"

var ErrBusinessUnitAlreadyExists = errors.New("business unit already exists")
var ErrBusinessUnitNotFound = errors.New("business unit not found")
var ErrInvalidBusinessUnitInput = errors.New("invalid business unit input")
var ErrBusinessNotResolved = errors.New("business not resolved")
