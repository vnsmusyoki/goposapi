package supplier

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	"pos/internal/models"
	reposupplier "pos/internal/repository/business/supplier"
	"pos/internal/validation"
)

type createBusinessSupplierPayload struct {
	SupplierType           *string  `json:"supplier_type"`
	ContactID              *string  `json:"contact_id"`
	Prefix                 *string  `json:"prefix"`
	FirstName              *string  `json:"first_name"`
	MiddleName             *string  `json:"middle_name"`
	LastName               *string  `json:"last_name"`
	BusinessName           *string  `json:"business_name"`
	Mobile                 *string  `json:"mobile"`
	AlternateContactNumber *string  `json:"alternate_contact_number"`
	Landline               *string  `json:"landline"`
	Email                  *string  `json:"email"`
	TaxNumber              *string  `json:"tax_number"`
	OpeningBalance         *float64 `json:"opening_balance"`
	PayTermsType           *string  `json:"pay_terms_type"`
	PayTermsValue          *int     `json:"pay_terms_value"`
	AddressLine1           *string  `json:"address_line_1"`
	AddressLine2           *string  `json:"address_line_2"`
	City                   *string  `json:"city"`
	State                  *string  `json:"state"`
	Country                *string  `json:"country"`
	ZipCode                *string  `json:"zip_code"`
	Website                *string  `json:"website"`
	Notes                  *string  `json:"notes"`
}

type businessSupplierResponse struct {
	ID                     string   `json:"id"`
	BusinessID             string   `json:"businessId"`
	SupplierType           string   `json:"supplierType"`
	ContactID              string   `json:"contactId"`
	Prefix                 string   `json:"prefix"`
	FirstName              string   `json:"firstName"`
	MiddleName             string   `json:"middleName"`
	LastName               string   `json:"lastName"`
	BusinessName           string   `json:"businessName"`
	Mobile                 string   `json:"mobile"`
	AlternateContactNumber string   `json:"alternateContactNumber"`
	Landline               string   `json:"landline"`
	Email                  string   `json:"email"`
	TaxNumber              string   `json:"taxNumber"`
	OpeningBalance         float64  `json:"openingBalance"`
	PayTermsType           string   `json:"payTermsType"`
	PayTermsValue          int      `json:"payTermsValue"`
	AddressLine1           string   `json:"addressLine1"`
	AddressLine2           string   `json:"addressLine2"`
	City                   string   `json:"city"`
	State                  string   `json:"state"`
	Country                string   `json:"country"`
	ZipCode                string   `json:"zipCode"`
	Website                string   `json:"website"`
	Notes                  string   `json:"notes"`
	Name                   string   `json:"name"`
	CompanyName            string   `json:"companyName"`
	Phone                  string   `json:"phone"`
	AlternatePhone         string   `json:"alternatePhone"`
	Address                string   `json:"address"`
	RegistrationNumber     string   `json:"registrationNumber"`
	Status                 string   `json:"status"`
	Tier                   string   `json:"tier"`
	Rating                 float64  `json:"rating"`
	TotalPurchases         int      `json:"totalPurchases"`
	TotalAmount            float64  `json:"totalAmount"`
	OutstandingBalance     float64  `json:"outstandingBalance"`
	PaymentTerms           string   `json:"paymentTerms"`
	LeadTime               int      `json:"leadTime"`
	Categories             []string `json:"categories"`
	PaymentMethods         []string `json:"paymentMethods"`
	BankName               string   `json:"bankName"`
	BankAccount            string   `json:"bankAccount"`
	BankBranch             string   `json:"bankBranch"`
	ContactPerson          string   `json:"contactPerson"`
	ContactPersonPhone     string   `json:"contactPersonPhone"`
	ContactPersonEmail     string   `json:"contactPersonEmail"`
	IsVerified             bool     `json:"isVerified"`
	IsFeatured             bool     `json:"isFeatured"`
	CreatedAt              string   `json:"createdAt"`
	UpdatedAt              string   `json:"updatedAt"`
	Message                string   `json:"message,omitempty"`
}

