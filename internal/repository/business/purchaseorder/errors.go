package purchaseorder

import "errors"

var (
	ErrBusinessNotResolved       = errors.New("business not resolved")
	ErrInvalidPurchaseOrderInput = errors.New("invalid purchase order input")
	ErrPurchaseOrderNotFound     = errors.New("purchase order not found")
)
