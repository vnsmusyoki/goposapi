package sales

type Sale struct {
	ID                    string  `json:"id"`
	BusinessID            string  `json:"businessId"`
	LocationID            string  `json:"locationId"`
	CustomerID            string  `json:"customerId"`
	ReferenceNumber       string  `json:"referenceNumber"`
	SaleDate              string  `json:"saleDate"`
	CustomerName          string  `json:"customerName"`
	CustomerPhone         string  `json:"customerPhone"`
	CustomerEmail         string  `json:"customerEmail"`
	Status                string  `json:"status"`
	Subtotal              float64 `json:"subtotal"`
	TotalDiscount         float64 `json:"totalDiscount"`
	TotalTax              float64 `json:"totalTax"`
	GrandTotal            float64 `json:"grandTotal"`
	ItemsCount            int     `json:"itemsCount"`
	TotalQuantity         float64 `json:"totalQuantity"`
	Notes                 string  `json:"notes"`
	StockAccountingMethod string  `json:"stockAccountingMethod"`
	ReserveOrderItems     bool    `json:"reserveOrderItems"`
	ShippingStatus        string  `json:"shippingStatus"`
	PaidAmount            float64 `json:"paidAmount"`
	BalanceDue            float64 `json:"balanceDue"`
	SaleID                string  `json:"saleId"`
	ConvertedAt           string  `json:"convertedAt"`
	CreatedBy             string  `json:"createdBy"`
	CreatedAt             string  `json:"createdAt"`
	UpdatedAt             string  `json:"updatedAt"`
}

type SalesOrderListItem struct {
	ID              string  `json:"id"`
	BusinessID      string  `json:"businessId"`
	LocationID      string  `json:"locationId"`
	CustomerID      string  `json:"customerId"`
	LocationName    string  `json:"locationName"`
	ReferenceNumber string  `json:"referenceNumber"`
	SaleDate        string  `json:"saleDate"`
	CustomerName    string  `json:"customerName"`
	CustomerPhone   string  `json:"customerPhone"`
	Status          string  `json:"status"`
	ShippingStatus  string  `json:"shippingStatus"`
	ItemsCount      int     `json:"itemsCount"`
	GrandTotal      float64 `json:"grandTotal"`
	PaidAmount      float64 `json:"paidAmount"`
	BalanceDue      float64 `json:"balanceDue"`
	SaleID          string  `json:"saleId"`
	ConvertedAt     string  `json:"convertedAt"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type SalesOrderFilters struct {
	LocationID     string
	CustomerID     string
	Status         string
	ShippingStatus string
	DateFrom       string
	DateTo         string
	SearchQuery    string
}

type UpdateSalesOrderStatusInput struct {
	BusinessID        string
	SalesOrderID      string
	Status            string
	ReserveOrderItems bool
	CreatedBy         string
}

type SaleItem struct {
	ID                   string  `json:"id"`
	SaleID               string  `json:"saleId"`
	BusinessID           string  `json:"businessId"`
	ProductID            string  `json:"productId"`
	ProductName          string  `json:"productName"`
	SKU                  string  `json:"sku"`
	Unit                 string  `json:"unit"`
	Quantity             float64 `json:"quantity"`
	UnitCost             float64 `json:"unitCost"`
	DiscountPercentage   float64 `json:"discountPercentage"`
	DiscountAmount       float64 `json:"discountAmount"`
	TaxRate              float64 `json:"taxRate"`
	TaxAmount            float64 `json:"taxAmount"`
	UnitPrice            float64 `json:"unitPrice"`
	LineTotal            float64 `json:"lineTotal"`
	BatchTrackingEnabled bool    `json:"batchTrackingEnabled"`
	SortOrder            int     `json:"sortOrder"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
}

type SaleItemBatchAllocation struct {
	ID                string  `json:"id"`
	SaleID            string  `json:"saleId"`
	SaleItemID        string  `json:"saleItemId"`
	BusinessID        string  `json:"businessId"`
	InventoryBatchID  string  `json:"inventoryBatchId"`
	AllocatedQuantity float64 `json:"allocatedQuantity"`
	UnitCost          float64 `json:"unitCost"`
	LineTotal         float64 `json:"lineTotal"`
	SortOrder         int     `json:"sortOrder"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

type CreateSaleItemInput struct {
	ProductID            string
	Quantity             float64
	UnitCost             float64
	DiscountPercentage   float64
	DiscountAmount       float64
	TaxRate              float64
	TaxAmount            float64
	UnitPrice            float64
	LineTotal            float64
	BatchTrackingEnabled bool
	SortOrder            int
}

type CreateSaleOrderInput struct {
	BusinessID                string
	LocationID                string
	CustomerID                string
	ReferenceNumber           string
	SaleDate                  string
	CustomerName              string
	CustomerPhone             string
	CustomerEmail             string
	Status                    string
	Notes                     string
	StockAccountingMethod     string
	PreserveSaleOrderRequests bool
	ReserveOrderItems         bool
	CreatedBy                 string
	Items                     []CreateSaleItemInput
	Subtotal                  float64
	TotalDiscount             float64
	TotalTax                  float64
	GrandTotal                float64
	ItemsCount                int
	TotalQuantity             float64
}
