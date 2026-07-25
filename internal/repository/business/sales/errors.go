package sales

import "errors"

var (
	ErrBusinessNotResolved                  = errors.New("business not resolved")
	ErrInvalidSaleInput                     = errors.New("invalid sale input")
	ErrInvalidSalePayment                   = errors.New("invalid sale payment")
	ErrSalePaymentMethodNotEnabled          = errors.New("sale payment method not enabled")
	ErrSaleCreditCustomerRequired           = errors.New("customer is required for credit sale")
	ErrSaleCreditLimitExceeded              = errors.New("customer credit limit exceeded")
	ErrActiveCashRegisterRequired           = errors.New("active cash register required")
	ErrSaleNotFound                         = errors.New("sale not found")
	ErrSalesOrderCannotDelete               = errors.New("sales order cannot be deleted in its current status")
	ErrSalesOrderCannotUpdate               = errors.New("sales order cannot be updated in its current status")
	ErrSalesOrderStatusDefinitionNotFound   = errors.New("sales order status definition not found")
	ErrSalesOrderStatusRegressionNotAllowed = errors.New("sales order status regression not allowed")
)
