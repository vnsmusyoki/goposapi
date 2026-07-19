package cashregister

import "errors"

var (
	ErrBusinessNotResolved  = errors.New("business not resolved")
	ErrLocationNotFound     = errors.New("business location not found")
	ErrActiveRegisterExists = errors.New("active cash register already exists")
	ErrInvalidRegisterInput = errors.New("invalid cash register input")
)
