package purchaseorder

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repopurchaseorder "pos/internal/repository/business/purchaseorder"
)

type createPurchaseOrderPayload struct {
	SupplierID       *string                          `json:"supplier_id"`
	ReferenceNumber  *string                          `json:"reference_number"`
	OrderDate        *string                          `json:"order_date"`
	DeliveryDate     *string                          `json:"delivery_date"`
	LocationID       *string                          `json:"location_id"`
	DeliveryAddress  *string                          `json:"delivery_address"`
	DeliveryCharges  *float64                         `json:"delivery_charges"`
	DeliveryDocument *string                          `json:"delivery_document"`
	OrderDiscountAmount *float64                      `json:"order_discount_amount"`
	PaymentTermValue *int                             `json:"payment_term_value"`
	PaymentTermUnit  *string                          `json:"payment_term_unit"`
	Attachment       *string                          `json:"attachment"`
	Notes            *string                          `json:"notes"`
	Status           *string                          `json:"status"`
	DeliveryStatus   *string                          `json:"delivery_status"`
	PaymentStatus    *string                          `json:"payment_status"`
	Subtotal         *float64                         `json:"subtotal"`
	TotalDiscount    *float64                         `json:"total_discount"`
	TotalTax         *float64                         `json:"total_tax"`
	GrandTotal       *float64                         `json:"grand_total"`
	ItemsCount       *int                             `json:"items_count"`
	TotalQuantity    *float64                         `json:"total_quantity"`
	Items            []createPurchaseOrderItemPayload `json:"items"`
	AdditionalExpenses []createPurchaseOrderAdditionalExpensePayload `json:"additionalExpenses"`
}

type createPurchaseOrderItemPayload struct {
	ProductID              *string  `json:"productId"`
	OrderQuantity          *float64 `json:"orderQuantity"`
	UnitCostBeforeDiscount *float64 `json:"unitCostBeforeDiscount"`
	DiscountPercentage     *float64 `json:"discountPercentage"`
	DiscountAmount         *float64 `json:"discountAmount"`
	UnitCostBeforeTax      *float64 `json:"unitCostBeforeTax"`
	ProductTaxRate         *float64 `json:"productTaxRate"`
	TaxAmount              *float64 `json:"taxAmount"`
	NetCost                *float64 `json:"netCost"`
	SellingPrice           *float64 `json:"sellingPrice"`
	LineCost               *float64 `json:"lineCost"`
	ExpiryDate             *string  `json:"expiryDate"`
	LotNumber              *string  `json:"lotNumber"`
}

type createPurchaseOrderAdditionalExpensePayload struct {
	Name      *string  `json:"name"`
	Amount    *float64 `json:"amount"`
	SortOrder *int     `json:"sortOrder"`
}

type purchaseOrderResponse struct {
	repopurchaseorder.PurchaseOrder
	Message string `json:"message,omitempty"`
}

type purchaseOrdersResponse struct {
	PurchaseOrders []repopurchaseorder.PurchaseOrder `json:"purchaseOrders"`
	Message        string                            `json:"message"`
}

func ListPurchaseOrdersRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list purchase orders handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		if businessID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		orders, err := repopurchaseorder.ListPurchaseOrdersRepository(pool, businessID)
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			default:
				log.Printf("list purchase orders handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load purchase orders"})
			}
			return
		}

		c.JSON(http.StatusOK, purchaseOrdersResponse{
			PurchaseOrders: orders,
			Message:        "Purchase orders loaded successfully",
		})
	}
}

func CreatePurchaseOrderRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("create purchase order handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(purchaseOrderFieldErrors(nil)))
			return
		}

		var payload createPurchaseOrderPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create purchase order handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := purchaseOrderFieldErrors(&payload); len(errs) > 0 {
			log.Printf("create purchase order handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create purchase order handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		if businessID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		items := make([]repopurchaseorder.CreatePurchaseOrderItemInput, 0, len(payload.Items))
		for _, item := range payload.Items {
			items = append(items, repopurchaseorder.CreatePurchaseOrderItemInput{
				ProductID:              derefString(item.ProductID),
				OrderQuantity:          derefFloat64(item.OrderQuantity),
				UnitCostBeforeDiscount: derefFloat64(item.UnitCostBeforeDiscount),
				DiscountPercentage:     derefFloat64(item.DiscountPercentage),
				DiscountAmount:         derefFloat64(item.DiscountAmount),
				UnitCostBeforeTax:      derefFloat64(item.UnitCostBeforeTax),
				ProductTaxRate:         derefFloat64(item.ProductTaxRate),
				TaxAmount:              derefFloat64(item.TaxAmount),
				NetCost:                derefFloat64(item.NetCost),
				SellingPrice:           derefFloat64(item.SellingPrice),
				LineCost:               derefFloat64(item.LineCost),
				ExpiryDate:             derefString(item.ExpiryDate),
				LotNumber:              derefString(item.LotNumber),
			})
		}

		additionalExpenses := make([]repopurchaseorder.CreatePurchaseOrderAdditionalExpenseInput, 0, len(payload.AdditionalExpenses))
		for idx, expense := range payload.AdditionalExpenses {
			sortOrder := idx
			if expense.SortOrder != nil {
				sortOrder = *expense.SortOrder
			}
			additionalExpenses = append(additionalExpenses, repopurchaseorder.CreatePurchaseOrderAdditionalExpenseInput{
				Name:      derefString(expense.Name),
				Amount:    derefFloat64(expense.Amount),
				SortOrder: sortOrder,
			})
		}

		createdOrder, err := repopurchaseorder.CreatePurchaseOrderRepository(pool, repopurchaseorder.CreatePurchaseOrderInput{
			BusinessID:         businessID,
			SupplierID:         derefString(payload.SupplierID),
			LocationID:         derefString(payload.LocationID),
			ReferenceNumber:    derefString(payload.ReferenceNumber),
			OrderDate:          derefString(payload.OrderDate),
			DeliveryDate:       derefString(payload.DeliveryDate),
			DeliveryAddress:    derefString(payload.DeliveryAddress),
			DeliveryCharges:    derefFloat64(payload.DeliveryCharges),
			DeliveryDocument:   derefString(payload.DeliveryDocument),
			OrderDiscountAmount: derefFloat64(payload.OrderDiscountAmount),
			PaymentTermValue:   derefInt(payload.PaymentTermValue),
			PaymentTermUnit:    derefString(payload.PaymentTermUnit),
			AttachmentName:     derefString(payload.Attachment),
			Notes:              derefString(payload.Notes),
			Status:             derefString(payload.Status),
			DeliveryStatus:     derefString(payload.DeliveryStatus),
			PaymentStatus:      derefString(payload.PaymentStatus),
			Subtotal:           derefFloat64(payload.Subtotal),
			TotalDiscount:      derefFloat64(payload.TotalDiscount),
			TotalTax:           derefFloat64(payload.TotalTax),
			GrandTotal:         derefFloat64(payload.GrandTotal),
			ItemsCount:         derefInt(payload.ItemsCount),
			TotalQuantity:      derefFloat64(payload.TotalQuantity),
			CreatedBy:          user.ID,
			Items:              items,
			AdditionalExpenses: additionalExpenses,
		})
		if err != nil {
			switch err {
			case repopurchaseorder.ErrInvalidPurchaseOrderInput:
				c.JSON(http.StatusBadRequest, validationFailed(purchaseOrderFieldErrors(&payload)))
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			default:
				log.Printf("create purchase order handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create purchase order"})
			}
			return
		}

		c.JSON(http.StatusCreated, purchaseOrderResponse{
			PurchaseOrder: *createdOrder,
			Message:       "Purchase order created successfully",
		})
	}
}

func DeletePurchaseOrderRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete purchase order handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		purchaseOrderID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || purchaseOrderID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Purchase order id is required.",
			}))
			return
		}

		if err := repopurchaseorder.DeletePurchaseOrderRepository(pool, businessID, purchaseOrderID); err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case repopurchaseorder.ErrPurchaseOrderNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase order not found"})
			default:
				log.Printf("delete purchase order handler: repository failed business_id=%s order_id=%s err=%v", businessID, purchaseOrderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete purchase order"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":      purchaseOrderID,
			"message": "Purchase order deleted successfully",
		})
	}
}

func purchaseOrderFieldErrors(payload *createPurchaseOrderPayload) map[string]string {
	if payload == nil {
		return map[string]string{
			"supplier_id":       "Supplier is required.",
			"location_id":       "Business location is required.",
			"order_date":        "Order date is required.",
			"items":             "At least one item is required.",
			"payment_term_unit": "Payment term unit is required.",
		}
	}

	errs := make(map[string]string)
	if payload.SupplierID == nil || strings.TrimSpace(*payload.SupplierID) == "" {
		errs["supplier_id"] = "Supplier is required."
	}
	if payload.LocationID == nil || strings.TrimSpace(*payload.LocationID) == "" {
		errs["location_id"] = "Business location is required."
	}
	if payload.OrderDate == nil || strings.TrimSpace(*payload.OrderDate) == "" {
		errs["order_date"] = "Order date is required."
	}
	if payload.PaymentTermUnit == nil || strings.TrimSpace(*payload.PaymentTermUnit) == "" {
		errs["payment_term_unit"] = "Payment term unit is required."
	}
	if len(payload.Items) == 0 {
		errs["items"] = "At least one item is required."
	}
	for _, item := range payload.Items {
		keyPrefix := "items"
		if item.ProductID == nil || strings.TrimSpace(*item.ProductID) == "" {
			errs[keyPrefix] = "Each item must reference a product."
		}
		if item.OrderQuantity == nil || *item.OrderQuantity <= 0 {
			errs[keyPrefix] = "Item quantity must be greater than 0."
		}
		if item.UnitCostBeforeDiscount != nil && *item.UnitCostBeforeDiscount < 0 {
			errs[keyPrefix] = "Item costs cannot be negative."
		}
	}

	return errs
}

func validationFailed(errorsMap map[string]string) gin.H {
	return gin.H{
		"message": "Validation failed",
		"errors":  errorsMap,
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func derefFloat64(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Code), "business") {
			return true
		}
	}
	return false
}
