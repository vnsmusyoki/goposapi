package customer

import "errors"

var ErrBusinessNotResolved = errors.New("business not resolved")
var ErrInvalidBusinessCustomerInput = errors.New("invalid business customer input")
var ErrBusinessCustomerCodeAlreadyExists = errors.New("business customer code already exists")
