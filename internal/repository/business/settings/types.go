package settings

type UpdateBusinessSettingsInput struct {
	BusinessID              string
	Name                    string
	StartDate               string
	DefaultProfitPercentage float64
	Currency                string
	CurrencySymbolPlacement string
	Timezone                string
	LogoURL                 string
	FinancialYearStartMonth string
	StockAccountingMethod   string
	TransactionEditDays     int
	DateFormat              string
	TimeFormat              string
	CurrencyPrecision       int
	QuantityPrecision       int
}
