package supplier

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	reposupplier "pos/internal/repository/business/supplier"
)

type supplierStockReportItemResponse struct {
	ID                     string  `json:"id"`
	ProductID              string  `json:"productId"`
	ProductName            string  `json:"productName"`
	SKU                    string  `json:"sku"`
	CategoryName           string  `json:"categoryName"`
	LocationID             string  `json:"locationId"`
	LocationName           string  `json:"locationName"`
	SuppliedBySupplier     float64 `json:"suppliedBySupplier"`
	SoldAlreadyForSupplier float64 `json:"soldAlreadyForSupplier"`
	QuantityAvailable      float64 `json:"quantityAvailable"`
	CostPrice              float64 `json:"costPrice"`
	SellingPrice           float64 `json:"sellingPrice"`
	Status                 string  `json:"status"`
	LastUpdated            string  `json:"lastUpdated"`
}

type supplierStockReportResponse struct {
	Items []supplierStockReportItemResponse `json:"items"`
}

func GetBusinessSupplierStockReportRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("supplier stock report handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			return
		}

		supplierID := strings.TrimSpace(c.Param("id"))
		if supplierID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Supplier ID is required."})
			return
		}

		items, err := reposupplier.GetBusinessSupplierStockReportRepository(pool, businessID, supplierID)
		if err != nil {
			switch {
			case errors.Is(err, reposupplier.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("supplier stock report handler: repository failed business_id=%s supplier_id=%s err=%v", businessID, supplierID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load supplier stock report"})
			}
			return
		}

		responseItems := make([]supplierStockReportItemResponse, 0, len(items))
		for _, item := range items {
			responseItems = append(responseItems, supplierStockReportItemResponse{
				ID:                     item.ID,
				ProductID:              item.ProductID,
				ProductName:            item.ProductName,
				SKU:                    item.SKU,
				CategoryName:           item.CategoryName,
				LocationID:             item.LocationID,
				LocationName:           item.LocationName,
				SuppliedBySupplier:     item.SuppliedBySupplier,
				SoldAlreadyForSupplier: item.SoldAlreadyForSupplier,
				QuantityAvailable:      item.QuantityAvailable,
				CostPrice:              item.CostPrice,
				SellingPrice:           item.SellingPrice,
				Status:                 item.Status,
				LastUpdated:            item.LastUpdated,
			})
		}

		c.JSON(http.StatusOK, supplierStockReportResponse{Items: responseItems})
	}
}
