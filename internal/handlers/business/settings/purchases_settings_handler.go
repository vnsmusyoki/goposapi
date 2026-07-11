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

type updateBusinessPurchasesSettingsPayload struct {
	EnableEditingProductPriceFromPurchaseScreen *bool `json:"enableEditingProductPriceFromPurchaseScreen"`
	EnablePurchaseStatus                        *bool `json:"enablePurchaseStatus"`
	EnableLotNumber                             *bool `json:"enableLotNumber"`
	EnablePurchaseOrder                         *bool `json:"enablePurchaseOrder"`
}

type BusinessPurchasesSettingsResponse struct {
	ID                                          string `json:"id"`
	EnableEditingProductPriceFromPurchaseScreen bool   `json:"enableEditingProductPriceFromPurchaseScreen"`
	EnablePurchaseStatus                        bool   `json:"enablePurchaseStatus"`
	EnableLotNumber                             bool   `json:"enableLotNumber"`
	EnablePurchaseOrder                         bool   `json:"enablePurchaseOrder"`
	Message                                     string `json:"message"`
}

func GetBusinessPurchasesSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business purchases settings handler: auth lookup failed err=%v", err)
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

		settings, err := reposettings.GetBusinessPurchasesSettingsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, reposettings.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
				return
			}

			log.Printf("get business purchases settings handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load purchases settings"})
			return
		}

		c.JSON(http.StatusOK, toBusinessPurchasesSettingsResponse(settings, "Purchases settings loaded successfully"))
	}
}

func UpdateBusinessPurchasesSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business purchases settings handler: auth lookup failed err=%v", err)
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
			log.Printf("update business purchases settings handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(purchasesSettingsFieldErrors(nil)))
			return
		}

		var payload updateBusinessPurchasesSettingsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update business purchases settings handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := purchasesSettingsFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update business purchases settings handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		settings, err := reposettings.UpdateBusinessPurchasesSettingsRepository(pool, reposettings.UpdateBusinessPurchasesSettingsInput{
			BusinessID: businessID,
			EnableEditingProductPriceFromPurchaseScreen: boolValue(payload.EnableEditingProductPriceFromPurchaseScreen),
			EnablePurchaseStatus:                        boolValue(payload.EnablePurchaseStatus),
			EnableLotNumber:                             boolValue(payload.EnableLotNumber),
			EnablePurchaseOrder:                         boolValue(payload.EnablePurchaseOrder),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrInvalidBusinessSettingsInput):
				c.JSON(http.StatusBadRequest, validationFailed(purchasesSettingsFieldErrors(&payload)))
			default:
				log.Printf("update business purchases settings handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save purchases settings"})
			}
			return
		}

		c.JSON(http.StatusOK, toBusinessPurchasesSettingsResponse(settings, "Purchases settings saved successfully"))
	}
}

func purchasesSettingsFieldErrors(payload *updateBusinessPurchasesSettingsPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.EnableEditingProductPriceFromPurchaseScreen == nil {
		errs["enableEditingProductPriceFromPurchaseScreen"] = "Editing product price setting is required."
	}
	if payload == nil || payload.EnablePurchaseStatus == nil {
		errs["enablePurchaseStatus"] = "Purchase status setting is required."
	}
	if payload == nil || payload.EnableLotNumber == nil {
		errs["enableLotNumber"] = "Lot number setting is required."
	}
	if payload == nil || payload.EnablePurchaseOrder == nil {
		errs["enablePurchaseOrder"] = "Purchase order setting is required."
	}

	return errs
}

func toBusinessPurchasesSettingsResponse(settings *models.BusinessPurchasesSettings, message string) BusinessPurchasesSettingsResponse {
	return BusinessPurchasesSettingsResponse{
		ID: settings.ID,
		EnableEditingProductPriceFromPurchaseScreen: settings.EnableEditingProductPriceFromPurchaseScreen,
		EnablePurchaseStatus:                        settings.EnablePurchaseStatus,
		EnableLotNumber:                             settings.EnableLotNumber,
		EnablePurchaseOrder:                         settings.EnablePurchaseOrder,
		Message:                                     message,
	}
}
