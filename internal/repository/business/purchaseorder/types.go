package purchaseorder

type PurchaseOrderCreatedBy struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PurchaseOrder struct {
	ID                    string                   `json:"id"`
	BusinessID            string                   `json:"businessId"`
	SupplierID            string                   `json:"supplierId"`
	SupplierName          string                   `json:"supplierName"`
	LocationID            string                   `json:"locationId"`
	LocationName          string                   `json:"locationName"`
	ReferenceNumber       string                   `json:"referenceNumber"`
	OrderDate             string                   `json:"orderDate"`
	DeliveryDate          string                   `json:"deliveryDate"`
	PaymentTermValue      int                      `json:"paymentTermValue"`
	PaymentTermUnit       string                   `json:"paymentTermUnit"`
	AttachmentName        string                   `json:"attachmentName"`
	AttachmentURL         string                   `json:"attachmentUrl"`
	DeliveryAddress       string                   `json:"deliveryAddress"`
	DeliveryCharges       float64                  `json:"deliveryCharges"`
	DeliveryDocumentName  string                   `json:"deliveryDocumentName"`
	DeliveryDocumentURL   string                   `json:"deliveryDocumentUrl"`
	OrderDiscountAmount   float64                  `json:"orderDiscountAmount"`
	Notes                 string                   `json:"notes"`
	Status                string                   `json:"status"`
	DeliveryStatus        string                   `json:"deliveryStatus"`
	PaymentStatus         string                   `json:"paymentStatus"`
	Subtotal              float64                  `json:"subtotal"`
	TotalDiscount         float64                  `json:"totalDiscount"`
	TotalTax              float64                  `json:"totalTax"`
	GrandTotal            float64                  `json:"totalAmount"`
	ItemsCount            int                      `json:"itemsCount"`
	TotalQuantity         float64                  `json:"totalQuantity"`
	CreatedBy             *PurchaseOrderCreatedBy  `json:"createdBy,omitempty"`
	AdditionalExpenses    []PurchaseOrderExtraCost  `json:"additionalExpenses,omitempty"`
	CreatedAt             string                   `json:"createdAt"`
	UpdatedAt             string                   `json:"updatedAt"`
}

type PurchaseOrderExtraCost struct {
	Name      string  `json:"name"`
	Amount    float64 `json:"amount"`
	SortOrder int     `json:"sortOrder"`
}

type PurchaseOrderItem struct {
	ID                     string   `json:"id"`
	PurchaseOrderID        *string  `json:"purchaseOrderId,omitempty"`
	BusinessID             string   `json:"businessId"`
	ProductID              string   `json:"productId"`
	ProductName            string   `json:"productName"`
	SKU                    string   `json:"sku"`
	Unit                   string   `json:"unit"`
	OrderQuantity          float64  `json:"orderQuantity"`
	UnitCostBeforeDiscount float64  `json:"unitCostBeforeDiscount"`
	DiscountPercentage     float64  `json:"discountPercentage"`
	DiscountAmount         float64  `json:"discountAmount"`
	UnitCostBeforeTax      float64  `json:"unitCostBeforeTax"`
	ProductTaxRate         float64  `json:"productTaxRate"`
	TaxAmount              float64  `json:"taxAmount"`
	NetCost                float64  `json:"netCost"`
	SellingPrice           float64  `json:"sellingPrice"`
	LineCost               float64  `json:"lineCost"`
	ExpiryDate             string   `json:"expiryDate"`
	LotNumber              string   `json:"lotNumber"`
	ReceivedQuantity       *float64 `json:"receivedQuantity,omitempty"`
	ItemsReceived          float64  `json:"itemsReceived"`
	ReceivedStatus         string   `json:"receivedStatus"`
	SortOrder              int      `json:"sortOrder"`
	CreatedAt              string   `json:"createdAt"`
	UpdatedAt              string   `json:"updatedAt"`
}

type CreatePurchaseOrderItemInput struct {
	PurchaseOrderID        string
	ProductID              string
	OrderQuantity          float64
	UnitCostBeforeDiscount float64
	DiscountPercentage     float64
	DiscountAmount         float64
	UnitCostBeforeTax      float64
	ProductTaxRate         float64
	TaxAmount              float64
	NetCost                float64
	SellingPrice           float64
	LineCost               float64
	ExpiryDate             string
	LotNumber              string
}

type CreatePurchaseOrderAdditionalExpenseInput struct {
	Name      string
	Amount    float64
	SortOrder int
}

type CreatePurchaseOrderInput struct {
	BusinessID          string
	SupplierID          string
	LocationID          string
	ReferenceNumber     string
	OrderDate           string
	DeliveryDate        string
	PaymentTermValue    int
	PaymentTermUnit     string
	AttachmentName      string
	AttachmentURL       string
	DeliveryAddress     string
	DeliveryCharges     float64
	DeliveryDocument    string
	OrderDiscountAmount float64
	Notes               string
	Status              string
	DeliveryStatus      string
	PaymentStatus       string
	Subtotal            float64
	TotalDiscount       float64
	TotalTax            float64
	GrandTotal          float64
	ItemsCount          int
	TotalQuantity       float64
	CreatedBy           string
	Items               []CreatePurchaseOrderItemInput
	AdditionalExpenses  []CreatePurchaseOrderAdditionalExpenseInput
}
