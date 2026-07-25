package sales

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
	reposales "pos/internal/repository/business/sales"
)

type createPosSalePayload struct {
	CustomerID    *string                       `json:"customer_id"`
	CustomerName  *string                       `json:"customer_name"`
	CustomerPhone *string                       `json:"customer_phone"`
	CustomerEmail *string                       `json:"customer_email"`
	SaleDate      *string                       `json:"sale_date"`
	LocationID    *string                       `json:"location_id"`
	Notes         *string                       `json:"notes"`
	Subtotal      *float64                      `json:"subtotal"`
	TotalDiscount *float64                      `json:"total_discount"`
	TotalTax      *float64                      `json:"total_tax"`
	GrandTotal    *float64                      `json:"grand_total"`
	ItemsCount    *int                          `json:"items_count"`
	TotalQuantity *float64                      `json:"total_quantity"`
	Items         []createSaleOrderItemPayload  `json:"items"`
	Payments      []createPosSalePaymentPayload `json:"payments"`
}

type createPosSalePaymentPayload struct {
	PaymentMethodCode *string  `json:"payment_method_code"`
	Amount            *float64 `json:"amount"`
	ReferenceNumber   *string  `json:"reference_number"`
	Phone             *string  `json:"phone"`
	Notes             *string  `json:"notes"`
}

type posSaleResponse struct {
	Sale     reposales.Sale             `json:"sale"`
	Payments []reposales.PosSalePayment `json:"payments"`
	Message  string                     `json:"message,omitempty"`
}

func CreatePosSaleRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create POS sale handler: auth lookup failed err=%v", err)
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

		var payload createPosSalePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create POS sale handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		errs := posSaleFieldErrors(&payload)
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

		payments := make([]reposales.CreatePosSalePaymentInput, 0, len(payload.Payments))
		for _, payment := range payload.Payments {
			payments = append(payments, reposales.CreatePosSalePaymentInput{
				PaymentMethodCode: derefString(payment.PaymentMethodCode),
				Amount:            derefFloat(payment.Amount),
				ReferenceNumber:   derefString(payment.ReferenceNumber),
				Phone:             derefString(payment.Phone),
				Notes:             derefString(payment.Notes),
			})
		}

		result, err := reposales.CreatePosSaleRepository(pool, reposales.CreatePosSaleInput{
			BusinessID:    businessID,
			LocationID:    derefString(payload.LocationID),
			CustomerID:    derefString(payload.CustomerID),
			CustomerName:  derefString(payload.CustomerName),
			CustomerPhone: derefString(payload.CustomerPhone),
			CustomerEmail: derefString(payload.CustomerEmail),
			SaleDate:      derefString(payload.SaleDate),
			Notes:         derefString(payload.Notes),
			CreatedBy:     user.ID,
			Items:         items,
			Payments:      payments,
			Subtotal:      derefFloat(payload.Subtotal),
			TotalDiscount: derefFloat(payload.TotalDiscount),
			TotalTax:      derefFloat(payload.TotalTax),
			GrandTotal:    derefFloat(payload.GrandTotal),
			ItemsCount:    derefInt(payload.ItemsCount, len(items)),
			TotalQuantity: derefFloat(payload.TotalQuantity),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposales.ErrInvalidSaleInput), errors.Is(err, reposales.ErrInvalidSalePayment):
				c.JSON(http.StatusBadRequest, validationFailed(posSaleFieldErrors(&payload)))
			case errors.Is(err, reposales.ErrSalePaymentMethodNotEnabled):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"payments": "One or more payment methods are not enabled."}))
			case errors.Is(err, reposales.ErrSaleCreditCustomerRequired):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"customer_id": "Customer is required for credit sales."}))
			case errors.Is(err, reposales.ErrSaleCreditLimitExceeded):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"payments": "Credit amount exceeds the customer's available credit limit."}))
			case errors.Is(err, reposales.ErrActiveCashRegisterRequired):
				c.JSON(http.StatusConflict, gin.H{"message": "Open a cash register for this location before using POS."})
			default:
				log.Printf("create POS sale handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to complete POS sale"})
			}
			return
		}

		c.JSON(http.StatusCreated, posSaleResponse{
			Sale:     result.Sale,
			Payments: result.Payments,
			Message:  "POS sale completed successfully",
		})
	}
}

func posSaleFieldErrors(payload *createPosSalePayload) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(derefString(payload.LocationID)) == "" {
		errs["location_id"] = "Business location is required."
	}
	if derefFloat(payload.GrandTotal) <= 0 {
		errs["grand_total"] = "Grand total must be greater than zero."
	}
	if len(payload.Items) == 0 {
		errs["items"] = "At least one sale item is required."
	}
	if len(payload.Payments) == 0 {
		errs["payments"] = "At least one payment is required."
	}
	for index, payment := range payload.Payments {
		if strings.TrimSpace(derefString(payment.PaymentMethodCode)) == "" {
			errs[formatSaleOrderItemKey(index, "payment_method_code")] = "Payment method is required."
		}
		if derefFloat(payment.Amount) <= 0 {
			errs[formatSaleOrderItemKey(index, "amount")] = "Payment amount must be greater than zero."
		}
	}
	return errs
}
