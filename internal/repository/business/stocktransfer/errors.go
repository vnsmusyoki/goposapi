package stocktransfer

import "errors"

var ErrStockTransferNotFound = errors.New("stock transfer not found")
var ErrBusinessNotResolved = errors.New("business not resolved")
var ErrInvalidStockTransferInput = errors.New("invalid stock transfer input")
