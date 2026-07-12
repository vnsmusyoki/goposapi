package location

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	"pos/internal/models"
	repolocation "pos/internal/repository/business/location"
	"pos/internal/validation"
)

type createBusinessLocationPayload struct {
	LocationID               *string  `json:"locationId"`
	LocationName             *string  `json:"locationName"`
	Landmark                 *string  `json:"landmark"`
	ExactAddress             *string  `json:"exactAddress"`
	City                     *string  `json:"city"`
	ZipCode                  *string  `json:"zipCode"`
	State                    *string  `json:"state"`
	Country                  *string  `json:"country"`
	Latitude                 *string  `json:"latitude"`
	Longitude                *string  `json:"longitude"`
	Mobile                   *string  `json:"mobile"`
	AlternateContactNumber   *string  `json:"alternateContactNumber"`
	Email                    *string  `json:"email"`
	Website                  *string  `json:"website"`
	InvoiceScheme            *string  `json:"invoiceScheme"`
	PosInvoiceLayout         *string  `json:"posInvoiceLayout"`
	SaleInvoiceLayout        *string  `json:"saleInvoiceLayout"`
	DefaultSellingPriceGroup *string  `json:"defaultSellingPriceGroup"`
	PaymentMethods           []string `json:"paymentMethods"`
	KraPin                   *string  `json:"kraPin"`
	TaxJurisdiction          *string  `json:"taxJurisdiction"`
	IsVatRegistered          *bool    `json:"isVatRegistered"`
	VatNumber                *string  `json:"vatNumber"`
	DefaultTaxType           *string  `json:"defaultTaxType"`
	PricesIncludeTax         *bool    `json:"pricesIncludeTax"`
	IssueTaxInvoices         *bool    `json:"issueTaxInvoices"`
	TaxNote                  *string  `json:"taxNote"`
	EtimsEnabled             *bool    `json:"etimsEnabled"`
	Environment              *string  `json:"environment"`
	IntegrationType          *string  `json:"integrationType"`
	IsHeadOfficeBranch       *bool    `json:"isHeadOfficeBranch"`
	KraBranchID              *string  `json:"kraBranchId"`
	DeviceSerialNumber       *string  `json:"deviceSerialNumber"`
	CmcKey                   *string  `json:"cmcKey"`
	AutoSubmitInvoices       *bool    `json:"autoSubmitInvoices"`
	AllowOfflineSales        *bool    `json:"allowOfflineSales"`
	RetryFailedInvoices      *bool    `json:"retryFailedInvoices"`
	PrintQrCode              *bool    `json:"printQrCode"`
	PrintFiscalDetails       *bool    `json:"printFiscalDetails"`
}

type BusinessLocationResponse struct {
	models.BusinessLocation
	Message string `json:"message"`
}

type BusinessLocationsResponse struct {
	Locations []models.BusinessLocation `json:"locations"`
	Message   string                    `json:"message"`
}

type BusinessLocationDeleteResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func GetBusinessLocationsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business locations handler: auth lookup failed err=%v", err)
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

		locations, err := repolocation.ListBusinessLocationsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, repolocation.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
				return
			}

			log.Printf("get business locations handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load business locations"})
			return
		}

		c.JSON(http.StatusOK, BusinessLocationsResponse{
			Locations: locations,
			Message:   "Business locations loaded successfully",
		})
	}
}

func CreateBusinessLocationRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create business location handler: auth lookup failed err=%v", err)
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
			log.Printf("create business location handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(locationFieldErrors(nil)))
			return
		}

		var payload createBusinessLocationPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create business location handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := locationFieldErrors(&payload); len(errs) > 0 {
			log.Printf("create business location handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		latitude, longitude, locationErrs := parseCoordinates(payload.Latitude, payload.Longitude)
		if len(locationErrs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(locationErrs))
			return
		}

		req := repolocation.CreateBusinessLocationInput{
			BusinessID:               businessID,
			LocationID:               trimmedValue(payload.LocationID),
			LocationName:             trimmedValue(payload.LocationName),
			Landmark:                 trimmedValue(payload.Landmark),
			ExactAddress:             trimmedValue(payload.ExactAddress),
			City:                     trimmedValue(payload.City),
			ZipCode:                  trimmedValue(payload.ZipCode),
			State:                    trimmedValue(payload.State),
			Country:                  trimmedValueOrDefault(payload.Country, "Kenya"),
			Latitude:                 latitude,
			Longitude:                longitude,
			Mobile:                   validation.NormalizePhoneNumber(trimmedValue(payload.Mobile)),
			AlternateContactNumber:   validation.NormalizePhoneNumber(trimmedValue(payload.AlternateContactNumber)),
			Email:                    trimmedValue(payload.Email),
			Website:                  trimmedValue(payload.Website),
			InvoiceScheme:            trimmedValueOrDefault(payload.InvoiceScheme, "default"),
			PosInvoiceLayout:         trimmedValueOrDefault(payload.PosInvoiceLayout, "default"),
			SaleInvoiceLayout:        trimmedValueOrDefault(payload.SaleInvoiceLayout, "default"),
			DefaultSellingPriceGroup: trimmedValueOrDefault(payload.DefaultSellingPriceGroup, "retail"),
			PaymentMethods:           payload.PaymentMethods,
			KraPin:                   trimmedValue(payload.KraPin),
			TaxJurisdiction:          trimmedValueOrDefault(payload.TaxJurisdiction, "Kenya"),
			IsVatRegistered:          boolValue(payload.IsVatRegistered),
			VatNumber:                trimmedValue(payload.VatNumber),
			DefaultTaxType:           trimmedValue(payload.DefaultTaxType),
			PricesIncludeTax:         boolValue(payload.PricesIncludeTax),
			IssueTaxInvoices:         boolValue(payload.IssueTaxInvoices),
			TaxNote:                  trimmedValue(payload.TaxNote),
			EtimsEnabled:             boolValue(payload.EtimsEnabled),
			Environment:              trimmedValueOrDefault(payload.Environment, "sandbox"),
			IntegrationType:          trimmedValueOrDefault(payload.IntegrationType, "OSCU"),
			IsHeadOfficeBranch:       boolValue(payload.IsHeadOfficeBranch),
			KraBranchID:              trimmedValue(payload.KraBranchID),
			DeviceSerialNumber:       trimmedValue(payload.DeviceSerialNumber),
			CmcKey:                   trimmedValue(payload.CmcKey),
			AutoSubmitInvoices:       boolValue(payload.AutoSubmitInvoices),
			AllowOfflineSales:        boolValue(payload.AllowOfflineSales),
			RetryFailedInvoices:      boolValue(payload.RetryFailedInvoices),
			PrintQrCode:              boolValue(payload.PrintQrCode),
			PrintFiscalDetails:       boolValue(payload.PrintFiscalDetails),
		}

		location, err := repolocation.CreateBusinessLocationRepository(pool, req)
		if err != nil {
			switch {
			case errors.Is(err, repolocation.ErrBusinessLocationAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Business location already exists"})
			case errors.Is(err, repolocation.ErrInvalidBusinessLocationInput), errors.Is(err, repolocation.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(locationFieldErrors(&payload)))
			default:
				log.Printf("create business location handler: repository failed business_id=%s location_id=%s name=%q err=%v", businessID, req.LocationID, req.LocationName, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create business location"})
			}
			return
		}

		c.JSON(http.StatusCreated, BusinessLocationResponse{
			BusinessLocation: *location,
			Message:          "Business location created successfully",
		})
	}
}

func DeleteBusinessLocationRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete business location handler: auth lookup failed err=%v", err)
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

		locationID := strings.TrimSpace(c.Param("id"))
		if locationID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"id": "Location id is required.",
			}))
			return
		}

		if err := repolocation.DeleteBusinessLocationRepository(pool, businessID, locationID); err != nil {
			switch {
			case errors.Is(err, repolocation.ErrBusinessLocationNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Business location not found"})
			case errors.Is(err, repolocation.ErrInvalidBusinessLocationInput), errors.Is(err, repolocation.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"id": "Location id is required.",
				}))
			default:
				log.Printf("delete business location handler: repository failed business_id=%s location_id=%s err=%v", businessID, locationID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete business location"})
			}
			return
		}

		c.JSON(http.StatusOK, BusinessLocationDeleteResponse{
			ID:      locationID,
			Message: "Business location deleted successfully",
		})
	}
}

