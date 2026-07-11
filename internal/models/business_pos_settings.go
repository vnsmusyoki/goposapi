package models

type BusinessPosSettings struct {
	ID                                    string `json:"id"`
	DisableMultiplePay                    bool   `json:"disableMultiplePay"`
	DisableDraft                          bool   `json:"disableDraft"`
	DisableExpressCheckout                bool   `json:"disableExpressCheckout"`
	DisableDiscount                       bool   `json:"disableDiscount"`
	DisableOrderTax                       bool   `json:"disableOrderTax"`
	DisableCreditSaleButton               bool   `json:"disableCreditSaleButton"`
	DisableSuspendSale                    bool   `json:"disableSuspendSale"`
	SubtotalEditable                      bool   `json:"subtotalEditable"`
	HideProductSuggestion                 bool   `json:"hideProductSuggestion"`
	ShowPricingOnProductSuggestionTooltip bool   `json:"showPricingOnProductSuggestionTooltip"`
	HideRecentTransactions                bool   `json:"hideRecentTransactions"`
	EnableTransactionDateOnPosScreen      bool   `json:"enableTransactionDateOnPosScreen"`
	EnableWeighingScale                   bool   `json:"enableWeighingScale"`
	EnableServiceStaffInProductLine       bool   `json:"enableServiceStaffInProductLine"`
	IsServiceStaffRequired                bool   `json:"isServiceStaffRequired"`
	InvoiceScheme                         string `json:"invoiceScheme"`
	InvoiceLayout                         string `json:"invoiceLayout"`
	PrintInvoiceOnSuspend                 bool   `json:"printInvoiceOnSuspend"`
	Message                               string `json:"message"`
}