type businessSupplierListResponse struct {
	Suppliers []businessSupplierResponse `json:"suppliers"`
	Message   string                     `json:"message"`
}

func ListBusinessSuppliersRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list business suppliers handler: auth lookup failed err=%v", err)
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

		suppliers, err := reposupplier.ListBusinessSuppliersRepository(pool, businessID)
		if err != nil {
			switch err {
			case reposupplier.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list business suppliers handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load suppliers"})
			}
			return
		}

		responseSuppliers := make([]businessSupplierResponse, 0, len(suppliers))
		for _, supplier := range suppliers {
			responseSuppliers = append(responseSuppliers, toBusinessSupplierResponse(supplier))
		}

		c.JSON(http.StatusOK, businessSupplierListResponse{
			Suppliers: responseSuppliers,
			Message:   "Suppliers loaded successfully",
		})
	}
}

func CreateBusinessSupplierRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("create business supplier handler: request started remote_ip=%s content_length=%d", c.ClientIP(), c.Request.ContentLength)

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("create business supplier handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(supplierFieldErrors(nil)))
			return
		}

		var payload createBusinessSupplierPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create business supplier handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := supplierFieldErrors(&payload); len(errs) > 0 {
			log.Printf("create business supplier handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create business supplier handler: auth lookup failed err=%v", err)
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

		createdSupplier, err := reposupplier.CreateBusinessSupplierRepository(pool, reposupplier.BusinessSupplierInput{
			BusinessID:             businessID,
			SupplierType:           derefString(payload.SupplierType),
			ContactID:              derefString(payload.ContactID),
			Prefix:                 derefString(payload.Prefix),
			FirstName:              derefString(payload.FirstName),
			MiddleName:             derefString(payload.MiddleName),
			LastName:               derefString(payload.LastName),
			BusinessName:           derefString(payload.BusinessName),
			Mobile:                 validation.NormalizePhoneNumber(derefString(payload.Mobile)),
			AlternateContactNumber: validation.NormalizePhoneNumber(derefString(payload.AlternateContactNumber)),
			Landline:               validation.NormalizePhoneNumber(derefString(payload.Landline)),
			Email:                  derefString(payload.Email),
			TaxNumber:              derefString(payload.TaxNumber),
			OpeningBalance:         derefFloat64(payload.OpeningBalance),
			PayTermsType:           derefString(payload.PayTermsType),
			PayTermsValue:          derefInt(payload.PayTermsValue),
			AddressLine1:           derefString(payload.AddressLine1),
			AddressLine2:           derefString(payload.AddressLine2),
			City:                   derefString(payload.City),
			State:                  derefString(payload.State),
			Country:                derefString(payload.Country),
			ZipCode:                derefString(payload.ZipCode),
			Website:                derefString(payload.Website),
			Notes:                  derefString(payload.Notes),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposupplier.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case errors.Is(err, reposupplier.ErrInvalidBusinessSupplierInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			case errors.Is(err, reposupplier.ErrBusinessSupplierContactIDAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Contact ID already exists for this business."})
			default:
				log.Printf("create business supplier handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create supplier"})
			}
			return
		}

		response := toBusinessSupplierResponse(*createdSupplier)
		response.Message = "Supplier created successfully"
		c.JSON(http.StatusCreated, response)
	}
}

