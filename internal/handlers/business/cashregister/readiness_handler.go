package cashregister

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
	repocashregister "pos/internal/repository/business/cashregister"
)

type openRegisterPayload struct {
	BusinessLocationID *string  `json:"business_location_id"`
	OpeningCashAmount  *float64 `json:"opening_cash_amount"`
	Notes              *string  `json:"notes"`
}

func OpenCashRegisterRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("open cash register handler: auth lookup failed err=%v", err)
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

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Unable to read request body."})
			return
		}

		var payload openRegisterPayload
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &payload); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Request body must be valid JSON."})
				return
			}
		}

		register, err := repocashregister.OpenCashRegisterRepository(pool, repocashregister.OpenRegisterInput{
			BusinessID:         businessID,
			BusinessLocationID: derefString(payload.BusinessLocationID),
			OpenedBy:           user.ID,
			OpeningCashAmount:  floatValue(payload.OpeningCashAmount, 0),
			Notes:              derefString(payload.Notes),
		})
		if err != nil {
			switch {
			case errors.Is(err, repocashregister.ErrInvalidRegisterInput):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Business location and opening cash amount are required."})
			case errors.Is(err, repocashregister.ErrLocationNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Business location was not found."})
			case errors.Is(err, repocashregister.ErrActiveRegisterExists):
				c.JSON(http.StatusConflict, gin.H{"message": "There is already an open cash register for this location."})
			default:
				log.Printf("open cash register handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to open cash register"})
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"register": register,
			"message":  "Cash register opened successfully",
		})
	}
}

func GetPosReadinessRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("POS readiness handler: auth lookup failed err=%v", err)
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

		readiness, err := repocashregister.GetPosReadinessRepository(
			pool,
			businessID,
			user.ID,
			c.Query("business_location_id"),
		)
		if err != nil {
			switch {
			case errors.Is(err, repocashregister.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			case errors.Is(err, repocashregister.ErrLocationNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Business location was not found."})
			default:
				log.Printf("POS readiness handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load POS readiness checks"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"businessLocationId":    readiness.BusinessLocationID,
			"businessLocationName":  readiness.BusinessLocationName,
			"hasActiveCashRegister": readiness.HasActiveCashRegister,
			"activeRegister":        readiness.ActiveRegister,
			"printerConfigured":     readiness.PrinterConfigured,
			"printerTestRequired":   readiness.PrinterTestRequired,
			"mpesaConfigured":       readiness.MpesaConfigured,
			"mpesaStkPushEnabled":   readiness.MpesaStkPushEnabled,
			"paymentMethods":        readiness.PaymentMethods,
			"blockingReasons":       readiness.BlockingReasons,
			"warnings":              readiness.Warnings,
			"message":               "POS readiness checks loaded successfully",
		})
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func floatValue(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		normalized := strings.ToLower(strings.TrimSpace(role.Name))
		if normalized == "business" || normalized == "business_admin" || normalized == "business_manager" || normalized == "business_staff" {
			return true
		}
	}
	return false
}
