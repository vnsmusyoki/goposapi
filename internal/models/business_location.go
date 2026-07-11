package models

type BusinessLocation struct {
	ID                       string   `json:"id"`
	BusinessID               string   `json:"business_id"`
	LocationID               string   `json:"location_id"`
	LocationName             string   `json:"location_name"`
	Landmark                 string   `json:"landmark"`
	ExactAddress             string   `json:"exact_address"`
	City                     string   `json:"city"`
	ZipCode                  string   `json:"zip_code"`
	State                    string   `json:"state"`
	Country                  string   `json:"country"`
	Latitude                 *float64 `json:"latitude,omitempty"`
	Longitude                *float64 `json:"longitude,omitempty"`
	Mobile                   string   `json:"mobile"`
	AlternateContactNumber   string   `json:"alternate_contact_number"`
	Email                    string   `json:"email"`
	Website                  string   `json:"website"`
	InvoiceScheme            string   `json:"invoice_scheme"`
	PosInvoiceLayout         string   `json:"pos_invoice_layout"`
	SaleInvoiceLayout        string   `json:"sale_invoice_layout"`
	DefaultSellingPriceGroup string   `json:"default_selling_price_group"`
	PaymentMethods           []string `json:"payment_methods"`
	KraPin                   string   `json:"kra_pin"`
	TaxJurisdiction          string   `json:"tax_jurisdiction"`
	IsVatRegistered          bool     `json:"is_vat_registered"`
	VatNumber                string   `json:"vat_number"`
	DefaultTaxType           string   `json:"default_tax_type"`
	PricesIncludeTax         bool     `json:"prices_include_tax"`
	IssueTaxInvoices         bool     `json:"issue_tax_invoices"`
	TaxNote                  string   `json:"tax_note"`
	EtimsEnabled             bool     `json:"etims_enabled"`
	Environment              string   `json:"environment"`
	IntegrationType          string   `json:"integration_type"`
	IsHeadOfficeBranch       bool     `json:"is_head_office_branch"`
	KraBranchID              string   `json:"kra_branch_id"`
	DeviceSerialNumber       string   `json:"device_serial_number"`
	CmcKey                   string   `json:"cmc_key"`
	AutoSubmitInvoices       bool     `json:"auto_submit_invoices"`
	AllowOfflineSales        bool     `json:"allow_offline_sales"`
	RetryFailedInvoices      bool     `json:"retry_failed_invoices"`
	PrintQrCode              bool     `json:"print_qr_code"`
	PrintFiscalDetails       bool     `json:"print_fiscal_details"`
	CreatedAt                string   `json:"created_at"`
	UpdatedAt                string   `json:"updated_at"`
}
