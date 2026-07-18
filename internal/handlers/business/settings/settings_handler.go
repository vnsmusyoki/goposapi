package settings

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	"pos/internal/models"
	reposettings "pos/internal/repository/business/settings"
)

type updateBusinessSettingsPayload struct {
	Name                    *string  `json:"name"`
	StartDate               *string  `json:"startDate"`
	DefaultProfitPercentage *float64 `json:"defaultProfitPercentage"`
	Currency                *string  `json:"currency"`
	CurrencySymbolPlacement *string  `json:"currencySymbolPlacement"`
	Timezone                *string  `json:"timezone"`
	LogoURL                 *string  `json:"logoUrl"`
	FinancialYearStartMonth *string  `json:"financialYearStartMonth"`
	StockAccountingMethod   *string  `json:"stockAccountingMethod"`
	PreserveSaleOrderRequests *bool   `json:"preserveSaleOrderRequests"`
	TransactionEditDays     *int     `json:"transactionEditDays"`
	DateFormat              *string  `json:"dateFormat"`
	TimeFormat              *string  `json:"timeFormat"`
	CurrencyPrecision       *int     `json:"currencyPrecision"`
	QuantityPrecision       *int     `json:"quantityPrecision"`
}

type BusinessSettingsResponse struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	StartDate               string   `json:"startDate"`
	DefaultProfitPercentage *float64 `json:"defaultProfitPercentage,omitempty"`
	Currency                string   `json:"currency"`
	CurrencySymbolPlacement string   `json:"currencySymbolPlacement"`
	Timezone                string   `json:"timezone"`
	LogoURL                 string   `json:"logoUrl"`
	FinancialYearStartMonth string   `json:"financialYearStartMonth"`
	StockAccountingMethod   string   `json:"stockAccountingMethod"`
	PreserveSaleOrderRequests bool   `json:"preserveSaleOrderRequests"`
	TransactionEditDays     *int     `json:"transactionEditDays,omitempty"`
	DateFormat              string   `json:"dateFormat"`
	TimeFormat              string   `json:"timeFormat"`
	CurrencyPrecision       *int     `json:"currencyPrecision,omitempty"`
	QuantityPrecision       *int     `json:"quantityPrecision,omitempty"`
	Message                 string   `json:"message"`
}

func GetBusinessSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business settings handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Session expired. Please log in again.",
			})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Business access is required",
			})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		if businessID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		settings, err := reposettings.GetBusinessSettingsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, reposettings.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
				return
			}

			log.Printf("get business settings handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Failed to load business settings",
			})
			return
		}

		c.JSON(http.StatusOK, toBusinessSettingsResponse(settings, "Business settings loaded successfully"))
	}
}

func UpdateBusinessSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business settings handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Session expired. Please log in again.",
			})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Business access is required",
			})
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
			log.Printf("update business settings handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Unable to read request body.",
			}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(businessSettingsFieldErrors(nil)))
			return
		}

		var payload updateBusinessSettingsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update business settings handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		if errs := businessSettingsFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update business settings handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		settings, err := reposettings.UpdateBusinessSettingsRepository(pool, reposettings.UpdateBusinessSettingsInput{
			BusinessID:              businessID,
			Name:                    strings.TrimSpace(*payload.Name),
			StartDate:               strings.TrimSpace(*payload.StartDate),
			DefaultProfitPercentage: *payload.DefaultProfitPercentage,
			Currency:                strings.TrimSpace(*payload.Currency),
			CurrencySymbolPlacement: strings.TrimSpace(*payload.CurrencySymbolPlacement),
			Timezone:                strings.TrimSpace(*payload.Timezone),
			LogoURL:                 derefString(payload.LogoURL),
			FinancialYearStartMonth: strings.TrimSpace(*payload.FinancialYearStartMonth),
			StockAccountingMethod:   strings.TrimSpace(*payload.StockAccountingMethod),
			PreserveSaleOrderRequests: boolValue(payload.PreserveSaleOrderRequests),
			TransactionEditDays:     *payload.TransactionEditDays,
			DateFormat:              strings.TrimSpace(*payload.DateFormat),
			TimeFormat:              strings.TrimSpace(*payload.TimeFormat),
			CurrencyPrecision:       *payload.CurrencyPrecision,
			QuantityPrecision:       *payload.QuantityPrecision,
		})
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrInvalidBusinessSettingsInput):
				c.JSON(http.StatusBadRequest, validationFailed(businessSettingsFieldErrors(&payload)))
			case errors.Is(err, reposettings.ErrInvalidBusinessSettingsLogo), errors.Is(err, reposettings.ErrBusinessSettingsLogoTooLarge), errors.Is(err, reposettings.ErrBusinessSettingsLogoTypeNotAllowed):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"logo_url": reposettings.BusinessSettingsLogoValidationMessage(err),
				}))
			default:
				log.Printf("update business settings handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to save business settings",
				})
			}
			return
		}

		c.JSON(http.StatusOK, toBusinessSettingsResponse(settings, "Business settings saved successfully"))
	}
}

