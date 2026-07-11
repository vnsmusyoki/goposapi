package models

type BusinessSettings struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	StartDate               string   `json:"startDate"`
	DefaultProfitPercentage *float64 `json:"defaultProfitPercentage,omitempty"`
	Currency                string   `json:"currency"`
	CurrencySymbolPlacement string   `json:"currencySymbolPlacement"`
	Timezone                string   `json:"timezone"`
	LogoURL                 string   `json:"logoUrl"`
	FinancialYearStartMonth string   `json:"financialYearStartMonth"`
	StockAccountingMethod   string   `json:"stockAccountingMethod"`
	TransactionEditDays     *int     `json:"transactionEditDays,omitempty"`
	DateFormat              string   `json:"dateFormat"`
	TimeFormat              string   `json:"timeFormat"`
	CurrencyPrecision       *int     `json:"currencyPrecision,omitempty"`
	QuantityPrecision       *int     `json:"quantityPrecision,omitempty"`
}
