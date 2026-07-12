package brand

import "errors"

var ErrBrandAlreadyExists = errors.New("brand already exists")
var ErrBrandNotFound = errors.New("brand not found")
var ErrInvalidBrandInput = errors.New("invalid brand input")
var ErrBusinessNotResolved = errors.New("business not resolved")