func supplierFieldErrors(payload *createBusinessSupplierPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.SupplierType == nil || strings.TrimSpace(*payload.SupplierType) == "" {
		errs["supplier_type"] = "Supplier type is required."
	} else {
		switch strings.ToLower(strings.TrimSpace(*payload.SupplierType)) {
		case "individual", "business":
		default:
			errs["supplier_type"] = "Supplier type must be either individual or business."
		}
	}

	if payload == nil || payload.Mobile == nil || strings.TrimSpace(*payload.Mobile) == "" {
		errs["mobile"] = "Mobile number is required."
	} else if err := validation.ValidatePhoneNumber(*payload.Mobile, true); err != nil {
		errs["mobile"] = "Mobile number must start with 0 and contain 10 digits."
	}

	if payload != nil && payload.AlternateContactNumber != nil && strings.TrimSpace(*payload.AlternateContactNumber) != "" {
		if err := validation.ValidatePhoneNumber(*payload.AlternateContactNumber, false); err != nil {
			errs["alternate_contact_number"] = "Alternate contact number must start with 0 and contain 10 digits."
		}
	}

	if payload != nil && payload.Landline != nil && strings.TrimSpace(*payload.Landline) != "" {
		if err := validation.ValidatePhoneNumber(*payload.Landline, false); err != nil {
			errs["landline"] = "Landline must start with 0 and contain 10 digits."
		}
	}

	if payload == nil || payload.PayTermsType == nil || strings.TrimSpace(*payload.PayTermsType) == "" {
		errs["pay_terms_type"] = "Pay terms type is required."
	} else {
		switch strings.ToLower(strings.TrimSpace(*payload.PayTermsType)) {
		case "days", "months":
		default:
			errs["pay_terms_type"] = "Pay terms type must be either days or months."
		}
	}

	if payload == nil || payload.PayTermsValue == nil {
		errs["pay_terms_value"] = "Pay terms value is required."
	} else if *payload.PayTermsValue < 0 {
		errs["pay_terms_value"] = "Pay terms value cannot be negative."
	}

	if payload == nil || payload.AddressLine1 == nil || strings.TrimSpace(*payload.AddressLine1) == "" {
		errs["address_line_1"] = "Address line 1 is required."
	}
	if payload == nil || payload.City == nil || strings.TrimSpace(*payload.City) == "" {
		errs["city"] = "City is required."
	}
	if payload == nil || payload.Country == nil || strings.TrimSpace(*payload.Country) == "" {
		errs["country"] = "Country is required."
	}

	if payload != nil && payload.Email != nil && strings.TrimSpace(*payload.Email) != "" {
		if !strings.Contains(strings.TrimSpace(*payload.Email), "@") {
			errs["email"] = "Email must be a valid email address."
		}
	}

	if payload != nil && payload.Website != nil && strings.TrimSpace(*payload.Website) != "" {
		if _, err := url.ParseRequestURI(strings.TrimSpace(*payload.Website)); err != nil {
			errs["website"] = "Website must be a valid URL."
		}
	}

	supplierType := ""
	if payload != nil && payload.SupplierType != nil {
		supplierType = strings.ToLower(strings.TrimSpace(*payload.SupplierType))
	}

	switch supplierType {
	case "individual":
		if payload == nil || payload.Prefix == nil || strings.TrimSpace(*payload.Prefix) == "" {
			errs["prefix"] = "Prefix is required for an individual supplier."
		}
		if payload == nil || payload.FirstName == nil || strings.TrimSpace(*payload.FirstName) == "" {
			errs["first_name"] = "First name is required for an individual supplier."
		}
		if payload == nil || payload.LastName == nil || strings.TrimSpace(*payload.LastName) == "" {
			errs["last_name"] = "Last name is required for an individual supplier."
		}
	case "business":
		if payload == nil || payload.BusinessName == nil || strings.TrimSpace(*payload.BusinessName) == "" {
			errs["business_name"] = "Business name is required for a business supplier."
		}
	}

	if payload != nil && payload.OpeningBalance != nil && *payload.OpeningBalance < 0 {
		errs["opening_balance"] = "Opening balance cannot be negative."
	}

	return errs
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

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Code), "business") {
			return true
		}
	}
	return false
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func derefFloat64(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func toBusinessSupplierResponse(supplier models.BusinessSupplier) businessSupplierResponse {
	return businessSupplierResponse{
		ID:                     supplier.ID,
		BusinessID:             supplier.BusinessID,
		SupplierType:           supplier.SupplierType,
		ContactID:              supplier.ContactID,
		Prefix:                 supplier.Prefix,
		FirstName:              supplier.FirstName,
		MiddleName:             supplier.MiddleName,
		LastName:               supplier.LastName,
		BusinessName:           supplier.BusinessName,
		Mobile:                 supplier.Mobile,
		AlternateContactNumber: supplier.AlternateContactNumber,
		Landline:               supplier.Landline,
		Email:                  supplier.Email,
		TaxNumber:              supplier.TaxNumber,
		OpeningBalance:         supplier.OpeningBalance,
		PayTermsType:           supplier.PayTermsType,
		PayTermsValue:          supplier.PayTermsValue,
		AddressLine1:           supplier.AddressLine1,
		AddressLine2:           supplier.AddressLine2,
		City:                   supplier.City,
		State:                  supplier.State,
		Country:                supplier.Country,
		ZipCode:                supplier.ZipCode,
		Website:                supplier.Website,
		Notes:                  supplier.Notes,
		Name:                   buildSupplierDisplayName(supplier),
		CompanyName:            supplier.BusinessName,
		Phone:                  supplier.Mobile,
		AlternatePhone:         supplier.AlternateContactNumber,
		Address:                buildSupplierAddress(supplier),
		RegistrationNumber:     supplier.ContactID,
		Status:                 supplier.Status,
		Tier:                   supplier.Tier,
		Rating:                 supplier.Rating,
		TotalPurchases:         supplier.TotalPurchases,
		TotalAmount:            supplier.TotalAmount,
		OutstandingBalance:     supplier.OutstandingBalance,
		PaymentTerms:           buildPaymentTerms(supplier),
		LeadTime:               supplier.LeadTime,
		Categories:             []string{},
		PaymentMethods:         []string{},
		BankName:               "",
		BankAccount:            "",
		BankBranch:             "",
		ContactPerson:          buildSupplierDisplayName(supplier),
		ContactPersonPhone:     supplier.Mobile,
		ContactPersonEmail:     supplier.Email,
		IsVerified:             supplier.IsVerified,
		IsFeatured:             supplier.IsFeatured,
		CreatedAt:              supplier.CreatedAt,
		UpdatedAt:              supplier.UpdatedAt,
	}
}

func buildSupplierDisplayName(supplier models.BusinessSupplier) string {
	if strings.EqualFold(strings.TrimSpace(supplier.SupplierType), "business") {
		if strings.TrimSpace(supplier.BusinessName) != "" {
			return strings.TrimSpace(supplier.BusinessName)
		}
	}

	parts := make([]string, 0, 4)
	if value := strings.TrimSpace(supplier.Prefix); value != "" {
		parts = append(parts, strings.Title(strings.ToLower(value)))
	}
	if value := strings.TrimSpace(supplier.FirstName); value != "" {
		parts = append(parts, value)
	}
	if value := strings.TrimSpace(supplier.MiddleName); value != "" {
		parts = append(parts, value)
	}
	if value := strings.TrimSpace(supplier.LastName); value != "" {
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return "Supplier"
	}
	return strings.Join(parts, " ")
}

func buildSupplierAddress(supplier models.BusinessSupplier) string {
	parts := []string{}
	if value := strings.TrimSpace(supplier.AddressLine1); value != "" {
		parts = append(parts, value)
	}
	if value := strings.TrimSpace(supplier.AddressLine2); value != "" {
		parts = append(parts, value)
	}
	if value := strings.TrimSpace(supplier.City); value != "" {
		parts = append(parts, value)
	}
	if value := strings.TrimSpace(supplier.State); value != "" {
		parts = append(parts, value)
	}
	if value := strings.TrimSpace(supplier.Country); value != "" {
		parts = append(parts, value)
	}
	if value := strings.TrimSpace(supplier.ZipCode); value != "" {
		parts = append(parts, value)
	}
	return strings.Join(parts, ", ")
}

func buildPaymentTerms(supplier models.BusinessSupplier) string {
	if supplier.PayTermsValue <= 0 {
		return supplier.PayTermsType
	}
	return strings.TrimSpace(strings.Join([]string{fmt.Sprintf("%d", supplier.PayTermsValue), supplier.PayTermsType}, " "))
}
