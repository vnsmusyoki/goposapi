package sales

import "errors"

var (
	ErrBusinessNotResolved = errors.New("business not resolved")
	ErrInvalidSaleInput     = errors.New("invalid sale input")
	ErrSaleNotFound         = errors.New("sale not found")
)
