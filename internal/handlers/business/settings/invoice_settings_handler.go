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

type invoiceSettingsPayload struct {
	Name                          string `json:"name"`
	Code                          string `json:"code"`
	ProductLabel                  string `json:"productLabel"`
	QuantityLabel                 string `json:"quantityLabel"`
	UnitPriceLabel                string `json:"unitPriceLabel"`
	SubTotalLabel                 string `json:"subTotalLabel"`
	CategoryHsnCodeLabel          string `json:"categoryHsnCodeLabel"`
	TotalQuantityLabel            string `json:"totalQuantityLabel"`
	ItemDiscountLabel             string `json:"itemDiscountLabel"`
	DiscountedUnitPriceLabel      string `json:"discountedUnitPriceLabel"`
	SubheadingLine1               string `json:"subheadingLine1"`
	SubheadingLine2               string `json:"subheadingLine2"`
	SubheadingLine3               string `json:"subheadingLine3"`
	SubheadingLine4               string `json:"subheadingLine4"`
	SubheadingLine5               string `json:"subheadingLine5"`
	Design                        string `json:"design"`
	PaperSize                     string `json:"paperSize"`
	IsDefault                     bool   `json:"isDefault"`
	ShowLogo                      bool   `json:"showLogo"`
	ShowBusinessDetails           bool   `json:"showBusinessDetails"`
	ShowCustomerDetails           bool   `json:"showCustomerDetails"`
	ShowItemsSku                  bool   `json:"showItemsSku"`
	ShowBrand                     bool   `json:"showBrand"`
	ShowSaleDescription           bool   `json:"showSaleDescription"`
	ShowQrCode                    bool   `json:"showQrCode"`
	ShowProductExpiry             bool   `json:"showProductExpiry"`
	ShowLotNumber                 bool   `json:"showLotNumber"`
	ShowProductImage              bool   `json:"showProductImage"`
	ShowWarrantyName              bool   `json:"showWarrantyName"`
	ShowWarrantyExpiryDate        bool   `json:"showWarrantyExpiryDate"`
	ShowWarrantyDescription       bool   `json:"showWarrantyDescription"`
	ShowTaxBreakdown              bool   `json:"showTaxBreakdown"`
	ShowDiscounts                 bool   `json:"showDiscounts"`
	ShowBarcode                   bool   `json:"showBarcode"`
	BarcodeTotalDueLabel          string `json:"barcodeTotalDueLabel"`
	ShowTotalBalanceDue           bool   `json:"showTotalBalanceDue"`
	BarcodeChangeReturnLabel      string `json:"barcodeChangeReturnLabel"`
	HideAllPrices                 bool   `json:"hideAllPrices"`
	ShowTotalInWords              bool   `json:"showTotalInWords"`
	BarcodeWordFormat             string `json:"barcodeWordFormat"`
	BarcodeTaxSummaryLabel        string `json:"barcodeTaxSummaryLabel"`
	HeaderAlignment               string `json:"headerAlignment"`
	LogoURL                       string `json:"logoUrl"`
	QrShowLabels                  bool   `json:"qrShowLabels"`
	QrShowBusinessName            bool   `json:"qrShowBusinessName"`
	QrShowBusinessLocationAddress bool   `json:"qrShowBusinessLocationAddress"`
	QrShowInvoiceNo               bool   `json:"qrShowInvoiceNo"`
	QrShowSubtotal                bool   `json:"qrShowSubtotal"`
	QrShowTotalAmountWithTax      bool   `json:"qrShowTotalAmountWithTax"`
	QrShowTotalTax                bool   `json:"qrShowTotalTax"`
	QrShowCustomerName            bool   `json:"qrShowCustomerName"`
	QrShowInvoiceUrl              bool   `json:"qrShowInvoiceUrl"`
	QrShowInvoiceDateTime         bool   `json:"qrShowInvoiceDateTime"`
	QrShowBusinessTax1            bool   `json:"qrShowBusinessTax1"`
	InvoiceNote                   string `json:"invoiceNote"`
}

type businessInvoiceSettingsResponse struct {
	models.BusinessInvoiceSettings
	Message string `json:"message"`
}

type businessInvoiceSettingsListResponse struct {
	Layouts []models.BusinessInvoiceSettings `json:"layouts"`
	Message string                           `json:"message"`
}

func GetBusinessInvoiceSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business invoice settings handler: auth lookup failed err=%v", err)
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

		layouts, err := reposettings.ListBusinessInvoiceSettingsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, reposettings.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
				return
			}

			log.Printf("get business invoice settings handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load invoice settings"})
			return
		}

		c.JSON(http.StatusOK, businessInvoiceSettingsListResponse{
			Layouts: layouts,
			Message: "Invoice settings loaded successfully",
		})
	}
}

func GetBusinessInvoiceSettingRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business invoice setting handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		id := strings.TrimSpace(c.Param("id"))
		if businessID == "" || id == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		layout, err := reposettings.GetBusinessInvoiceSettingRepository(pool, businessID, id)
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case errors.Is(err, reposettings.ErrBusinessInvoiceSettingsNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Invoice layout not found"})
			default:
				log.Printf("get business invoice setting handler: repository failed business_id=%s id=%s err=%v", businessID, id, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load invoice layout"})
			}
			return
		}

		c.JSON(http.StatusOK, businessInvoiceSettingsResponse{
			BusinessInvoiceSettings: *layout,
			Message:                 "Invoice layout loaded successfully",
		})
	}
}

func CreateBusinessInvoiceSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create business invoice settings handler: auth lookup failed err=%v", err)
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
			log.Printf("create business invoice settings handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(invoiceSettingsFieldErrors(nil)))
			return
		}

		var payload invoiceSettingsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create business invoice settings handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := invoiceSettingsFieldErrors(&payload); len(errs) > 0 {
			log.Printf("create business invoice settings handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		layout, err := reposettings.CreateBusinessInvoiceSettingsRepository(pool, toCreateInvoiceSettingsInput(businessID, &payload))
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrInvalidBusinessInvoiceSettingsInput):
				c.JSON(http.StatusBadRequest, validationFailed(invoiceSettingsFieldErrors(&payload)))
			case errors.Is(err, reposettings.ErrBusinessInvoiceSettingsDuplicateCode):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"code": "Invoice layout code already exists."}))
			case errors.Is(err, reposettings.ErrBusinessInvoiceSettingsLogoInvalid), errors.Is(err, reposettings.ErrBusinessInvoiceSettingsLogoTooLarge), errors.Is(err, reposettings.ErrBusinessInvoiceSettingsLogoTypeNotAllowed):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"logoUrl": reposettings.BusinessInvoiceSettingsLogoValidationMessage(err)}))
			default:
				log.Printf("create business invoice settings handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save invoice layout"})
			}
			return
		}

		c.JSON(http.StatusCreated, businessInvoiceSettingsResponse{
			BusinessInvoiceSettings: *layout,
			Message:                 "Invoice layout created successfully",
		})
	}
}

func UpdateBusinessInvoiceSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business invoice settings handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		id := strings.TrimSpace(c.Param("id"))
		if businessID == "" || id == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("update business invoice settings handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(invoiceSettingsFieldErrors(nil)))
			return
		}

		var payload invoiceSettingsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update business invoice settings handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := invoiceSettingsFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update business invoice settings handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		layout, err := reposettings.UpdateBusinessInvoiceSettingsRepository(pool, toUpdateInvoiceSettingsInput(businessID, id, &payload))
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrInvalidBusinessInvoiceSettingsInput):
				c.JSON(http.StatusBadRequest, validationFailed(invoiceSettingsFieldErrors(&payload)))
			case errors.Is(err, reposettings.ErrBusinessInvoiceSettingsNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Invoice layout not found"})
			case errors.Is(err, reposettings.ErrBusinessInvoiceSettingsDuplicateCode):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"code": "Invoice layout code already exists."}))
			case errors.Is(err, reposettings.ErrBusinessInvoiceSettingsLogoInvalid), errors.Is(err, reposettings.ErrBusinessInvoiceSettingsLogoTooLarge), errors.Is(err, reposettings.ErrBusinessInvoiceSettingsLogoTypeNotAllowed):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"logoUrl": reposettings.BusinessInvoiceSettingsLogoValidationMessage(err)}))
			default:
				log.Printf("update business invoice settings handler: repository failed business_id=%s id=%s err=%v", businessID, id, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update invoice layout"})
			}
			return
		}

		c.JSON(http.StatusOK, businessInvoiceSettingsResponse{
			BusinessInvoiceSettings: *layout,
			Message:                 "Invoice layout updated successfully",
		})
	}
}

