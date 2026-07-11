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

type updateBusinessPosSettingsPayload struct {
	DisableMultiplePay                    *bool   `json:"disableMultiplePay"`
	DisableDraft                          *bool   `json:"disableDraft"`
	DisableExpressCheckout                *bool   `json:"disableExpressCheckout"`
	DisableDiscount                       *bool   `json:"disableDiscount"`
	DisableOrderTax                       *bool   `json:"disableOrderTax"`
	DisableCreditSaleButton               *bool   `json:"disableCreditSaleButton"`
	DisableSuspendSale                    *bool   `json:"disableSuspendSale"`
	SubtotalEditable                      *bool   `json:"subtotalEditable"`
	HideProductSuggestion                 *bool   `json:"hideProductSuggestion"`
	ShowPricingOnProductSuggestionTooltip *bool   `json:"showPricingOnProductSuggestionTooltip"`
	HideRecentTransactions                *bool   `json:"hideRecentTransactions"`
	EnableTransactionDateOnPosScreen      *bool   `json:"enableTransactionDateOnPosScreen"`
	EnableWeighingScale                   *bool   `json:"enableWeighingScale"`
	EnableServiceStaffInProductLine       *bool   `json:"enableServiceStaffInProductLine"`
	IsServiceStaffRequired                *bool   `json:"isServiceStaffRequired"`
	InvoiceScheme                         *string `json:"invoiceScheme"`
	InvoiceLayout                         *string `json:"invoiceLayout"`
	PrintInvoiceOnSuspend                 *bool   `json:"printInvoiceOnSuspend"`
}

type BusinessPosSettingsResponse struct {
	ID                                    string `json:"id"`
	DisableMultiplePay                    bool   `json:"disableMultiplePay"`
	DisableDraft                          bool   `json:"disableDraft"`
	DisableExpressCheckout                bool   `json:"disableExpressCheckout"`
	DisableDiscount                       bool   `json:"disableDiscount"`
	DisableOrderTax                       bool   `json:"disableOrderTax"`
	DisableCreditSaleButton               bool   `json:"disableCreditSaleButton"`
	DisableSuspendSale                    bool   `json:"disableSuspendSale"`
	SubtotalEditable                      bool   `json:"subtotalEditable"`
	HideProductSuggestion                 bool   `json:"hideProductSuggestion"`
	ShowPricingOnProductSuggestionTooltip bool   `json:"showPricingOnProductSuggestionTooltip"`
	HideRecentTransactions                bool   `json:"hideRecentTransactions"`
	EnableTransactionDateOnPosScreen      bool   `json:"enableTransactionDateOnPosScreen"`
	EnableWeighingScale                   bool   `json:"enableWeighingScale"`
	EnableServiceStaffInProductLine       bool   `json:"enableServiceStaffInProductLine"`
	IsServiceStaffRequired                bool   `json:"isServiceStaffRequired"`
	InvoiceScheme                         string `json:"invoiceScheme"`
	InvoiceLayout                         string `json:"invoiceLayout"`
	PrintInvoiceOnSuspend                 bool   `json:"printInvoiceOnSuspend"`
	Message                               string `json:"message"`
}

func GetBusinessPosSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business pos settings handler: auth lookup failed err=%v", err)
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

		settings, err := reposettings.GetBusinessPosSettingsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, reposettings.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
				return
			}

			log.Printf("get business pos settings handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load POS settings"})
			return
		}

		c.JSON(http.StatusOK, toBusinessPosSettingsResponse(settings, "POS settings loaded successfully"))
	}
}

func UpdateBusinessPosSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business pos settings handler: auth lookup failed err=%v", err)
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
			log.Printf("update business pos settings handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(posSettingsFieldErrors(nil)))
			return
		}

		var payload updateBusinessPosSettingsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update business pos settings handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := posSettingsFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update business pos settings handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		settings, err := reposettings.UpdateBusinessPosSettingsRepository(pool, reposettings.UpdateBusinessPosSettingsInput{
			BusinessID:                            businessID,
			DisableMultiplePay:                    boolValue(payload.DisableMultiplePay),
			DisableDraft:                          boolValue(payload.DisableDraft),
			DisableExpressCheckout:                boolValue(payload.DisableExpressCheckout),
			DisableDiscount:                       boolValue(payload.DisableDiscount),
			DisableOrderTax:                       boolValue(payload.DisableOrderTax),
			DisableCreditSaleButton:               boolValue(payload.DisableCreditSaleButton),
			DisableSuspendSale:                    boolValue(payload.DisableSuspendSale),
			SubtotalEditable:                      boolValue(payload.SubtotalEditable),
			HideProductSuggestion:                 boolValue(payload.HideProductSuggestion),
			ShowPricingOnProductSuggestionTooltip: boolValue(payload.ShowPricingOnProductSuggestionTooltip),
			HideRecentTransactions:                boolValue(payload.HideRecentTransactions),
			EnableTransactionDateOnPosScreen:      boolValue(payload.EnableTransactionDateOnPosScreen),
			EnableWeighingScale:                   boolValue(payload.EnableWeighingScale),
			EnableServiceStaffInProductLine:       boolValue(payload.EnableServiceStaffInProductLine),
			IsServiceStaffRequired:                boolValue(payload.IsServiceStaffRequired),
			InvoiceScheme:                         stringValue(payload.InvoiceScheme),
			InvoiceLayout:                         stringValue(payload.InvoiceLayout),
			PrintInvoiceOnSuspend:                 boolValue(payload.PrintInvoiceOnSuspend),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrInvalidBusinessSettingsInput):
				c.JSON(http.StatusBadRequest, validationFailed(posSettingsFieldErrors(&payload)))
			default:
				log.Printf("update business pos settings handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save POS settings"})
			}
			return
		}

		c.JSON(http.StatusOK, toBusinessPosSettingsResponse(settings, "POS settings saved successfully"))
	}
}

