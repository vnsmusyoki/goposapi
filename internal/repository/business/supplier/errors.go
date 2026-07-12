package supplier

import "errors"

var ErrBusinessNotResolved = errors.New("business not resolved")
var ErrInvalidBusinessSupplierInput = errors.New("invalid business supplier input")
var ErrBusinessSupplierContactIDAlreadyExists = errors.New("business supplier contact id already exists")
