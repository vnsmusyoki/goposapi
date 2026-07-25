package stocktransfer

type StockTransfer struct {
	ID               string              `json:"id"`
	BusinessID       string              `json:"businessId"`
	ReferenceNumber  string              `json:"referenceNumber"`
	TransferDate     string              `json:"transferDate"`
	Status           string              `json:"status"`
	LocationFromID   string              `json:"locationFromId"`
	LocationToID     string              `json:"locationToId"`
	LocationFromName string              `json:"locationFromName"`
	LocationToName   string              `json:"locationToName"`
	Currency         string              `json:"currency"`
	ShippingCharges  float64             `json:"shippingCharges"`
	Subtotal         float64             `json:"subtotal"`
	TotalAmount      float64             `json:"totalAmount"`
	ItemsCount       int                 `json:"itemsCount"`
	TotalQuantity    float64             `json:"totalQuantity"`
	Notes            string              `json:"notes"`
	Type             string              `json:"type"`
	Items            []StockTransferItem `json:"items"`
	CreatedBy        string              `json:"createdBy"`
	ApprovedBy       string              `json:"approvedBy"`
	ApprovedAt       string              `json:"approvedAt"`
	CompletedAt      string              `json:"completedAt"`
	CreatedAt        string              `json:"createdAt"`
	UpdatedAt        string              `json:"updatedAt"`
}

type StockTransferListItem struct {
	ID               string              `json:"id"`
	ReferenceNumber  string              `json:"referenceNumber"`
	TransferDate     string              `json:"transferDate"`
	Status           string              `json:"status"`
	LocationFromID   string              `json:"locationFromId"`
	LocationToID     string              `json:"locationToId"`
	LocationFromName string              `json:"locationFromName"`
	LocationToName   string              `json:"locationToName"`
	Currency         string              `json:"currency"`
	ShippingCharges  float64             `json:"shippingCharges"`
	Subtotal         float64             `json:"subtotal"`
	TotalAmount      float64             `json:"totalAmount"`
	ItemsCount       int                 `json:"itemsCount"`
	TotalQuantity    float64             `json:"totalQuantity"`
	Notes            string              `json:"notes"`
	Type             string              `json:"type"`
	Items            []StockTransferItem `json:"items,omitempty"`
	CreatedAt        string              `json:"createdAt"`
	UpdatedAt        string              `json:"updatedAt"`
}

type StockTransferItem struct {
	ID              string                    `json:"id"`
	StockTransferID string                    `json:"stockTransferId"`
	BusinessID      string                    `json:"businessId"`
	ProductID       string                    `json:"productId"`
	ProductName     string                    `json:"productName"`
	SKU             string                    `json:"sku"`
	Unit            string                    `json:"unit"`
	Quantity        float64                   `json:"quantity"`
	UnitPrice       float64                   `json:"unitPrice"`
	Subtotal        float64                   `json:"subtotal"`
	SortOrder       int                       `json:"sortOrder"`
	Allocations     []StockTransferAllocation `json:"allocations,omitempty"`
	CreatedAt       string                    `json:"createdAt"`
	UpdatedAt       string                    `json:"updatedAt"`
}

type StockTransferAllocation struct {
	ID                  string  `json:"id"`
	StockTransferID     string  `json:"stockTransferId"`
	StockTransferItemID string  `json:"stockTransferItemId"`
	BusinessID          string  `json:"businessId"`
	InventoryBatchID    string  `json:"inventoryBatchId"`
	AllocatedQuantity   float64 `json:"allocatedQuantity"`
	UnitCost            float64 `json:"unitCost"`
	LineTotal           float64 `json:"lineTotal"`
	SortOrder           int     `json:"sortOrder"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
}

type CreateStockTransferInput struct {
	BusinessID            string
	ReferenceNumber       string
	TransferDate          string
	Status                string
	TransferType          string
	LocationFromID        string
	LocationToID          string
	LocationFromName      string
	LocationToName        string
	Currency              string
	ShippingCharges       float64
	Notes                 string
	CreatedBy             string
	StockAccountingMethod string
	Items                 []CreateStockTransferItemInput
}

type CreateStockTransferItemInput struct {
	ProductID string
	Quantity  float64
	UnitPrice float64
	SortOrder int
}
