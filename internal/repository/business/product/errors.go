package product

import "errors"

var ErrProductAlreadyExists = errors.New("product already exists")
var ErrProductNotFound = errors.New("product not found")
var ErrInvalidProductInput = errors.New("invalid product input")
var ErrBusinessNotResolved = errors.New("business not resolved")
var ErrInvalidComboProduct = errors.New("invalid combo item product")
