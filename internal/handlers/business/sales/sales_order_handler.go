package sales

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	reposales "pos/internal/repository/business/sales"
	reposettings "pos/internal/repository/business/settings"
)

type createSaleOrderPayload struct {
	CustomerID        *string                      `json:"customer_id"`
	CustomerName      *string                      `json:"customer_name"`
	CustomerPhone     *string                      `json:"customer_phone"`
	CustomerEmail     *string                      `json:"customer_email"`
	ReferenceNumber   *string                      `json:"reference_number"`
	SaleDate          *string                      `json:"sale_date"`
	LocationID        *string                      `json:"location_id"`
	Notes             *string                      `json:"notes"`
	Status            *string                      `json:"status"`
	Subtotal          *float64                     `json:"subtotal"`
	TotalDiscount     *float64                     `json:"total_discount"`
	TotalTax          *float64                     `json:"total_tax"`
	GrandTotal        *float64                     `json:"grand_total"`
	ReserveOrderItems *bool                        `json:"reserve_order_items"`
	ItemsCount        *int                         `json:"items_count"`
	TotalQuantity     *float64                     `json:"total_quantity"`
	Items             []createSaleOrderItemPayload `json:"items"`
}

type createSaleOrderItemPayload struct {
	ProductID            *string  `json:"product_id"`
	Quantity             *float64 `json:"quantity"`
	UnitCost             *float64 `json:"unit_cost"`
	DiscountPercentage   *float64 `json:"discount_percentage"`
	DiscountAmount       *float64 `json:"discount_amount"`
	TaxRate              *float64 `json:"tax_rate"`
	TaxAmount            *float64 `json:"tax_amount"`
	UnitPrice            *float64 `json:"unit_price"`
	LineTotal            *float64 `json:"line_total"`
	BatchTrackingEnabled *bool    `json:"batch_tracking_enabled"`
	SortOrder            *int     `json:"sort_order"`
}

type saleOrderResponse struct {
	Sale    reposales.Sale `json:"sale"`
	Message string         `json:"message,omitempty"`
}

type salesOrdersResponse struct {
	SalesOrders []reposales.SalesOrderListItem `json:"salesOrders"`
	Message     string                         `json:"message,omitempty"`
}

type salesOrderDetailResponse struct {
	SalesOrder reposales.SalesOrderDetail `json:"salesOrder"`
	Message    string                     `json:"message,omitempty"`
}

type deleteSaleOrderResponse struct {
	Message string `json:"message,omitempty"`
}

type updateSalesOrderStatusPayload struct {
	Status            *string `json:"status"`
	ReserveOrderItems *bool   `json:"reserve_order_items"`
}

func CreateSaleOrderRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create sale order handler: auth lookup failed err=%v", err)
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

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Unable to read request body.",
			}))
			return
		}

		var payload createSaleOrderPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create sale order handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		errs := saleOrderFieldErrors(&payload)
		if len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		settings, err := reposettings.GetBusinessSettingsRepository(pool, businessID)
		if err != nil {
			log.Printf("create sale order handler: load settings failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load business settings"})
			return
		}

		items := make([]reposales.CreateSaleItemInput, 0, len(payload.Items))
		for idx, item := range payload.Items {
			items = append(items, reposales.CreateSaleItemInput{
				ProductID:            strings.TrimSpace(derefString(item.ProductID)),
				Quantity:             derefFloat(item.Quantity),
				UnitCost:             derefFloat(item.UnitCost),
				DiscountPercentage:   derefFloat(item.DiscountPercentage),
				DiscountAmount:       derefFloat(item.DiscountAmount),
				TaxRate:              derefFloat(item.TaxRate),
				TaxAmount:            derefFloat(item.TaxAmount),
				UnitPrice:            derefFloat(item.UnitPrice),
				LineTotal:            derefFloat(item.LineTotal),
				BatchTrackingEnabled: derefBool(item.BatchTrackingEnabled),
				SortOrder:            derefInt(item.SortOrder, idx),
			})
		}

		sale, err := reposales.CreateSaleOrderRepository(pool, reposales.CreateSaleOrderInput{
			BusinessID:                businessID,
			LocationID:                derefString(payload.LocationID),
			CustomerID:                derefString(payload.CustomerID),
			ReferenceNumber:           derefString(payload.ReferenceNumber),
			SaleDate:                  derefString(payload.SaleDate),
			CustomerName:              derefString(payload.CustomerName),
			CustomerPhone:             derefString(payload.CustomerPhone),
			CustomerEmail:             derefString(payload.CustomerEmail),
			Status:                    derefString(payload.Status),
			Notes:                     derefString(payload.Notes),
			StockAccountingMethod:     settings.StockAccountingMethod,
			PreserveSaleOrderRequests: settings.PreserveSaleOrderRequests,
			ReserveOrderItems:         reserveOrderItemsValue(payload.ReserveOrderItems, settings.PreserveSaleOrderRequests),
			CreatedBy:                 user.ID,
			CreatedByName:             user.FullName,
			Items:                     items,
			Subtotal:                  derefFloat(payload.Subtotal),
			TotalDiscount:             derefFloat(payload.TotalDiscount),
			TotalTax:                  derefFloat(payload.TotalTax),
			GrandTotal:                derefFloat(payload.GrandTotal),
			ItemsCount:                derefInt(payload.ItemsCount, len(items)),
			TotalQuantity:             derefFloat(payload.TotalQuantity),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposales.ErrInvalidSaleInput):
				c.JSON(http.StatusBadRequest, validationFailed(saleOrderFieldErrors(&payload)))
			default:
				log.Printf("create sale order handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create sale order"})
			}
			return
		}

		c.JSON(http.StatusCreated, saleOrderResponse{
			Sale:    *sale,
			Message: "Sale order created successfully",
		})
	}
}

func UpdateSaleOrderRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update sale order handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		salesOrderID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || salesOrderID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Sales order id is required.",
			}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Unable to read request body.",
			}))
			return
		}

		var payload createSaleOrderPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update sale order handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		errs := saleOrderFieldErrors(&payload)
		if len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		items := make([]reposales.CreateSaleItemInput, 0, len(payload.Items))
		for idx, item := range payload.Items {
			items = append(items, reposales.CreateSaleItemInput{
				ProductID:            strings.TrimSpace(derefString(item.ProductID)),
				Quantity:             derefFloat(item.Quantity),
				UnitCost:             derefFloat(item.UnitCost),
				DiscountPercentage:   derefFloat(item.DiscountPercentage),
				DiscountAmount:       derefFloat(item.DiscountAmount),
				TaxRate:              derefFloat(item.TaxRate),
				TaxAmount:            derefFloat(item.TaxAmount),
				UnitPrice:            derefFloat(item.UnitPrice),
				LineTotal:            derefFloat(item.LineTotal),
				BatchTrackingEnabled: derefBool(item.BatchTrackingEnabled),
				SortOrder:            derefInt(item.SortOrder, idx),
			})
		}

		sale, err := reposales.UpdateSaleOrderRepository(pool, reposales.UpdateSaleOrderInput{
			BusinessID:        businessID,
			SalesOrderID:      salesOrderID,
			CustomerID:        derefString(payload.CustomerID),
			LocationID:        derefString(payload.LocationID),
			SaleDate:          derefString(payload.SaleDate),
			CustomerName:      derefString(payload.CustomerName),
			CustomerPhone:     derefString(payload.CustomerPhone),
			CustomerEmail:     derefString(payload.CustomerEmail),
			Notes:             derefString(payload.Notes),
			Status:            derefString(payload.Status),
			ReserveOrderItems: derefBool(payload.ReserveOrderItems),
			UpdatedBy:         user.ID,
			UpdatedByName:     user.FullName,
			Items:             items,
			Subtotal:          derefFloat(payload.Subtotal),
			TotalDiscount:     derefFloat(payload.TotalDiscount),
			TotalTax:          derefFloat(payload.TotalTax),
			GrandTotal:        derefFloat(payload.GrandTotal),
			ItemsCount:        derefInt(payload.ItemsCount, len(items)),
			TotalQuantity:     derefFloat(payload.TotalQuantity),
		})
		if err != nil {
			switch err {
			case reposales.ErrInvalidSaleInput:
				c.JSON(http.StatusBadRequest, validationFailed(saleOrderFieldErrors(&payload)))
			case reposales.ErrSaleNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Sales order not found"})
			case reposales.ErrSalesOrderCannotUpdate:
				c.JSON(http.StatusConflict, gin.H{"message": "This sales order cannot be updated in its current status"})
			default:
				log.Printf("update sale order handler: repository failed business_id=%s sales_order_id=%s err=%v", businessID, salesOrderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update sale order"})
			}
			return
		}

		c.JSON(http.StatusOK, saleOrderResponse{
			Sale:    *sale,
			Message: "Sale order updated successfully",
		})
	}
}

func ListSalesOrdersRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list sales orders handler: auth lookup failed err=%v", err)
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

		filters := reposales.SalesOrderFilters{
			LocationID:     strings.TrimSpace(c.Query("location_id")),
			CustomerID:     strings.TrimSpace(c.Query("customer_id")),
			Status:         strings.TrimSpace(c.Query("status")),
			ShippingStatus: strings.TrimSpace(c.Query("shipping_status")),
			DateFrom:       strings.TrimSpace(c.Query("date_from")),
			DateTo:         strings.TrimSpace(c.Query("date_to")),
			SearchQuery:    strings.TrimSpace(c.Query("search_query")),
		}

		orders, err := reposales.ListSalesOrdersRepository(pool, businessID, filters)
		if err != nil {
			switch err {
			case reposales.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			default:
				log.Printf("list sales orders handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load sales orders"})
			}
			return
		}

		c.JSON(http.StatusOK, salesOrdersResponse{
			SalesOrders: orders,
			Message:     "Sales orders loaded successfully",
		})
	}
}

func GetSalesOrderRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get sales order handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		salesOrderID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || salesOrderID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Sales order id is required.",
			}))
			return
		}

		order, err := reposales.GetSalesOrderDetailRepository(pool, businessID, salesOrderID)
		if err != nil {
			switch err {
			case reposales.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case reposales.ErrSaleNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Sales order not found"})
			default:
				log.Printf("get sales order handler: repository failed business_id=%s sales_order_id=%s err=%v", businessID, salesOrderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load sales order"})
			}
			return
		}

		c.JSON(http.StatusOK, salesOrderDetailResponse{
			SalesOrder: formatSalesOrderDetailDisplayTime(*order),
			Message:    "Sales order loaded successfully",
		})
	}
}

func UpdateSalesOrderStatusRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update sales order status handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		salesOrderID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || salesOrderID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Sales order id is required.",
			}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Unable to read request body.",
			}))
			return
		}

		var payload updateSalesOrderStatusPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update sales order status handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		status := strings.TrimSpace(derefString(payload.Status))
		if !allowedSaleStatuses[strings.ToLower(status)] {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"status": "A valid status is required.",
			}))
			return
		}

		updated, err := reposales.UpdateSalesOrderStatusRepository(pool, reposales.UpdateSalesOrderStatusInput{
			BusinessID:        businessID,
			SalesOrderID:      salesOrderID,
			Status:            status,
			ReserveOrderItems: derefBool(payload.ReserveOrderItems),
			CreatedBy:         user.ID,
			CreatedByName:     user.FullName,
		})
		if err != nil {
			switch err {
			case reposales.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case reposales.ErrSaleNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Sales order not found"})
			default:
				log.Printf("update sales order status handler: repository failed business_id=%s sales_order_id=%s err=%v", businessID, salesOrderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update sales order status"})
			}
			return
		}

		c.JSON(http.StatusOK, saleOrderResponse{
			Sale:    *updated,
			Message: "Sales order status updated successfully",
		})
	}
}

func DeleteSalesOrderRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete sales order handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		salesOrderID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || salesOrderID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Sales order id is required.",
			}))
			return
		}

		if err := reposales.DeleteSalesOrderRepository(pool, businessID, salesOrderID, user.ID, user.FullName); err != nil {
			switch err {
			case reposales.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case reposales.ErrSaleNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Sales order not found"})
			case reposales.ErrSalesOrderCannotDelete:
				c.JSON(http.StatusConflict, gin.H{"message": "This sales order cannot be deleted in its current status"})
			default:
				log.Printf("delete sales order handler: repository failed business_id=%s sales_order_id=%s err=%v", businessID, salesOrderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete sales order"})
			}
			return
		}

		c.JSON(http.StatusOK, deleteSaleOrderResponse{
			Message: "Sales order deleted successfully",
		})
	}
}

func formatSalesOrderDetailDisplayTime(detail reposales.SalesOrderDetail) reposales.SalesOrderDetail {
	detail.SaleOrder.CreatedAt = formatSalesOrderTime(detail.SaleOrder.CreatedAt)
	detail.SaleOrder.UpdatedAt = formatSalesOrderTime(detail.SaleOrder.UpdatedAt)
	detail.SaleOrder.SaleDate = formatSalesOrderTime(detail.SaleOrder.SaleDate)
	for idx := range detail.Activities {
		detail.Activities[idx].ActionDate = formatSalesOrderTime(detail.Activities[idx].ActionDate)
	}
	return detail
}

func formatSalesOrderTime(value string) string {
	parsed, ok := parseSalesOrderTimestamp(value)
	if !ok {
		return value
	}

	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		location = time.Local
	}

	return parsed.In(location).Format("02 Jan 2006, 03:04 PM")
}

func parseSalesOrderTimestamp(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	candidates := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999Z07:00",
		"2006-01-02 15:04:05.999Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999-07:00",
		"2006-01-02 15:04:05-07:00",
	}

	normalized := strings.Replace(value, " ", "T", 1)
	normalized = strings.Replace(normalized, "+00", "+00:00", 1)

	for _, layout := range candidates {
		if parsed, err := time.Parse(layout, normalized); err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
}

func saleOrderFieldErrors(payload *createSaleOrderPayload) map[string]string {
	errs := map[string]string{}
	if payload == nil || payload.CustomerName == nil || strings.TrimSpace(*payload.CustomerName) == "" {
		errs["customer_name"] = "Customer is required."
	}
	if payload == nil || payload.LocationID == nil || strings.TrimSpace(*payload.LocationID) == "" {
		errs["location_id"] = "Location is required."
	}
	if payload == nil || payload.SaleDate == nil || strings.TrimSpace(*payload.SaleDate) == "" {
		errs["sale_date"] = "Sale date is required."
	}
	if payload == nil || payload.Status == nil || !allowedSaleStatuses[strings.ToLower(strings.TrimSpace(derefString(payload.Status)))] {
		errs["status"] = "Status is required."
	}
	if payload == nil || len(payload.Items) == 0 {
		errs["items"] = "Add at least one item to the sale order."
	}
	for idx, item := range payload.Items {
		if item.ProductID == nil || strings.TrimSpace(*item.ProductID) == "" {
			errs[formatSaleOrderItemKey(idx, "product_id")] = "Product is required."
		}
		if item.Quantity == nil || *item.Quantity <= 0 {
			errs[formatSaleOrderItemKey(idx, "quantity")] = "Quantity must be greater than zero."
		}
	}
	return errs
}

func formatSaleOrderItemKey(index int, field string) string {
	return "items[" + strconv.Itoa(index) + "]." + field
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func derefBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func reserveOrderItemsValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func validationFailed(errorsMap map[string]string) gin.H {
	if len(errorsMap) == 0 {
		errorsMap = map[string]string{"form": "Validation failed."}
	}
	return gin.H{"message": "Validation failed", "errors": errorsMap}
}

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Code), "business") {
			return true
		}
	}
	return false
}

var allowedSaleStatuses = map[string]bool{
	"draft":              true,
	"pending_approval":   true,
	"approved":           true,
	"processing":         true,
	"ready_for_shipment": true,
	"completed":          true,
}