func invoiceSettingsFieldErrors(payload *invoiceSettingsPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || strings.TrimSpace(payload.Name) == "" {
		errs["name"] = "Invoice layout name is required."
	}

	if payload != nil {
		if payload.Design != "" && !allowedInvoiceLayoutDesigns[strings.TrimSpace(payload.Design)] {
			errs["design"] = "Selected layout design is invalid."
		}
		if payload.PaperSize != "" && !allowedInvoicePaperSizes[strings.TrimSpace(payload.PaperSize)] {
			errs["paperSize"] = "Selected paper size is invalid."
		}
		if payload.HeaderAlignment != "" && !allowedInvoiceHeaderAlignments[strings.TrimSpace(payload.HeaderAlignment)] {
			errs["headerAlignment"] = "Selected header alignment is invalid."
		}
		if payload.BarcodeWordFormat != "" && !allowedInvoiceBarcodeWordFormats[strings.TrimSpace(payload.BarcodeWordFormat)] {
			errs["barcodeWordFormat"] = "Selected barcode word format is invalid."
		}
	}

	return errs
}

func toCreateInvoiceSettingsInput(businessID string, payload *invoiceSettingsPayload) reposettings.CreateBusinessInvoiceSettingsInput {
	return reposettings.CreateBusinessInvoiceSettingsInput{
		BusinessInvoiceSettings: reposettings.BusinessInvoiceSettings{
			BusinessID:                    businessID,
			Name:                          payload.Name,
			Code:                          payload.Code,
			ProductLabel:                  payload.ProductLabel,
			QuantityLabel:                 payload.QuantityLabel,
			UnitPriceLabel:                payload.UnitPriceLabel,
			SubTotalLabel:                 payload.SubTotalLabel,
			CategoryHsnCodeLabel:          payload.CategoryHsnCodeLabel,
			TotalQuantityLabel:            payload.TotalQuantityLabel,
			ItemDiscountLabel:             payload.ItemDiscountLabel,
			DiscountedUnitPriceLabel:      payload.DiscountedUnitPriceLabel,
			SubheadingLine1:               payload.SubheadingLine1,
			SubheadingLine2:               payload.SubheadingLine2,
			SubheadingLine3:               payload.SubheadingLine3,
			SubheadingLine4:               payload.SubheadingLine4,
			SubheadingLine5:               payload.SubheadingLine5,
			Design:                        payload.Design,
			PaperSize:                     payload.PaperSize,
			IsDefault:                     payload.IsDefault,
			ShowLogo:                      payload.ShowLogo,
			ShowBusinessDetails:           payload.ShowBusinessDetails,
			ShowCustomerDetails:           payload.ShowCustomerDetails,
			ShowItemsSku:                  payload.ShowItemsSku,
			ShowBrand:                     payload.ShowBrand,
			ShowSaleDescription:           payload.ShowSaleDescription,
			ShowQrCode:                    payload.ShowQrCode,
			ShowProductExpiry:             payload.ShowProductExpiry,
			ShowLotNumber:                 payload.ShowLotNumber,
			ShowProductImage:              payload.ShowProductImage,
			ShowWarrantyName:              payload.ShowWarrantyName,
			ShowWarrantyExpiryDate:        payload.ShowWarrantyExpiryDate,
			ShowWarrantyDescription:       payload.ShowWarrantyDescription,
			ShowTaxBreakdown:              payload.ShowTaxBreakdown,
			ShowDiscounts:                 payload.ShowDiscounts,
			ShowBarcode:                   payload.ShowBarcode,
			BarcodeTotalDueLabel:          payload.BarcodeTotalDueLabel,
			ShowTotalBalanceDue:           payload.ShowTotalBalanceDue,
			BarcodeChangeReturnLabel:      payload.BarcodeChangeReturnLabel,
			HideAllPrices:                 payload.HideAllPrices,
			ShowTotalInWords:              payload.ShowTotalInWords,
			BarcodeWordFormat:             payload.BarcodeWordFormat,
			BarcodeTaxSummaryLabel:        payload.BarcodeTaxSummaryLabel,
			HeaderAlignment:               payload.HeaderAlignment,
			LogoURL:                       payload.LogoURL,
			QrShowLabels:                  payload.QrShowLabels,
			QrShowBusinessName:            payload.QrShowBusinessName,
			QrShowBusinessLocationAddress: payload.QrShowBusinessLocationAddress,
			QrShowInvoiceNo:               payload.QrShowInvoiceNo,
			QrShowSubtotal:                payload.QrShowSubtotal,
			QrShowTotalAmountWithTax:      payload.QrShowTotalAmountWithTax,
			QrShowTotalTax:                payload.QrShowTotalTax,
			QrShowCustomerName:            payload.QrShowCustomerName,
			QrShowInvoiceUrl:              payload.QrShowInvoiceUrl,
			QrShowInvoiceDateTime:         payload.QrShowInvoiceDateTime,
			QrShowBusinessTax1:            payload.QrShowBusinessTax1,
			InvoiceNote:                   payload.InvoiceNote,
		},
	}
}

