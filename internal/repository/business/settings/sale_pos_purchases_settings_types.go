package settings

type UpdateBusinessSaleSettingsInput struct {
	BusinessID                     string
	DefaultSaleDiscount            float64
	DefaultSaleTax                 float64
	SaleItemAdditionMethod         string
	EnableSaleOrder                bool
	IsPayTermRequired              bool
	SalePriceIsMinimumSellingPrice bool
	EnableSaleCommissionAgent      bool
	CommissionCalculationType      string
	IsCommissionAgentRequired      bool
}

type UpdateBusinessPosSettingsInput struct {
	BusinessID                            string
	DisableMultiplePay                    bool
	DisableDraft                          bool
	DisableExpressCheckout                bool
	DisableDiscount                       bool
	DisableOrderTax                       bool
	DisableCreditSaleButton               bool
	DisableSuspendSale                    bool
	SubtotalEditable                      bool
	HideProductSuggestion                 bool
	ShowPricingOnProductSuggestionTooltip bool
	HideRecentTransactions                bool
	EnableTransactionDateOnPosScreen      bool
	EnableWeighingScale                   bool
	EnableServiceStaffInProductLine       bool
	IsServiceStaffRequired                bool
	InvoiceScheme                         string
	InvoiceLayout                         string
	PrintInvoiceOnSuspend                 bool
}

type UpdateBusinessPurchasesSettingsInput struct {
	BusinessID                                  string
	EnableEditingProductPriceFromPurchaseScreen bool
	EnablePurchaseStatus                        bool
	EnableLotNumber                             bool
	EnablePurchaseOrder                         bool
}
