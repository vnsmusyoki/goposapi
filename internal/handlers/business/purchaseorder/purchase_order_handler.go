package purchaseorder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repopurchaseorder "pos/internal/repository/business/purchaseorder"
)

type createPurchaseOrderPayload struct {
	SupplierID                *string                                       `json:"supplier_id"`
	ReferenceNumber           *string                                       `json:"reference_number"`
	OrderDate                 *string                                       `json:"order_date"`
	DeliveryDate              *string                                       `json:"delivery_date"`
	LocationID                *string                                       `json:"location_id"`
	DeliveryAddress           *string                                       `json:"delivery_address"`
	DeliveryCharges           *float64                                      `json:"delivery_charges"`
	DeliveryDocument          *string                                       `json:"delivery_document"`
	OrderDiscountAmount       *float64                                      `json:"order_discount_amount"`
	PaymentTermValue          *int                                          `json:"payment_term_value"`
	PaymentTermUnit           *string                                       `json:"payment_term_unit"`
	Attachment                *string                                       `json:"attachment"`
	Notes                     *string                                       `json:"notes"`
	Status                    *string                                       `json:"status"`
	DeliveryStatus            *string                                       `json:"delivery_status"`
	PaymentStatus             *string                                       `json:"payment_status"`
	ApprovalReminderChannels  []string                                      `json:"approval_reminder_channels"`
	ApprovalReminderMessage   *string                                       `json:"approval_reminder_message"`
	ApprovalReminderReceivers []string                                      `json:"approval_reminder_receivers"`
	StatusChangeReason        *string                                       `json:"status_change_reason"`
	Subtotal                  *float64                                      `json:"subtotal"`
	TotalDiscount             *float64                                      `json:"total_discount"`
	TotalTax                  *float64                                      `json:"total_tax"`
	GrandTotal                *float64                                      `json:"grand_total"`
	ItemsCount                *int                                          `json:"items_count"`
	TotalQuantity             *float64                                      `json:"total_quantity"`
	Items                     []createPurchaseOrderItemPayload              `json:"items"`
	AdditionalExpenses        []createPurchaseOrderAdditionalExpensePayload `json:"additionalExpenses"`
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
	ManufactureDate        *string  `json:"manufactureDate"`
	ExpiryDate             *string  `json:"expiryDate"`
	LotNumber              *string  `json:"lotNumber"`
	ReceivedQuantity       *float64 `json:"receivedQuantity"`
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

type purchaseOrderStatusesResponse struct {
	Statuses []repopurchaseorder.PurchaseOrderStatusDefinition `json:"statuses"`
	Message  string                                            `json:"message"`
}

type purchaseOrderDetailResponse struct {
	PurchaseOrder repopurchaseorder.PurchaseOrder                `json:"purchaseOrder"`
	Supplier      repopurchaseorder.PurchaseOrderSupplierDetails `json:"supplier"`
	Business      repopurchaseorder.PurchaseOrderBusinessDetails `json:"business"`
	Location      repopurchaseorder.PurchaseOrderLocationDetails `json:"location"`
	Items         []repopurchaseorder.PurchaseOrderItem          `json:"items"`
	Activities    []repopurchaseorder.PurchaseOrderLog           `json:"activities"`
	Message       string                                         `json:"message,omitempty"`
}

func ListPurchaseOrderStatusesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list purchase order statuses handler: auth lookup failed err=%v", err)
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

		statuses, err := repopurchaseorder.ListPurchaseOrderStatusesRepository(pool, businessID)
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			default:
				log.Printf("list purchase order statuses handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load purchase order statuses"})
			}
			return
		}

		c.JSON(http.StatusOK, purchaseOrderStatusesResponse{
			Statuses: statuses,
			Message:  "Purchase order statuses loaded successfully",
		})
	}
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

func GetPurchaseOrderRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get purchase order handler: auth lookup failed err=%v", err)
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

		order, err := repopurchaseorder.GetPurchaseOrderDetailRepository(pool, businessID, purchaseOrderID)
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case repopurchaseorder.ErrPurchaseOrderNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase order not found"})
			default:
				log.Printf("get purchase order handler: repository failed business_id=%s order_id=%s err=%v", businessID, purchaseOrderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load purchase order"})
			}
			return
		}

		c.JSON(http.StatusOK, purchaseOrderDetailResponse{
			PurchaseOrder: formatPurchaseOrderDisplayTime(order.PurchaseOrder),
			Supplier:      order.Supplier,
			Business:      order.Business,
			Location:      order.Location,
			Items:         order.Items,
			Activities:    formatPurchaseOrderActivitiesDisplayTime(order.Activities),
			Message:       "Purchase order loaded successfully",
		})
	}
}

func UpdatePurchaseOrderRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("update purchase order handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(purchaseOrderFieldErrors(nil)))
			return
		}

		var payload createPurchaseOrderPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update purchase order handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := purchaseOrderFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update purchase order handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update purchase order handler: auth lookup failed err=%v", err)
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

		existing, err := repopurchaseorder.GetPurchaseOrderByIDRepository(pool, businessID, purchaseOrderID)
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case repopurchaseorder.ErrPurchaseOrderNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase order not found"})
			default:
				log.Printf("update purchase order handler: lookup failed business_id=%s order_id=%s err=%v", businessID, purchaseOrderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load purchase order"})
			}
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
				ManufactureDate:        derefString(item.ManufactureDate),
				ExpiryDate:             derefString(item.ExpiryDate),
				LotNumber:              derefString(item.LotNumber),
				ReceivedQuantity:       float64Ptr(item.ReceivedQuantity),
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

		approvalReminderChannels := normalizeApprovalReminderChannels(payload.ApprovalReminderChannels)
		approvalReminderReceivers := normalizeApprovalReminderReceivers(payload.ApprovalReminderReceivers)
		if containsApprovalReminderChannel(approvalReminderChannels, "sms") || containsApprovalReminderChannel(approvalReminderChannels, "whatsapp") {
			if len(approvalReminderReceivers) == 0 {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"approval_reminder_receivers": "Enter at least one valid phone number for SMS or WhatsApp reminders.",
				}))
				return
			}
		}

		purchaseOrderID = strings.TrimSpace(purchaseOrderID)
		nextStatus := strings.ToLower(strings.TrimSpace(derefString(payload.Status)))
		if err := validatePurchaseOrderStatusTransition(existing.Status, nextStatus); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"status": err.Error(),
			}))
			return
		}
		statusChanged := !strings.EqualFold(existing.Status, nextStatus)
		statusChangeReason := strings.TrimSpace(derefString(payload.StatusChangeReason))
		nextDeliveryStatus := strings.ToLower(strings.TrimSpace(derefString(payload.DeliveryStatus)))
		if statusChanged {
			nextDeliveryStatus = derivePurchaseOrderDeliveryStatus(nextStatus)
		}
		if statusChanged && statusChangeReason == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"status_change_reason": "Please provide a reason for the status change.",
			}))
			return
		}

		note := buildPurchaseOrderUpdateNote(existing.ReferenceNumber, user.FullName, *existing, repopurchaseorder.PurchaseOrder{
			Status:         nextStatus,
			DeliveryStatus: nextDeliveryStatus,
			PaymentStatus:  strings.ToLower(strings.TrimSpace(derefString(payload.PaymentStatus))),
		}, len(items), derefFloat64(payload.GrandTotal))
		if statusChanged {
			note = buildPurchaseOrderStatusChangeNote(existing.ReferenceNumber, user.FullName, existing.Status, nextStatus, statusChangeReason)
		}

		updatedOrder, err := repopurchaseorder.UpdatePurchaseOrderRepository(pool, repopurchaseorder.UpdatePurchaseOrderInput{
			BusinessID:          businessID,
			PurchaseOrderID:     purchaseOrderID,
			SupplierID:          derefString(payload.SupplierID),
			LocationID:          derefString(payload.LocationID),
			ReferenceNumber:     derefString(payload.ReferenceNumber),
			OrderDate:           derefString(payload.OrderDate),
			DeliveryDate:        derefString(payload.DeliveryDate),
			DeliveryAddress:     derefString(payload.DeliveryAddress),
			DeliveryCharges:     derefFloat64(payload.DeliveryCharges),
			DeliveryDocument:    derefString(payload.DeliveryDocument),
			OrderDiscountAmount: derefFloat64(payload.OrderDiscountAmount),
			PaymentTermValue:    derefInt(payload.PaymentTermValue),
			PaymentTermUnit:     derefString(payload.PaymentTermUnit),
			AttachmentName:      derefString(payload.Attachment),
			Notes:               derefString(payload.Notes),
			Status:              derefString(payload.Status),
			DeliveryStatus:      nextDeliveryStatus,
			PaymentStatus:       derefString(payload.PaymentStatus),
			Subtotal:            derefFloat64(payload.Subtotal),
			TotalDiscount:       derefFloat64(payload.TotalDiscount),
			TotalTax:            derefFloat64(payload.TotalTax),
			GrandTotal:          derefFloat64(payload.GrandTotal),
			ItemsCount:          derefInt(payload.ItemsCount),
			TotalQuantity:       derefFloat64(payload.TotalQuantity),
			UpdatedBy:           user.ID,
			Items:               items,
			AdditionalExpenses:  additionalExpenses,
			ActivityAction: func() string {
				if statusChanged {
					return "status_changed"
				}
				return "updated"
			}(),
			ActivityActionedBy:        user.ID,
			ActivityNote:              note,
			PreviousStatus:            existing.Status,
			PreviousDeliveryStatus:    existing.DeliveryStatus,
			PreviousPaymentStatus:     existing.PaymentStatus,
			ApprovalReminderChannels:  normalizeApprovalReminderChannels(payload.ApprovalReminderChannels),
			ApprovalReminderMessage:   derefString(payload.ApprovalReminderMessage),
			ApprovalReminderReceivers: approvalReminderReceivers,
			StatusChangeReason:        statusChangeReason,
		})
		if err != nil {
			switch err {
			case repopurchaseorder.ErrInvalidPurchaseOrderInput:
				c.JSON(http.StatusBadRequest, validationFailed(purchaseOrderFieldErrors(&payload)))
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case repopurchaseorder.ErrPurchaseOrderNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase order not found"})
			default:
				log.Printf("update purchase order handler: repository failed business_id=%s order_id=%s err=%v", businessID, purchaseOrderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update purchase order"})
			}
			return
		}

		c.JSON(http.StatusOK, purchaseOrderResponse{
			PurchaseOrder: formatPurchaseOrderDisplayTime(*updatedOrder),
			Message:       buildPurchaseOrderResponseMessage(derefString(payload.Status), approvalReminderChannels),
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
				ManufactureDate:        derefString(item.ManufactureDate),
				ExpiryDate:             derefString(item.ExpiryDate),
				LotNumber:              derefString(item.LotNumber),
				ReceivedQuantity:       float64Ptr(item.ReceivedQuantity),
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
			BusinessID:          businessID,
			SupplierID:          derefString(payload.SupplierID),
			LocationID:          derefString(payload.LocationID),
			ReferenceNumber:     derefString(payload.ReferenceNumber),
			OrderDate:           derefString(payload.OrderDate),
			DeliveryDate:        derefString(payload.DeliveryDate),
			DeliveryAddress:     derefString(payload.DeliveryAddress),
			DeliveryCharges:     derefFloat64(payload.DeliveryCharges),
			DeliveryDocument:    derefString(payload.DeliveryDocument),
			OrderDiscountAmount: derefFloat64(payload.OrderDiscountAmount),
			PaymentTermValue:    derefInt(payload.PaymentTermValue),
			PaymentTermUnit:     derefString(payload.PaymentTermUnit),
			AttachmentName:      derefString(payload.Attachment),
			Notes:               derefString(payload.Notes),
			Status:              derefString(payload.Status),
			DeliveryStatus:      derefString(payload.DeliveryStatus),
			PaymentStatus:       derefString(payload.PaymentStatus),
			Subtotal:            derefFloat64(payload.Subtotal),
			TotalDiscount:       derefFloat64(payload.TotalDiscount),
			TotalTax:            derefFloat64(payload.TotalTax),
			GrandTotal:          derefFloat64(payload.GrandTotal),
			ItemsCount:          derefInt(payload.ItemsCount),
			TotalQuantity:       derefFloat64(payload.TotalQuantity),
			CreatedBy:           user.ID,
			Items:               items,
			AdditionalExpenses:  additionalExpenses,
			ActivityAction:      "created",
			ActivityActionedBy:  user.ID,
			ActivityNote:        buildPurchaseOrderActivityNote("created", derefString(payload.ReferenceNumber), user.FullName, "", "", len(items), derefFloat64(payload.GrandTotal)),
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

		if err := repopurchaseorder.DeletePurchaseOrderRepository(pool, businessID, purchaseOrderID, user.ID); err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case repopurchaseorder.ErrPurchaseOrderNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase order not found"})
			case repopurchaseorder.ErrPurchaseOrderCannotDelete:
				c.JSON(http.StatusConflict, gin.H{"message": "This purchase order cannot be deleted in its current status"})
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

func buildPurchaseOrderActivityNote(action, referenceNumber, actorName, previousStatus, nextStatus string, itemsCount int, totalAmount float64) string {
	ref := strings.TrimSpace(referenceNumber)
	actor := strings.TrimSpace(actorName)
	if actor == "" {
		actor = "System"
	}

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "created":
		if ref == "" {
			ref = "purchase order"
		}
		return fmt.Sprintf("%s created by %s with %d item(s) totaling %.2f.", titleCase(ref), actor, itemsCount, totalAmount)
	case "status_changed":
		if ref == "" {
			ref = "purchase order"
		}
		if strings.TrimSpace(previousStatus) == "" {
			previousStatus = "unknown"
		}
		if strings.TrimSpace(nextStatus) == "" {
			nextStatus = "unknown"
		}
		return fmt.Sprintf("%s status changed from %s to %s by %s.", titleCase(ref), previousStatus, nextStatus, actor)
	default:
		if ref == "" {
			ref = "purchase order"
		}
		return fmt.Sprintf("%s %s by %s.", titleCase(ref), strings.TrimSpace(action), actor)
	}
}

func buildPurchaseOrderUpdateNote(referenceNumber, actorName string, previousOrder, nextOrder repopurchaseorder.PurchaseOrder, itemsCount int, totalAmount float64) string {
	actor := strings.TrimSpace(actorName)
	if actor == "" {
		actor = "System"
	}

	changes := make([]string, 0, 3)
	if prev, next := strings.ToLower(strings.TrimSpace(previousOrder.Status)), strings.ToLower(strings.TrimSpace(nextOrder.Status)); prev != next {
		changes = append(changes, fmt.Sprintf("status from %s to %s", humanizePurchaseOrderState(prev), humanizePurchaseOrderState(next)))
	}
	if prev, next := strings.ToLower(strings.TrimSpace(previousOrder.DeliveryStatus)), strings.ToLower(strings.TrimSpace(nextOrder.DeliveryStatus)); prev != next {
		changes = append(changes, fmt.Sprintf("delivery status from %s to %s", humanizePurchaseOrderState(prev), humanizePurchaseOrderState(next)))
	}
	if prev, next := strings.ToLower(strings.TrimSpace(previousOrder.PaymentStatus)), strings.ToLower(strings.TrimSpace(nextOrder.PaymentStatus)); prev != next {
		changes = append(changes, fmt.Sprintf("payment status from %s to %s", humanizePurchaseOrderState(prev), humanizePurchaseOrderState(next)))
	}

	ref := strings.TrimSpace(referenceNumber)
	if ref == "" {
		ref = "purchase order"
	}

	summary := fmt.Sprintf("%s updated by %s with %d item(s) totaling %.2f.", titleCase(ref), actor, itemsCount, totalAmount)
	if len(changes) == 0 {
		return summary
	}

	return summary + " Changes: " + strings.Join(changes, "; ") + "."
}

func buildPurchaseOrderStatusChangeNote(referenceNumber, actorName, previousStatus, nextStatus, reason string) string {
	ref := strings.TrimSpace(referenceNumber)
	if ref == "" {
		ref = "purchase order"
	}

	actor := strings.TrimSpace(actorName)
	if actor == "" {
		actor = "System"
	}

	prev := humanizePurchaseOrderState(previousStatus)
	next := humanizePurchaseOrderState(nextStatus)
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "No reason provided."
	}

	return fmt.Sprintf("%s status changed from %s to %s by %s. Reason: %s", titleCase(ref), prev, next, actor, trimmedReason)
}

func derivePurchaseOrderDeliveryStatus(orderStatus string) string {
	switch strings.ToLower(strings.TrimSpace(orderStatus)) {
	case "ordered":
		return "in_transit"
	case "partially_received":
		return "in_transit"
	case "received", "completed", "closed":
		return "delivered"
	case "cancelled":
		return "pending_delivery"
	default:
		return "pending_delivery"
	}
}

func validatePurchaseOrderStatusTransition(previousStatus, nextStatus string) error {
	prev := strings.ToLower(strings.TrimSpace(previousStatus))
	next := strings.ToLower(strings.TrimSpace(nextStatus))

	if prev == "" || next == "" || prev == next {
		return nil
	}

	allowed := map[string][]string{
		"draft":              {"pending_approval", "approved", "cancelled"},
		"pending_approval":   {"approved", "cancelled"},
		"approved":           {"ordered", "cancelled"},
		"ordered":            {"partially_received", "received", "cancelled"},
		"partially_received": {"received", "cancelled"},
		"received":           {"completed", "closed", "cancelled"},
		"completed":          {"closed"},
		"cancelled":          {},
		"closed":             {},
	}

	for _, candidate := range allowed[prev] {
		if candidate == next {
			return nil
		}
	}

	return fmt.Errorf("cannot change purchase order status from %s to %s", humanizePurchaseOrderState(prev), humanizePurchaseOrderState(next))
}

func buildPurchaseOrderResponseMessage(status string, reminderChannels []string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	reminderChannels = normalizeApprovalReminderChannels(reminderChannels)

	if status == "pending_approval" {
		channelLabels := make([]string, 0, len(reminderChannels))
		for _, channel := range reminderChannels {
			channelLabels = append(channelLabels, humanizeReminderChannel(channel))
		}
		if len(channelLabels) == 0 {
			channelLabels = []string{"notification"}
		}
		return fmt.Sprintf("Purchase order moved to pending approval and %s reminder queued successfully.", strings.Join(channelLabels, ", "))
	}

	return "Purchase order updated successfully"
}

func normalizeApprovalReminderChannels(channels []string) []string {
	seen := make(map[string]struct{}, len(channels))
	normalized := make([]string, 0, len(channels))

	for _, channel := range channels {
		channel = strings.ToLower(strings.TrimSpace(channel))
		if channel == "" {
			continue
		}
		switch channel {
		case "notification", "sms", "whatsapp":
		default:
			continue
		}
		if _, exists := seen[channel]; exists {
			continue
		}
		seen[channel] = struct{}{}
		normalized = append(normalized, channel)
	}

	if len(normalized) == 0 {
		return []string{"notification"}
	}

	return normalized
}

func normalizeApprovalReminderReceivers(receivers []string) []string {
	seen := make(map[string]struct{}, len(receivers))
	normalized := make([]string, 0, len(receivers))

	for _, receiver := range receivers {
		receiver = strings.TrimSpace(receiver)
		if receiver == "" {
			continue
		}
		if !strings.HasPrefix(receiver, "0") || len(receiver) != 10 {
			continue
		}
		valid := true
		for _, ch := range receiver {
			if ch < '0' || ch > '9' {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		if _, exists := seen[receiver]; exists {
			continue
		}
		seen[receiver] = struct{}{}
		normalized = append(normalized, receiver)
	}

	return normalized
}

func containsApprovalReminderChannel(channels []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, channel := range channels {
		if strings.ToLower(strings.TrimSpace(channel)) == target {
			return true
		}
	}
	return false
}

func humanizeReminderChannel(channel string) string {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "sms":
		return "SMS"
	case "whatsapp":
		return "WhatsApp"
	default:
		return "Notification"
	}
}

func float64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	next := *value
	return &next
}

func formatPurchaseOrderDisplayTime(order repopurchaseorder.PurchaseOrder) repopurchaseorder.PurchaseOrder {
	order.CreatedAt = formatPurchaseOrderTime(order.CreatedAt)
	order.UpdatedAt = formatPurchaseOrderTime(order.UpdatedAt)
	return order
}

func formatPurchaseOrderActivitiesDisplayTime(activities []repopurchaseorder.PurchaseOrderLog) []repopurchaseorder.PurchaseOrderLog {
	formatted := make([]repopurchaseorder.PurchaseOrderLog, 0, len(activities))
	for _, activity := range activities {
		activity.ActionDate = formatPurchaseOrderTime(activity.ActionDate)
		formatted = append(formatted, activity)
	}
	return formatted
}

func formatPurchaseOrderTime(value string) string {
	parsed, ok := parsePurchaseOrderTimestamp(value)
	if !ok {
		return value
	}

	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		location = time.Local
	}

	return parsed.In(location).Format("02 Jan 2006, 03:04 PM")
}

func parsePurchaseOrderTimestamp(value string) (time.Time, bool) {
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

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	runes := []rune(value)
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}

func humanizePurchaseOrderState(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	if value == "" {
		return "unknown"
	}
	return titleCase(value)
}
