package stocktransfer

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
	reposettings "pos/internal/repository/business/settings"
	repostocktransfer "pos/internal/repository/business/stocktransfer"
)

type createStockTransferPayload struct {
	ReferenceNumber  *string                          `json:"reference_number"`
	TransferDate     *string                          `json:"transfer_date"`
	Status           *string                          `json:"status"`
	TransferType     *string                          `json:"type"`
	LocationFromID   *string                          `json:"location_from_id"`
	LocationToID     *string                          `json:"location_to_id"`
	LocationFromName *string                          `json:"location_from_name"`
	LocationToName   *string                          `json:"location_to_name"`
	Currency         *string                          `json:"currency"`
	ShippingCharges  *float64                         `json:"shipping_charges"`
	Notes            *string                          `json:"notes"`
	Items            []createStockTransferItemPayload `json:"items"`
}

type createStockTransferItemPayload struct {
	ID           *string  `json:"id"`
	ProductID    *string  `json:"product_id"`
	ProductID2   *string  `json:"productId"`
	ProductName  *string  `json:"product_name"`
	ProductName2 *string  `json:"productName"`
	SKU          *string  `json:"sku"`
	Unit         *string  `json:"unit"`
	Quantity     *float64 `json:"quantity"`
	UnitPrice    *float64 `json:"unit_price"`
	UnitPrice2   *float64 `json:"unitPrice"`
	Subtotal     *float64 `json:"subtotal"`
}

type stockTransferResponse struct {
	Transfer repostocktransfer.StockTransfer `json:"transfer"`
	Message  string                          `json:"message,omitempty"`
}

type stockTransfersResponse struct {
	Transfers []repostocktransfer.StockTransferListItem `json:"transfers"`
	Message   string                                    `json:"message,omitempty"`
}

func ListStockTransfersRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list stock transfers handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		transfers, err := repostocktransfer.ListStockTransfersRepository(pool, businessID)
		if err != nil {
			switch err {
			case repostocktransfer.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			default:
				log.Printf("list stock transfers handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load stock transfers"})
			}
			return
		}

		c.JSON(http.StatusOK, stockTransfersResponse{
			Transfers: transfers,
			Message:   "Stock transfers loaded successfully",
		})
	}
}

func CreateStockTransferRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create stock transfer handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		var payload createStockTransferPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create stock transfer handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := transferFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		settings, err := reposettings.GetBusinessSettingsRepository(pool, businessID)
		if err != nil {
			log.Printf("create stock transfer handler: load settings failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load business settings"})
			return
		}

		items := make([]repostocktransfer.CreateStockTransferItemInput, 0, len(payload.Items))
		for idx, item := range payload.Items {
			items = append(items, repostocktransfer.CreateStockTransferItemInput{
				ProductID: strings.TrimSpace(derefString(item.ProductID, item.ProductID2)),
				Quantity:  derefFloat(item.Quantity),
				UnitPrice: derefFloat(item.UnitPrice, item.UnitPrice2),
				SortOrder: idx,
			})
		}

		transfer, err := repostocktransfer.CreateStockTransferRepository(pool, repostocktransfer.CreateStockTransferInput{
			BusinessID:            businessID,
			ReferenceNumber:       derefString(payload.ReferenceNumber),
			TransferDate:          derefString(payload.TransferDate),
			Status:                derefString(payload.Status),
			TransferType:          derefString(payload.TransferType),
			LocationFromID:        derefString(payload.LocationFromID),
			LocationToID:          derefString(payload.LocationToID),
			LocationFromName:      derefString(payload.LocationFromName),
			LocationToName:        derefString(payload.LocationToName),
			Currency:              derefString(payload.Currency),
			ShippingCharges:       derefFloat(payload.ShippingCharges),
			Notes:                 derefString(payload.Notes),
			CreatedBy:             user.ID,
			StockAccountingMethod: settings.StockAccountingMethod,
			Items:                 items,
		})
		if err != nil {
			switch {
			case errors.Is(err, repostocktransfer.ErrInvalidStockTransferInput):
				c.JSON(http.StatusBadRequest, validationFailed(transferFieldErrors(&payload)))
			case errors.Is(err, repostocktransfer.ErrStockTransferNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Stock transfer not found"})
			default:
				log.Printf("create stock transfer handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create stock transfer"})
			}
			return
		}

		c.JSON(http.StatusCreated, stockTransferResponse{
			Transfer: *transfer,
			Message:  "Stock transfer created successfully",
		})
	}
}

func GetStockTransferRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get stock transfer handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		stockTransferID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || stockTransferID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Stock transfer id is required."}))
			return
		}

		transfer, err := repostocktransfer.GetStockTransferByIDRepository(pool, businessID, stockTransferID)
		if err != nil {
			switch err {
			case repostocktransfer.ErrStockTransferNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Stock transfer not found"})
			default:
				log.Printf("get stock transfer handler: repository failed business_id=%s transfer_id=%s err=%v", businessID, stockTransferID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load stock transfer"})
			}
			return
		}

		c.JSON(http.StatusOK, stockTransferResponse{
			Transfer: *transfer,
			Message:  "Stock transfer loaded successfully",
		})
	}
}

func transferFieldErrors(payload *createStockTransferPayload) map[string]string {
	errs := make(map[string]string)
	if payload == nil {
		errs["form"] = "Transfer payload is required."
		return errs
	}
	if payload.TransferDate == nil || strings.TrimSpace(derefString(payload.TransferDate)) == "" {
		errs["transfer_date"] = "Transfer date is required."
	}
	if payload.LocationFromID == nil || strings.TrimSpace(derefString(payload.LocationFromID)) == "" {
		errs["location_from_id"] = "Source location is required."
	}
	if payload.LocationToID == nil || strings.TrimSpace(derefString(payload.LocationToID)) == "" {
		errs["location_to_id"] = "Destination location is required."
	}
	if payload.LocationFromID != nil && payload.LocationToID != nil && strings.TrimSpace(derefString(payload.LocationFromID)) == strings.TrimSpace(derefString(payload.LocationToID)) {
		errs["location_to_id"] = "Destination location must be different from source location."
	}
	if len(payload.Items) == 0 {
		errs["items"] = "Add at least one product."
	}
	return errs
}

func validationFailed(errorsMap map[string]string) gin.H {
	return gin.H{"message": "Validation failed", "errors": errorsMap}
}

func hasBusinessRole(roles []auth.RoleResponse) bool {
	if len(roles) == 0 {
		return false
	}
	for _, role := range roles {
		code := strings.ToLower(strings.TrimSpace(role.Code))
		switch code {
		case "superadmin", "super_admin", "admin", "business", "business_owner", "manager", "staff", "cashier":
			return true
		}
	}
	return false
}

func derefString(values ...*string) string {
	for _, value := range values {
		if value == nil {
			continue
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func derefFloat(values ...*float64) float64 {
	for _, value := range values {
		if value == nil {
			continue
		}
		return *value
	}
	return 0
}