func locationFieldErrors(payload *createBusinessLocationPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.LocationName == nil || strings.TrimSpace(*payload.LocationName) == "" {
		errs["locationName"] = "Location name is required."
	}
	if payload == nil || payload.LocationID == nil || strings.TrimSpace(*payload.LocationID) == "" {
		errs["locationId"] = "Location ID is required."
	}
	if payload == nil || payload.Mobile == nil || strings.TrimSpace(*payload.Mobile) == "" {
		errs["mobile"] = "Mobile number is required."
	} else if err := validation.ValidatePhoneNumber(*payload.Mobile, true); err != nil {
		errs["mobile"] = "Mobile number must start with 0 and contain 10 digits."
	}
	if payload != nil && payload.AlternateContactNumber != nil && strings.TrimSpace(*payload.AlternateContactNumber) != "" {
		if err := validation.ValidatePhoneNumber(*payload.AlternateContactNumber, false); err != nil {
			errs["alternateContactNumber"] = "Alternate contact number must start with 0 and contain 10 digits."
		}
	}
	if payload == nil || payload.KraPin == nil || strings.TrimSpace(*payload.KraPin) == "" {
		errs["kraPin"] = "KRA PIN is required."
	}
	if payload != nil && payload.Email != nil && strings.TrimSpace(*payload.Email) != "" && !isValidEmail(strings.TrimSpace(*payload.Email)) {
		errs["email"] = "Enter a valid email address."
	}
	if payload == nil || len(payload.PaymentMethods) == 0 {
		errs["paymentMethods"] = "Select at least one payment method."
	}
	if payload != nil && payload.InvoiceScheme != nil && !allowedLocationInvoiceSchemes[strings.TrimSpace(*payload.InvoiceScheme)] {
		errs["invoiceScheme"] = "Invoice scheme is invalid."
	}
	if payload != nil && payload.PosInvoiceLayout != nil && !allowedLocationInvoiceLayouts[strings.TrimSpace(*payload.PosInvoiceLayout)] {
		errs["posInvoiceLayout"] = "POS invoice layout is invalid."
	}
	if payload != nil && payload.SaleInvoiceLayout != nil && !allowedLocationInvoiceLayouts[strings.TrimSpace(*payload.SaleInvoiceLayout)] {
		errs["saleInvoiceLayout"] = "Sale invoice layout is invalid."
	}
	if payload != nil && payload.DefaultSellingPriceGroup != nil && !allowedLocationPriceGroups[strings.TrimSpace(*payload.DefaultSellingPriceGroup)] {
		errs["defaultSellingPriceGroup"] = "Default selling price group is invalid."
	}
	if payload != nil && payload.Environment != nil && !allowedLocationEnvironments[strings.TrimSpace(*payload.Environment)] {
		errs["environment"] = "Environment is invalid."
	}
	if payload != nil && payload.IntegrationType != nil && !allowedLocationIntegrationTypes[strings.TrimSpace(*payload.IntegrationType)] {
		errs["integrationType"] = "Integration type is invalid."
	}
	if payload != nil && payload.Latitude != nil && strings.TrimSpace(*payload.Latitude) != "" {
		if _, err := strconv.ParseFloat(strings.TrimSpace(*payload.Latitude), 64); err != nil {
			errs["latitude"] = "Latitude must be a valid number."
		}
	}
	if payload != nil && payload.Longitude != nil && strings.TrimSpace(*payload.Longitude) != "" {
		if _, err := strconv.ParseFloat(strings.TrimSpace(*payload.Longitude), 64); err != nil {
			errs["longitude"] = "Longitude must be a valid number."
		}
	}

	return errs
}

func parseCoordinates(latitudeValue, longitudeValue *string) (*float64, *float64, map[string]string) {
	errs := map[string]string{}

	var latitude *float64
	var longitude *float64

	if latitudeValue != nil && strings.TrimSpace(*latitudeValue) != "" {
		value, err := strconv.ParseFloat(strings.TrimSpace(*latitudeValue), 64)
		if err != nil {
			errs["latitude"] = "Latitude must be a valid number."
		} else if value < -90 || value > 90 {
			errs["latitude"] = "Latitude must be between -90 and 90."
		} else {
			latitude = &value
		}
	}

	if longitudeValue != nil && strings.TrimSpace(*longitudeValue) != "" {
		value, err := strconv.ParseFloat(strings.TrimSpace(*longitudeValue), 64)
		if err != nil {
			errs["longitude"] = "Longitude must be a valid number."
		} else if value < -180 || value > 180 {
			errs["longitude"] = "Longitude must be between -180 and 180."
		} else {
			longitude = &value
		}
	}

	if len(errs) > 0 {
		return nil, nil, errs
	}

	return latitude, longitude, nil
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

func trimmedValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func trimmedValueOrDefault(value *string, fallback string) string {
	trimmed := trimmedValue(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Code), "business") {
			return true
		}
	}
	return false
}

var (
	allowedLocationInvoiceSchemes = map[string]bool{
		"default":  true,
		"scheme_a": true,
		"scheme_b": true,
	}
	allowedLocationInvoiceLayouts = map[string]bool{
		"default":  true,
		"compact":  true,
		"detailed": true,
	}
	allowedLocationPriceGroups = map[string]bool{
		"retail":    true,
		"wholesale": true,
		"vip":       true,
	}
	allowedLocationEnvironments = map[string]bool{
		"sandbox":    true,
		"production": true,
	}
	allowedLocationIntegrationTypes = map[string]bool{
		"OSCU": true,
		"VSCU": true,
	}
	emailPattern = regexp.MustCompile(`^\S+@\S+\.\S+$`)
)

func isValidEmail(value string) bool {
	return emailPattern.MatchString(value)
}
