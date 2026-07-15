package purchaseorder

import "errors"

var (
	ErrBusinessNotResolved        = errors.New("business not resolved")
	ErrInvalidPurchaseOrderInput  = errors.New("invalid purchase order input")
	ErrPurchaseOrderNotFound      = errors.New("purchase order not found")
	ErrPurchaseOrderCannotDelete  = errors.New("purchase order cannot be deleted in its current status")
	ErrInvalidPurchaseReturnInput = errors.New("invalid purchase return input")
	ErrPurchaseReturnNotFound     = errors.New("purchase return not found")
)
