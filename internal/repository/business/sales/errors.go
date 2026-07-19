package sales

import "errors"

var (
	ErrBusinessNotResolved                 = errors.New("business not resolved")
	ErrInvalidSaleInput                    = errors.New("invalid sale input")
	ErrSaleNotFound                        = errors.New("sale not found")
	ErrSalesOrderCannotDelete              = errors.New("sales order cannot be deleted in its current status")
	ErrSalesOrderCannotUpdate              = errors.New("sales order cannot be updated in its current status")
	ErrSalesOrderStatusDefinitionNotFound  = errors.New("sales order status definition not found")
	ErrSalesOrderStatusRegressionNotAllowed = errors.New("sales order status regression not allowed")
)
