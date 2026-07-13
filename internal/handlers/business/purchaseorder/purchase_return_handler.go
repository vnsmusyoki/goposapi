package purchaseorder

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repopurchaseorder "pos/internal/repository/business/purchaseorder"
)

type purchaseReturnableStockResponse struct {
	Items   []repopurchaseorder.PurchaseReturnableStockGroup `json:"items"`
	Message string                                           `json:"message"`
}

func SearchPurchaseReturnableStockRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("search purchase returnable stock handler: auth lookup failed err=%v", err)
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

		query := strings.TrimSpace(c.Query("query"))
		if query == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"query": "Enter a product name or SKU to search.",
			}))
			return
		}

		items, err := repopurchaseorder.SearchPurchaseReturnableStockRepository(pool, businessID, query, c.Query("locationId"), c.Query("supplierId"))
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			default:
				log.Printf("search purchase returnable stock handler: repository failed business_id=%s query=%s err=%v", businessID, query, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to search returnable stock"})
			}
			return
		}

		c.JSON(http.StatusOK, purchaseReturnableStockResponse{
			Items:   items,
			Message: "Returnable stock loaded successfully",
		})
	}
}
