package models

import "database/sql"

type Product struct {
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	SKU         *string `json:"sku" db:"sku"`
	ProductType string  `json:"productType" db:"product_type"`
	Message     string  `json:"message,omitempty"`
}

type ProductSearchItem struct {
	ID           string  `json:"id" db:"id"`
	Name         string  `json:"name" db:"name"`
	SKU          *string `json:"sku" db:"sku"`
	UnitName     string  `json:"unitName" db:"unit_name"`
	SellingPrice float64 `json:"sellingPrice" db:"selling_price"`
	ProductType  string  `json:"productType" db:"product_type"`
}

type ProductListItem struct {
	ID                    string   `json:"id" db:"id"`
	Name                  string   `json:"name" db:"name"`
	SKU                   *string  `json:"sku" db:"sku"`
	ImageURL              string   `json:"imageUrl" db:"image_url"`
	Barcode               string   `json:"barcode" db:"barcode"`
	ProductType           string   `json:"productType" db:"product_type"`
	UnitID                string   `json:"unitId" db:"unit_id"`
	UnitName              string   `json:"unitName" db:"unit_name"`
	BrandID               string   `json:"brandId" db:"brand_id"`
	BrandName             string   `json:"brandName" db:"brand_name"`
	CategoryID            string   `json:"categoryId" db:"category_id"`
	CategoryName          string   `json:"categoryName" db:"category_name"`
	SubCategoryID         string   `json:"subCategoryId" db:"sub_category_id"`
	SubCategoryName       string   `json:"subCategoryName" db:"sub_category_name"`
	LocationIDs           []string `json:"locationIds" db:"location_ids"`
	LocationNames         []string `json:"locationNames" db:"location_names"`
	ManageStock           bool     `json:"manageStock" db:"manage_stock"`
	AlertQuantity         int      `json:"alertQuantity" db:"alert_quantity"`
	IsForSelling          bool     `json:"isForSelling" db:"is_for_selling"`
	TaxType               string   `json:"taxType" db:"tax_type"`
	TaxRate               float64  `json:"taxRate" db:"tax_rate"`
	DefaultPurchasePrice  float64  `json:"defaultPurchasePrice" db:"default_purchase_price"`
	DefaultSellingPrice   float64  `json:"defaultSellingPrice" db:"default_selling_price"`
	ProfitMargin          float64  `json:"profitMargin" db:"profit_margin"`
	CurrentStock          int      `json:"currentStock" db:"current_stock"`
	CurrentStockValue     float64  `json:"currentStockValue" db:"current_stock_value"`
	TotalUnitsSold        int      `json:"totalUnitsSold" db:"total_units_sold"`
	TotalUnitsTransferred int      `json:"totalUnitsTransferred" db:"total_units_transferred"`
	TotalUnitsAdjusted    int      `json:"totalUnitsAdjusted" db:"total_units_adjusted"`
	CreatedAt             string   `json:"createdAt" db:"created_at"`
	UpdatedAt             string   `json:"updatedAt" db:"updated_at"`
	Status                string   `json:"status" db:"status"`
}

type ProductImage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	IsPrimary bool   `json:"isPrimary"`
}

type ProductComboItem struct {
	ID          string  `json:"id"`
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	SKU         string  `json:"sku"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	PriceEach   float64 `json:"priceEach"`
	Subtotal    float64 `json:"subtotal"`
}

type ProductVariant struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	SKU                string  `json:"sku"`
	Barcode            string  `json:"barcode"`
	Cost               float64 `json:"cost"`
	Selling            float64 `json:"selling"`
	Stock              float64 `json:"stock"`
	ShowOptionalFields bool    `json:"showOptionalFields"`
	Weight             string  `json:"weight"`
	Length             string  `json:"length"`
	Width              string  `json:"width"`
	Height             string  `json:"height"`
	ImageName          string  `json:"imageName"`
	ImageURL           string  `json:"imageUrl"`
	ReorderLevel       *int    `json:"reorderLevel"`
	ExpiryDate         string  `json:"expiryDate"`
	SupplierCode       string  `json:"supplierCode"`
}

type ProductDetail struct {
	ProductListItem
	UnitID                  string             `json:"unitId"`
	SubUnitIDs              []string           `json:"subUnitIds"`
	BrandID                 string             `json:"brandId"`
	CategoryID              string             `json:"categoryId"`
	SubCategoryID           string             `json:"subCategoryId"`
	AllLocations            bool               `json:"allLocations"`
	ManageStock             bool               `json:"manageStock"`
	AlertQuantity           int                `json:"alertQuantity"`
	IsForSelling            bool               `json:"isForSelling"`
	TaxType                 string             `json:"taxType"`
	TaxRate                 float64            `json:"taxRate"`
	DefaultPurchasePrice    float64            `json:"defaultPurchasePrice"`
	PurchasePriceExclusive  float64            `json:"purchasePriceExclusive"`
	PurchasePriceInclusive  float64            `json:"purchasePriceInclusive"`
	ProfitMargin            float64            `json:"profitMargin"`
	DefaultSellingPrice     float64            `json:"defaultSellingPrice"`
	Description             string             `json:"description"`
	HasWarranty             bool               `json:"hasWarranty"`
	WarrantyDuration        string             `json:"warrantyDuration"`
	WarrantyPeriod          string             `json:"warrantyPeriod"`
	WarrantyCoverage        string             `json:"warrantyCoverage"`
	BrochureName            string             `json:"brochureName"`
	BrochureURL             string             `json:"brochureUrl"`
	CurrencyCode            string             `json:"currencyCode"`
	CurrencySymbolPlacement string             `json:"currencySymbolPlacement"`
	CurrencyPrecision       int                `json:"currencyPrecision"`
	Images                  []ProductImage     `json:"images"`
	ComboItems              []ProductComboItem `json:"comboItems"`
	Variants                []ProductVariant   `json:"variants"`
}

func StringPtrFromNullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}
