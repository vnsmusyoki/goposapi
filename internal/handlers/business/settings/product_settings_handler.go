package settings

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
	"pos/internal/models"
	reposettings "pos/internal/repository/business/settings"
)

type updateBusinessProductSettingsPayload struct {
	SKUPrefix             *string `json:"skuPrefix"`
	EnableProductExpiry   *bool   `json:"enableProductExpiry"`
	ExpiryTrackingMethod  *string `json:"expiryTrackingMethod"`
	ExpirySellingBehavior *string `json:"expirySellingBehavior"`
	StopSellingDaysBefore *int    `json:"stopSellingDaysBefore"`
	EnableBrands          *bool   `json:"enableBrands"`
	EnableCategories      *bool   `json:"enableCategories"`
	EnableSubCategories   *bool   `json:"enableSubCategories"`
	EnablePriceTaxInfo    *bool   `json:"enablePriceTaxInfo"`
	DefaultUnit           *string `json:"defaultUnit"`
	EnableSubUnits        *bool   `json:"enableSubUnits"`
	EnableSecondaryUnit   *bool   `json:"enableSecondaryUnit"`
	EnableRacks           *bool   `json:"enableRacks"`
	EnableRow             *bool   `json:"enableRow"`
	EnablePosition        *bool   `json:"enablePosition"`
	EnableWarranty        *bool   `json:"enableWarranty"`
}

type BusinessProductSettingsResponse struct {
	ID                    string `json:"id"`
	SKUPrefix             string `json:"skuPrefix"`
	EnableProductExpiry   bool   `json:"enableProductExpiry"`
	ExpiryTrackingMethod  string `json:"expiryTrackingMethod"`
	ExpirySellingBehavior string `json:"expirySellingBehavior"`
	StopSellingDaysBefore *int   `json:"stopSellingDaysBefore,omitempty"`
	EnableBrands          bool   `json:"enableBrands"`
	EnableCategories      bool   `json:"enableCategories"`
	EnableSubCategories   bool   `json:"enableSubCategories"`
	EnablePriceTaxInfo    bool   `json:"enablePriceTaxInfo"`
	DefaultUnit           string `json:"defaultUnit"`
	EnableSubUnits        bool   `json:"enableSubUnits"`
	EnableSecondaryUnit   bool   `json:"enableSecondaryUnit"`
	EnableRacks           bool   `json:"enableRacks"`
	EnableRow             bool   `json:"enableRow"`
	EnablePosition        bool   `json:"enablePosition"`
	EnableWarranty        bool   `json:"enableWarranty"`
	Message               string `json:"message"`
}

func GetBusinessProductSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business product settings handler: auth lookup failed err=%v", err)
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

		settings, err := reposettings.GetBusinessProductSettingsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, reposettings.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
				return
			}

			log.Printf("get business product settings handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load product settings"})
			return
		}

		c.JSON(http.StatusOK, toBusinessProductSettingsResponse(settings, "Product settings loaded successfully"))
	}
}

func UpdateBusinessProductSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business product settings handler: auth lookup failed err=%v", err)
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
			log.Printf("update business product settings handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(productSettingsFieldErrors(nil)))
			return
		}

		var payload updateBusinessProductSettingsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update business product settings handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := productSettingsFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update business product settings handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		settings, err := reposettings.UpdateBusinessProductSettingsRepository(pool, reposettings.UpdateBusinessProductSettingsInput{
			BusinessID:            businessID,
			SKUPrefix:             stringValue(payload.SKUPrefix),
			EnableProductExpiry:   boolValue(payload.EnableProductExpiry),
			ExpiryTrackingMethod:  stringValue(payload.ExpiryTrackingMethod),
			ExpirySellingBehavior: stringValue(payload.ExpirySellingBehavior),
			StopSellingDaysBefore: payload.StopSellingDaysBefore,
			EnableBrands:          boolValue(payload.EnableBrands),
			EnableCategories:      boolValue(payload.EnableCategories),
			EnableSubCategories:   boolValue(payload.EnableSubCategories),
			EnablePriceTaxInfo:    boolValue(payload.EnablePriceTaxInfo),
			DefaultUnit:           stringValue(payload.DefaultUnit),
			EnableSubUnits:        boolValue(payload.EnableSubUnits),
			EnableSecondaryUnit:   boolValue(payload.EnableSecondaryUnit),
			EnableRacks:           boolValue(payload.EnableRacks),
			EnableRow:             boolValue(payload.EnableRow),
			EnablePosition:        boolValue(payload.EnablePosition),
			EnableWarranty:        boolValue(payload.EnableWarranty),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrInvalidBusinessSettingsInput):
				c.JSON(http.StatusBadRequest, validationFailed(productSettingsFieldErrors(&payload)))
			default:
				log.Printf("update business product settings handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save product settings"})
			}
			return
		}

		c.JSON(http.StatusOK, toBusinessProductSettingsResponse(settings, "Product settings saved successfully"))
	}
}