func posSettingsFieldErrors(payload *updateBusinessPosSettingsPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.DisableMultiplePay == nil {
		errs["disableMultiplePay"] = "Disable multiple pay is required."
	}
	if payload == nil || payload.DisableDraft == nil {
		errs["disableDraft"] = "Disable draft is required."
	}
	if payload == nil || payload.DisableExpressCheckout == nil {
		errs["disableExpressCheckout"] = "Disable express checkout is required."
	}
	if payload == nil || payload.DisableDiscount == nil {
		errs["disableDiscount"] = "Disable discount is required."
	}
	if payload == nil || payload.DisableOrderTax == nil {
		errs["disableOrderTax"] = "Disable order tax is required."
	}
	if payload == nil || payload.DisableCreditSaleButton == nil {
		errs["disableCreditSaleButton"] = "Disable credit sale button is required."
	}
	if payload == nil || payload.DisableSuspendSale == nil {
		errs["disableSuspendSale"] = "Disable suspend sale is required."
	}
	if payload == nil || payload.SubtotalEditable == nil {
		errs["subtotalEditable"] = "Subtotal editable is required."
	}
	if payload == nil || payload.HideProductSuggestion == nil {
		errs["hideProductSuggestion"] = "Hide product suggestion is required."
	}
	if payload == nil || payload.ShowPricingOnProductSuggestionTooltip == nil {
		errs["showPricingOnProductSuggestionTooltip"] = "Show pricing on product suggestion tooltip is required."
	}
	if payload == nil || payload.HideRecentTransactions == nil {
		errs["hideRecentTransactions"] = "Hide recent transactions is required."
	}
	if payload == nil || payload.EnableTransactionDateOnPosScreen == nil {
		errs["enableTransactionDateOnPosScreen"] = "Enable transaction date on POS screen is required."
	}
	if payload == nil || payload.EnableWeighingScale == nil {
		errs["enableWeighingScale"] = "Enable weighing scale is required."
	}
	if payload == nil || payload.EnableServiceStaffInProductLine == nil {
		errs["enableServiceStaffInProductLine"] = "Enable service staff in product line is required."
	}
	if payload == nil || payload.IsServiceStaffRequired == nil {
		errs["isServiceStaffRequired"] = "Service staff required is required."
	}
	if payload == nil || payload.InvoiceScheme == nil || !allowedInvoiceSchemes[strings.TrimSpace(*payload.InvoiceScheme)] {
		errs["invoiceScheme"] = "Invoice scheme is required."
	}
	if payload == nil || payload.InvoiceLayout == nil || !allowedInvoiceLayouts[strings.TrimSpace(*payload.InvoiceLayout)] {
		errs["invoiceLayout"] = "Invoice layout is required."
	}
	if payload == nil || payload.PrintInvoiceOnSuspend == nil {
		errs["printInvoiceOnSuspend"] = "Print invoice on suspend is required."
	}

	if payload != nil && payload.HideProductSuggestion != nil && *payload.HideProductSuggestion && payload.ShowPricingOnProductSuggestionTooltip != nil && *payload.ShowPricingOnProductSuggestionTooltip {
		errs["showPricingOnProductSuggestionTooltip"] = "Pricing tooltip cannot be enabled when product suggestions are hidden."
	}

	if payload != nil && payload.EnableServiceStaffInProductLine != nil && !*payload.EnableServiceStaffInProductLine && payload.IsServiceStaffRequired != nil && *payload.IsServiceStaffRequired {
		errs["isServiceStaffRequired"] = "Service staff cannot be required when service staff is disabled."
	}

	return errs
}

func toBusinessPosSettingsResponse(settings *models.BusinessPosSettings, message string) BusinessPosSettingsResponse {
	return BusinessPosSettingsResponse{
		ID:                                    settings.ID,
		DisableMultiplePay:                    settings.DisableMultiplePay,
		DisableDraft:                          settings.DisableDraft,
		DisableExpressCheckout:                settings.DisableExpressCheckout,
		DisableDiscount:                       settings.DisableDiscount,
		DisableOrderTax:                       settings.DisableOrderTax,
		DisableCreditSaleButton:               settings.DisableCreditSaleButton,
		DisableSuspendSale:                    settings.DisableSuspendSale,
		SubtotalEditable:                      settings.SubtotalEditable,
		HideProductSuggestion:                 settings.HideProductSuggestion,
		ShowPricingOnProductSuggestionTooltip: settings.ShowPricingOnProductSuggestionTooltip,
		HideRecentTransactions:                settings.HideRecentTransactions,
		EnableTransactionDateOnPosScreen:      settings.EnableTransactionDateOnPosScreen,
		EnableWeighingScale:                   settings.EnableWeighingScale,
		EnableServiceStaffInProductLine:       settings.EnableServiceStaffInProductLine,
		IsServiceStaffRequired:                settings.IsServiceStaffRequired,
		InvoiceScheme:                         settings.InvoiceScheme,
		InvoiceLayout:                         settings.InvoiceLayout,
		PrintInvoiceOnSuspend:                 settings.PrintInvoiceOnSuspend,
		Message:                               message,
	}
}

var allowedInvoiceSchemes = map[string]bool{
	"default":  true,
	"scheme_a": true,
	"scheme_b": true,
}

var allowedInvoiceLayouts = map[string]bool{
	"default":  true,
	"compact":  true,
	"detailed": true,
}
