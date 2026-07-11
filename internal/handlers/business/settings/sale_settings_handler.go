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

type updateBusinessSaleSettingsPayload struct {
	DefaultSaleDiscount            *float64 `json:"defaultSaleDiscount"`
	DefaultSaleTax                 *float64 `json:"defaultSaleTax"`
	SaleItemAdditionMethod         *string  `json:"saleItemAdditionMethod"`
	EnableSaleOrder                *bool    `json:"enableSaleOrder"`
	IsPayTermRequired              *bool    `json:"isPayTermRequired"`
	SalePriceIsMinimumSellingPrice *bool    `json:"salePriceIsMinimumSellingPrice"`
	EnableSaleCommissionAgent      *bool    `json:"enableSaleCommissionAgent"`
	CommissionCalculationType      *string  `json:"commissionCalculationType"`
	IsCommissionAgentRequired      *bool    `json:"isCommissionAgentRequired"`
}

type BusinessSaleSettingsResponse struct {
	ID                             string  `json:"id"`
	DefaultSaleDiscount            float64 `json:"defaultSaleDiscount"`
	DefaultSaleTax                 float64 `json:"defaultSaleTax"`
	SaleItemAdditionMethod         string  `json:"saleItemAdditionMethod"`
	EnableSaleOrder                bool    `json:"enableSaleOrder"`
	IsPayTermRequired              bool    `json:"isPayTermRequired"`
	SalePriceIsMinimumSellingPrice bool    `json:"salePriceIsMinimumSellingPrice"`
	EnableSaleCommissionAgent      bool    `json:"enableSaleCommissionAgent"`
	CommissionCalculationType      string  `json:"commissionCalculationType"`
	IsCommissionAgentRequired      bool    `json:"isCommissionAgentRequired"`
	Message                        string  `json:"message"`
}

func GetBusinessSaleSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business sale settings handler: auth lookup failed err=%v", err)
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

		settings, err := reposettings.GetBusinessSaleSettingsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, reposettings.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
				return
			}

			log.Printf("get business sale settings handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load sale settings"})
			return
		}

		c.JSON(http.StatusOK, toBusinessSaleSettingsResponse(settings, "Sale settings loaded successfully"))
	}
}

func UpdateBusinessSaleSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business sale settings handler: auth lookup failed err=%v", err)
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
			log.Printf("update business sale settings handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(saleSettingsFieldErrors(nil)))
			return
		}

		var payload updateBusinessSaleSettingsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update business sale settings handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := saleSettingsFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update business sale settings handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		settings, err := reposettings.UpdateBusinessSaleSettingsRepository(pool, reposettings.UpdateBusinessSaleSettingsInput{
			BusinessID:                     businessID,
			DefaultSaleDiscount:            *payload.DefaultSaleDiscount,
			DefaultSaleTax:                 *payload.DefaultSaleTax,
			SaleItemAdditionMethod:         stringValue(payload.SaleItemAdditionMethod),
			EnableSaleOrder:                boolValue(payload.EnableSaleOrder),
			IsPayTermRequired:              boolValue(payload.IsPayTermRequired),
			SalePriceIsMinimumSellingPrice: boolValue(payload.SalePriceIsMinimumSellingPrice),
			EnableSaleCommissionAgent:      boolValue(payload.EnableSaleCommissionAgent),
			CommissionCalculationType:      stringValue(payload.CommissionCalculationType),
			IsCommissionAgentRequired:      boolValue(payload.IsCommissionAgentRequired),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrInvalidBusinessSettingsInput):
				c.JSON(http.StatusBadRequest, validationFailed(saleSettingsFieldErrors(&payload)))
			default:
				log.Printf("update business sale settings handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save sale settings"})
			}
			return
		}

		c.JSON(http.StatusOK, toBusinessSaleSettingsResponse(settings, "Sale settings saved successfully"))
	}
}

func saleSettingsFieldErrors(payload *updateBusinessSaleSettingsPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.DefaultSaleDiscount == nil {
		errs["defaultSaleDiscount"] = "Default sale discount is required."
	} else if *payload.DefaultSaleDiscount < 0 || *payload.DefaultSaleDiscount > 100 {
		errs["defaultSaleDiscount"] = "Default sale discount must be between 0 and 100."
	}

	if payload == nil || payload.DefaultSaleTax == nil {
		errs["defaultSaleTax"] = "Default sale tax is required."
	} else if *payload.DefaultSaleTax < 0 || *payload.DefaultSaleTax > 100 {
		errs["defaultSaleTax"] = "Default sale tax must be between 0 and 100."
	}

	if payload == nil || payload.SaleItemAdditionMethod == nil || !allowedSaleItemAdditionMethods[strings.TrimSpace(*payload.SaleItemAdditionMethod)] {
		errs["saleItemAdditionMethod"] = "Sale item addition method is required."
	}

	if payload == nil || payload.EnableSaleOrder == nil {
		errs["enableSaleOrder"] = "Enable sale order is required."
	}

	if payload == nil || payload.IsPayTermRequired == nil {
		errs["isPayTermRequired"] = "Pay term required setting is required."
	}

	if payload == nil || payload.SalePriceIsMinimumSellingPrice == nil {
		errs["salePriceIsMinimumSellingPrice"] = "Sale price minimum setting is required."
	}

	if payload == nil || payload.EnableSaleCommissionAgent == nil {
		errs["enableSaleCommissionAgent"] = "Enable sale commission agent is required."
	}

	if payload == nil || payload.CommissionCalculationType == nil || !allowedCommissionCalculationTypes[strings.TrimSpace(*payload.CommissionCalculationType)] {
		errs["commissionCalculationType"] = "Commission calculation type is required."
	}

	if payload == nil || payload.IsCommissionAgentRequired == nil {
		errs["isCommissionAgentRequired"] = "Commission agent required setting is required."
	}

	if payload != nil && payload.EnableSaleCommissionAgent != nil && !*payload.EnableSaleCommissionAgent && payload.IsCommissionAgentRequired != nil && *payload.IsCommissionAgentRequired {
		errs["isCommissionAgentRequired"] = "Commission agent cannot be required when commission agents are disabled."
	}

	return errs
}

func toBusinessSaleSettingsResponse(settings *models.BusinessSaleSettings, message string) BusinessSaleSettingsResponse {
	return BusinessSaleSettingsResponse{
		ID:                             settings.ID,
		DefaultSaleDiscount:            settings.DefaultSaleDiscount,
		DefaultSaleTax:                 settings.DefaultSaleTax,
		SaleItemAdditionMethod:         settings.SaleItemAdditionMethod,
		EnableSaleOrder:                settings.EnableSaleOrder,
		IsPayTermRequired:              settings.IsPayTermRequired,
		SalePriceIsMinimumSellingPrice: settings.SalePriceIsMinimumSellingPrice,
		EnableSaleCommissionAgent:      settings.EnableSaleCommissionAgent,
		CommissionCalculationType:      settings.CommissionCalculationType,
		IsCommissionAgentRequired:      settings.IsCommissionAgentRequired,
		Message:                        message,
	}
}

var allowedSaleItemAdditionMethods = map[string]bool{
	"new_row":           true,
	"increase_quantity": true,
}

var allowedCommissionCalculationTypes = map[string]bool{
	"percentage":   true,
	"fixed_amount": true,
}