func businessSettingsFieldErrors(payload *updateBusinessSettingsPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Business name is required."
	}

	if payload == nil || payload.StartDate == nil || strings.TrimSpace(*payload.StartDate) == "" {
		errs["startDate"] = "Start date is required."
	} else if _, err := time.Parse("2006-01-02", strings.TrimSpace(*payload.StartDate)); err != nil {
		errs["startDate"] = "Start date must be a valid date."
	}

	if payload == nil || payload.DefaultProfitPercentage == nil {
		errs["defaultProfitPercentage"] = "Default profit percentage is required."
	} else if *payload.DefaultProfitPercentage < 0 || *payload.DefaultProfitPercentage > 100 {
		errs["defaultProfitPercentage"] = "Default profit percentage must be between 0 and 100."
	}

	if payload == nil || payload.Currency == nil || !allowedCurrencies[strings.TrimSpace(*payload.Currency)] {
		errs["currency"] = "Currency is required."
	}

	if payload == nil || payload.CurrencySymbolPlacement == nil || !allowedCurrencySymbolPlacements[strings.TrimSpace(*payload.CurrencySymbolPlacement)] {
		errs["currencySymbolPlacement"] = "Currency symbol placement is required."
	}

	if payload == nil || payload.Timezone == nil || !allowedTimezones[strings.TrimSpace(*payload.Timezone)] {
		errs["timezone"] = "Timezone is required."
	}

	if payload == nil || payload.FinancialYearStartMonth == nil || !allowedMonths[strings.TrimSpace(*payload.FinancialYearStartMonth)] {
		errs["financialYearStartMonth"] = "Financial year start month is required."
	}

	if payload == nil || payload.StockAccountingMethod == nil || !allowedStockMethods[strings.TrimSpace(*payload.StockAccountingMethod)] {
		errs["stockAccountingMethod"] = "Stock accounting method is required."
	}

	if payload == nil || payload.PreserveSaleOrderRequests == nil {
		errs["preserveSaleOrderRequests"] = "Preserve sale order requests is required."
	}

	if payload == nil || payload.TransactionEditDays == nil {
		errs["transactionEditDays"] = "Transaction edit days is required."
	} else if *payload.TransactionEditDays < 0 {
		errs["transactionEditDays"] = "Transaction edit days must be zero or more."
	}

	if payload == nil || payload.DateFormat == nil || !allowedDateFormats[strings.TrimSpace(*payload.DateFormat)] {
		errs["dateFormat"] = "Date format is required."
	}

	if payload == nil || payload.TimeFormat == nil || !allowedTimeFormats[strings.TrimSpace(*payload.TimeFormat)] {
		errs["timeFormat"] = "Time format is required."
	}

	if payload == nil || payload.CurrencyPrecision == nil {
		errs["currencyPrecision"] = "Currency precision is required."
	} else if *payload.CurrencyPrecision < 0 {
		errs["currencyPrecision"] = "Currency precision must be zero or more."
	}

	if payload == nil || payload.QuantityPrecision == nil {
		errs["quantityPrecision"] = "Quantity precision is required."
	} else if *payload.QuantityPrecision < 2 {
		errs["quantityPrecision"] = "Quantity precision must be two or more."
	}

	if payload != nil && payload.LogoURL != nil && strings.TrimSpace(*payload.LogoURL) != "" {
		if err := reposettings.ValidateBusinessSettingsLogoDataURL(*payload.LogoURL); err != nil {
			errs["logoUrl"] = reposettings.BusinessSettingsLogoValidationMessage(err)
		}
	}

	return errs
}

func toBusinessSettingsResponse(settings *models.BusinessSettings, message string) BusinessSettingsResponse {
	response := BusinessSettingsResponse{
		ID:                      settings.ID,
		Name:                    settings.Name,
		StartDate:               settings.StartDate,
		Currency:                settings.Currency,
		CurrencySymbolPlacement: settings.CurrencySymbolPlacement,
		Timezone:                settings.Timezone,
		LogoURL:                 settings.LogoURL,
		FinancialYearStartMonth: settings.FinancialYearStartMonth,
		StockAccountingMethod:   settings.StockAccountingMethod,
		PreserveSaleOrderRequests: settings.PreserveSaleOrderRequests,
		DateFormat:              settings.DateFormat,
		TimeFormat:              settings.TimeFormat,
		Message:                 message,
	}

	if settings.DefaultProfitPercentage != nil {
		value := *settings.DefaultProfitPercentage
		response.DefaultProfitPercentage = &value
	}
	if settings.TransactionEditDays != nil {
		value := *settings.TransactionEditDays
		response.TransactionEditDays = &value
	}
	if settings.CurrencyPrecision != nil {
		value := *settings.CurrencyPrecision
		response.CurrencyPrecision = &value
	}
	if settings.QuantityPrecision != nil {
		value := *settings.QuantityPrecision
		response.QuantityPrecision = &value
	}

	return response
}

func validationFailed(errorsMap map[string]string) gin.H {
	if len(errorsMap) == 0 {
		errorsMap = map[string]string{"form": "Validation failed."}
	}

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

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Code), "business") {
			return true
		}
	}

	return false
}

var allowedCurrencySymbolPlacements = map[string]bool{
	"before": true,
	"after":  true,
}

var allowedCurrencies = map[string]bool{
	"KES": true,
	"USD": true,
	"EUR": true,
	"GBP": true,
	"NGN": true,
}

var allowedTimezones = map[string]bool{
	"Africa/Nairobi":   true,
	"UTC":              true,
	"Europe/London":    true,
	"America/New_York": true,
}

var allowedMonths = map[string]bool{
	"January":   true,
	"February":  true,
	"March":     true,
	"April":     true,
	"May":       true,
	"June":      true,
	"July":      true,
	"August":    true,
	"September": true,
	"October":   true,
	"November":  true,
	"December":  true,
}

var allowedStockMethods = map[string]bool{
	"FIFO":         true,
	"FEFO":         true,
	"LIFO":         true,
	"Average Cost": true,
}

var allowedDateFormats = map[string]bool{
	"DD/MM/YYYY": true,
	"MM/DD/YYYY": true,
	"YYYY-MM-DD": true,
}

var allowedTimeFormats = map[string]bool{
	"12 Hour": true,
	"24 Hour": true,
}
