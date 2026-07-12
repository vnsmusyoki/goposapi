package settings

type BusinessInvoiceSettings struct {
	BusinessID                    string
	Name                          string
	Code                          string
	ProductLabel                  string
	QuantityLabel                 string
	UnitPriceLabel                string
	SubTotalLabel                 string
	CategoryHsnCodeLabel          string
	TotalQuantityLabel            string
	ItemDiscountLabel             string
	DiscountedUnitPriceLabel      string
	SubheadingLine1               string
	SubheadingLine2               string
	SubheadingLine3               string
	SubheadingLine4               string
	SubheadingLine5               string
	Design                        string
	PaperSize                     string
	IsDefault                     bool
	ShowLogo                      bool
	ShowBusinessDetails           bool
	ShowCustomerDetails           bool
	ShowItemsSku                  bool
	ShowBrand                     bool
	ShowSaleDescription           bool
	ShowQrCode                    bool
	ShowProductExpiry             bool
	ShowLotNumber                 bool
	ShowProductImage              bool
	ShowWarrantyName              bool
	ShowWarrantyExpiryDate        bool
	ShowWarrantyDescription       bool
	ShowTaxBreakdown              bool
	ShowDiscounts                 bool
	ShowBarcode                   bool
	BarcodeTotalDueLabel          string
	ShowTotalBalanceDue           bool
	BarcodeChangeReturnLabel      string
	HideAllPrices                 bool
	ShowTotalInWords              bool
	BarcodeWordFormat             string
	BarcodeTaxSummaryLabel        string
	HeaderAlignment               string
	LogoURL                       string
	QrShowLabels                  bool
	QrShowBusinessName            bool
	QrShowBusinessLocationAddress bool
	QrShowInvoiceNo               bool
	QrShowSubtotal                bool
	QrShowTotalAmountWithTax      bool
	QrShowTotalTax                bool
	QrShowCustomerName            bool
	QrShowInvoiceUrl              bool
	QrShowInvoiceDateTime         bool
	QrShowBusinessTax1            bool
	InvoiceNote                   string
}

type CreateBusinessInvoiceSettingsInput struct {
	BusinessInvoiceSettings
}

type UpdateBusinessInvoiceSettingsInput struct {
	ID string
	BusinessInvoiceSettings
}
