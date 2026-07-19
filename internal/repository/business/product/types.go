package product

import "database/sql"

type CreateProductImageInput struct {
	Name      string
	URL       string
	IsPrimary bool
}

type CreateProductComboItemInput struct {
	ProductID   string
	ProductName string
	SKU         string
	Unit        string
	Quantity    float64
	PriceEach   float64
	Subtotal    float64
}

type CreateProductVariantInput struct {
	Name               string
	SKU                string
	Barcode            string
	Cost               float64
	Selling            float64
	Stock              float64
	ShowOptionalFields bool
	Weight             string
	Length             string
	Width              string
	Height             string
	ImageName          string
	ImageURL           string
	ReorderLevel       *int
	ExpiryDate         string
	SupplierCode       string
}

type CreateProductPriceInput struct {
	PriceType     string
	MinQuantity   float64
	Price         float64
	LocationID    string
	CustomerGroup string
	StartsAt      string
	EndsAt        string
	Active        bool
	Priority      int
}

type CreateProductInput struct {
	BusinessID              string
	Name                    string
	SKU                     string
	Barcode                 string
	ProductType             string
	UnitID                  string
	SubUnitIDs              []string
	BrandID                 string
	CategoryID              string
	SubCategoryID           string
	LocationIDs             []string
	AllLocations            bool
	ManageStock             bool
	AlertQuantity           *int
	IsForSelling            bool
	TaxType                 string
	TaxRate                 float64
	DefaultPurchasePrice    *float64
	PurchasePriceExclusive  *float64
	PurchasePriceInclusive  *float64
	ProfitMargin            *float64
	DefaultSellingPrice     *float64
	Description             string
	HasWarranty             bool
	WarrantyDuration        string
	WarrantyPeriod          string
	WarrantyCoverage        string
	BrochureName            string
	BrochureURL             string
	CurrencyCode            string
	CurrencySymbolPlacement string
	CurrencyPrecision       int
	CreatedBy               string
	Images                  []CreateProductImageInput
	ComboItems              []CreateProductComboItemInput
	Variants                []CreateProductVariantInput
	ProductPrices           []CreateProductPriceInput
}

type ProductImageItem struct {
	ID        string
	Name      string
	URL       string
	IsPrimary bool
}

type ProductComboItemItem struct {
	ID          string
	ProductID   string
	ProductName string
	SKU         string
	Unit        string
	Quantity    float64
	PriceEach   float64
	Subtotal    float64
}

type ProductVariantItem struct {
	ID                 string
	Name               string
	SKU                string
	Barcode            string
	Cost               float64
	Selling            float64
	Stock              float64
	ShowOptionalFields bool
	Weight             string
	Length             string
	Width              string
	Height             string
	ImageName          string
	ImageURL           string
	ReorderLevel       *int
	ExpiryDate         string
	SupplierCode       string
}

type ProductPriceItem struct {
	ID            string
	PriceType     string
	MinQuantity   float64
	Price         float64
	LocationID    string
	CustomerGroup string
	StartsAt      string
	EndsAt        string
	Active        bool
	Priority      int
}

type ProductDetail struct {
	ID                      string
	Name                    string
	SKU                     *string
	ImageURL                string
	Barcode                 string
	ProductType             string
	UnitID                  string
	UnitName                string
	SubUnitIDs              []string
	SubUnits                []ProductSubUnitItem
	BrandID                 string
	BrandName               string
	CategoryID              string
	CategoryName            string
	SubCategoryID           string
	SubCategoryName         string
	LocationIDs             []string
	LocationNames           []string
	AllLocations            bool
	ManageStock             bool
	AlertQuantity           int
	IsForSelling            bool
	TaxType                 string
	TaxRate                 float64
	DefaultPurchasePrice    float64
	PurchasePriceExclusive  float64
	ProfitAmount            float64
	PurchasePriceInclusive  float64
	ProfitMargin            float64
	DefaultSellingPrice     float64
	Description             string
	HasWarranty             bool
	WarrantyDuration        string
	WarrantyPeriod          string
	WarrantyCoverage        string
	BrochureName            string
	BrochureURL             string
	CurrencyCode            string
	CurrencySymbolPlacement string
	CurrencyPrecision       int
	Status                  string
	CurrentStock            int
	CurrentStockValue       float64
	TotalUnitsSold          int
	TotalUnitsTransferred   int
	TotalUnitsAdjusted      int
	CreatedAt               string
	UpdatedAt               string
	Images                  []ProductImageItem
	ComboItems              []ProductComboItemItem
	Variants                []ProductVariantItem
	ProductPrices           []ProductPriceItem
}

type ProductSubUnitItem struct {
	ID   string
	Name string
}

type ProductPriceHistoryItem struct {
	ID             string
	ProductID      string
	ProductPriceID sql.NullString
	Action         string
	PriceType      string
	MinQuantity    float64
	OldPrice       sql.NullFloat64
	NewPrice       float64
	LocationID     string
	CustomerGroup  string
	StartsAt       string
	EndsAt         string
	Active         bool
	Priority       int
	Reason         sql.NullString
	ChangedByID    string
	ChangedByName  string
	CreatedAt      string
}

type ListProductsFilters struct {
	Search            string
	ProductType       string
	CategoryID        string
	BrandID           string
	UnitID            string
	LocationID        string
	TaxType           string
	ShowNotForSelling bool
}

type ProductImportBatch struct {
	ID         string `json:"id"`
	BusinessID string `json:"businessId"`
	FileName   string `json:"fileName"`
	Status     string `json:"status"`
	CreatedBy  string `json:"createdBy"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type ProductImportBatchRow struct {
	ID                string            `json:"id"`
	BatchID           string            `json:"batchId"`
	RowNumber         int               `json:"rowNumber"`
	RowData           map[string]string `json:"rowData"`
	ValidationErrors  []string          `json:"validationErrors"`
	Status            string            `json:"status"`
	ImportedProductID string            `json:"importedProductId"`
	CreatedAt         string            `json:"createdAt"`
	UpdatedAt         string            `json:"updatedAt"`
}
