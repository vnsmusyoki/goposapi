package location

import "errors"

var ErrBusinessLocationAlreadyExists = errors.New("business location already exists")
var ErrBusinessLocationNotFound = errors.New("business location not found")
var ErrInvalidBusinessLocationInput = errors.New("invalid business location input")
var ErrBusinessNotResolved = errors.New("business not resolved")