func toUpdateInvoiceSettingsInput(businessID, id string, payload *invoiceSettingsPayload) reposettings.UpdateBusinessInvoiceSettingsInput {
	return reposettings.UpdateBusinessInvoiceSettingsInput{
		ID: id,
		BusinessInvoiceSettings: reposettings.BusinessInvoiceSettings{
			BusinessID:                    businessID,
			Name:                          payload.Name,
			Code:                          payload.Code,
			ProductLabel:                  payload.ProductLabel,
			QuantityLabel:                 payload.QuantityLabel,
			UnitPriceLabel:                payload.UnitPriceLabel,
			SubTotalLabel:                 payload.SubTotalLabel,
			CategoryHsnCodeLabel:          payload.CategoryHsnCodeLabel,
			TotalQuantityLabel:            payload.TotalQuantityLabel,
			ItemDiscountLabel:             payload.ItemDiscountLabel,
			DiscountedUnitPriceLabel:      payload.DiscountedUnitPriceLabel,
			SubheadingLine1:               payload.SubheadingLine1,
			SubheadingLine2:               payload.SubheadingLine2,
			SubheadingLine3:               payload.SubheadingLine3,
			SubheadingLine4:               payload.SubheadingLine4,
			SubheadingLine5:               payload.SubheadingLine5,
			Design:                        payload.Design,
			PaperSize:                     payload.PaperSize,
			IsDefault:                     payload.IsDefault,
			ShowLogo:                      payload.ShowLogo,
			ShowBusinessDetails:           payload.ShowBusinessDetails,
			ShowCustomerDetails:           payload.ShowCustomerDetails,
			ShowItemsSku:                  payload.ShowItemsSku,
			ShowBrand:                     payload.ShowBrand,
			ShowSaleDescription:           payload.ShowSaleDescription,
			ShowQrCode:                    payload.ShowQrCode,
			ShowProductExpiry:             payload.ShowProductExpiry,
			ShowLotNumber:                 payload.ShowLotNumber,
			ShowProductImage:              payload.ShowProductImage,
			ShowWarrantyName:              payload.ShowWarrantyName,
			ShowWarrantyExpiryDate:        payload.ShowWarrantyExpiryDate,
			ShowWarrantyDescription:       payload.ShowWarrantyDescription,
			ShowTaxBreakdown:              payload.ShowTaxBreakdown,
			ShowDiscounts:                 payload.ShowDiscounts,
			ShowBarcode:                   payload.ShowBarcode,
			BarcodeTotalDueLabel:          payload.BarcodeTotalDueLabel,
			ShowTotalBalanceDue:           payload.ShowTotalBalanceDue,
			BarcodeChangeReturnLabel:      payload.BarcodeChangeReturnLabel,
			HideAllPrices:                 payload.HideAllPrices,
			ShowTotalInWords:              payload.ShowTotalInWords,
			BarcodeWordFormat:             payload.BarcodeWordFormat,
			BarcodeTaxSummaryLabel:        payload.BarcodeTaxSummaryLabel,
			HeaderAlignment:               payload.HeaderAlignment,
			LogoURL:                       payload.LogoURL,
			QrShowLabels:                  payload.QrShowLabels,
			QrShowBusinessName:            payload.QrShowBusinessName,
			QrShowBusinessLocationAddress: payload.QrShowBusinessLocationAddress,
			QrShowInvoiceNo:               payload.QrShowInvoiceNo,
			QrShowSubtotal:                payload.QrShowSubtotal,
			QrShowTotalAmountWithTax:      payload.QrShowTotalAmountWithTax,
			QrShowTotalTax:                payload.QrShowTotalTax,
			QrShowCustomerName:            payload.QrShowCustomerName,
			QrShowInvoiceUrl:              payload.QrShowInvoiceUrl,
			QrShowInvoiceDateTime:         payload.QrShowInvoiceDateTime,
			QrShowBusinessTax1:            payload.QrShowBusinessTax1,
			InvoiceNote:                   payload.InvoiceNote,
		},
	}
}

var allowedInvoiceLayoutDesigns = map[string]bool{
	"classic": true,
	"modern":  true,
	"minimal": true,
	"compact": true,
}

var allowedInvoicePaperSizes = map[string]bool{
	"a4":      true,
	"thermal": true,
}

var allowedInvoiceHeaderAlignments = map[string]bool{
	"left":   true,
	"center": true,
	"right":  true,
}

var allowedInvoiceBarcodeWordFormats = map[string]bool{
	"international": true,
	"indian":        true,
}