func productSettingsFieldErrors(payload *updateBusinessProductSettingsPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.DefaultUnit == nil || !allowedDefaultUnits[strings.TrimSpace(*payload.DefaultUnit)] {
		errs["defaultUnit"] = "Default unit is required."
	}

	enableExpiry := payload != nil && payload.EnableProductExpiry != nil && *payload.EnableProductExpiry
	if enableExpiry {
		if payload == nil || payload.ExpiryTrackingMethod == nil || !allowedExpiryTrackingMethods[strings.TrimSpace(*payload.ExpiryTrackingMethod)] {
			errs["expiryTrackingMethod"] = "Expiry tracking method is required."
		}
		if payload == nil || payload.ExpirySellingBehavior == nil || !allowedExpirySellingBehaviors[strings.TrimSpace(*payload.ExpirySellingBehavior)] {
			errs["expirySellingBehavior"] = "Expiry selling behavior is required."
		}
		if payload != nil && payload.ExpirySellingBehavior != nil && strings.TrimSpace(*payload.ExpirySellingBehavior) == "stop_selling_before" {
			if payload.StopSellingDaysBefore == nil || *payload.StopSellingDaysBefore <= 0 {
				errs["stopSellingDaysBefore"] = "Enter how many days before expiry selling should stop."
			}
		}
	}

	return errs
}

func toBusinessProductSettingsResponse(settings *models.BusinessProductSettings, message string) BusinessProductSettingsResponse {
	response := BusinessProductSettingsResponse{
		ID:                    settings.ID,
		SKUPrefix:             settings.SKUPrefix,
		EnableProductExpiry:   settings.EnableProductExpiry,
		ExpiryTrackingMethod:  settings.ExpiryTrackingMethod,
		ExpirySellingBehavior: settings.ExpirySellingBehavior,
		EnableBrands:          settings.EnableBrands,
		EnableCategories:      settings.EnableCategories,
		EnableSubCategories:   settings.EnableSubCategories,
		EnablePriceTaxInfo:    settings.EnablePriceTaxInfo,
		DefaultUnit:           settings.DefaultUnit,
		EnableSubUnits:        settings.EnableSubUnits,
		EnableSecondaryUnit:   settings.EnableSecondaryUnit,
		EnableRacks:           settings.EnableRacks,
		EnableRow:             settings.EnableRow,
		EnablePosition:        settings.EnablePosition,
		EnableWarranty:        settings.EnableWarranty,
		Message:               message,
	}

	if settings.StopSellingDaysBefore != nil {
		value := *settings.StopSellingDaysBefore
		response.StopSellingDaysBefore = &value
	}

	return response
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}

var allowedExpiryTrackingMethods = map[string]bool{
	"item_expiry":              true,
	"manufacturing_and_period": true,
}

var allowedExpirySellingBehaviors = map[string]bool{
	"keep_selling":        true,
	"stop_selling_before": true,
}

var allowedDefaultUnits = map[string]bool{
	"Pieces":   true,
	"Kilogram": true,
	"Litre":    true,
	"Box":      true,
	"Pack":     true,
	"Dozen":    true,
}
