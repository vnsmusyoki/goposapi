package customer

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
	repo "pos/internal/repository/business/customer"
	"pos/internal/validation"
)

type createBusinessCustomerPayload struct {
	ContactID          *string  `json:"contact_id"`
	CustomerCode       *string  `json:"customer_code"`
	FirstName          *string  `json:"first_name"`
	MiddleName         *string  `json:"middle_name"`
	LastName           *string  `json:"last_name"`
	CompanyName        *string  `json:"company_name"`
	Phone              *string  `json:"phone"`
	Email              *string  `json:"email"`
	Address            *string  `json:"address"`
	ShippingAddress    *string  `json:"shipping_address"`
	TaxNumber          *string  `json:"tax_number"`
	OpeningBalance     *float64 `json:"opening_balance"`
	PayTermsType       *string  `json:"pay_terms_type"`
	PayTermsValue      *int     `json:"pay_terms_value"`
	CreditLimit        *float64 `json:"credit_limit"`
	CustomerGroup      *string  `json:"customer_group"`
	AdvanceBalance     *float64 `json:"advance_balance"`
	TotalSaleDue       *float64 `json:"total_sale_due"`
	TotalSellReturnDue *float64 `json:"total_sell_return_due"`
	CustomField1       *string  `json:"custom_field_1"`
	CustomField2       *string  `json:"custom_field_2"`
	CustomField3       *string  `json:"custom_field_3"`
	CustomField4       *string  `json:"custom_field_4"`
	CustomField5       *string  `json:"custom_field_5"`
	Notes              *string  `json:"notes"`
	IsActive           *bool    `json:"is_active"`
}

type customerResponse struct {
	ID                 string  `json:"id"`
	BusinessID         string  `json:"businessId"`
	ContactID          string  `json:"contactId"`
	CustomerCode       string  `json:"customerCode"`
	FirstName          string  `json:"firstName"`
	MiddleName         string  `json:"middleName"`
	LastName           string  `json:"lastName"`
	CompanyName        string  `json:"companyName"`
	Phone              string  `json:"phone"`
	Email              string  `json:"email"`
	Address            string  `json:"address"`
	ShippingAddress    string  `json:"shippingAddress"`
	TaxNumber          string  `json:"taxNumber"`
	OpeningBalance     float64 `json:"openingBalance"`
	PayTermsType       string  `json:"payTermsType"`
	PayTermsValue      int     `json:"payTermsValue"`
	CreditLimit        float64 `json:"creditLimit"`
	CustomerGroup      string  `json:"customerGroup"`
	AdvanceBalance     float64 `json:"advanceBalance"`
	TotalSaleDue       float64 `json:"totalSaleDue"`
	TotalSellReturnDue float64 `json:"totalSellReturnDue"`
	CustomField1       string  `json:"customField1"`
	CustomField2       string  `json:"customField2"`
	CustomField3       string  `json:"customField3"`
	CustomField4       string  `json:"customField4"`
	CustomField5       string  `json:"customField5"`
	Notes              string  `json:"notes"`
	IsActive           bool    `json:"isActive"`
	CreatedBy          string  `json:"createdBy"`
	Deleted            bool    `json:"deleted"`
	DeletedAt          string  `json:"deletedAt"`
	DeletedBy          string  `json:"deletedBy"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
	Name               string  `json:"name"`
	DisplayName        string  `json:"displayName"`
	Message            string  `json:"message,omitempty"`
}

type customerListResponse struct {
	Customers []customerResponse `json:"customers"`
	Message   string             `json:"message"`
}

func ListBusinessCustomersRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list business customers handler: auth lookup failed err=%v", err)
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

		customers, err := repo.ListBusinessCustomersRepository(pool, businessID)
		if err != nil {
			switch err {
			case repo.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list business customers handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load customers"})
			}
			return
		}

		responseCustomers := make([]customerResponse, 0, len(customers))
		for _, customer := range customers {
			responseCustomers = append(responseCustomers, toCustomerResponse(customer))
		}

		c.JSON(http.StatusOK, customerListResponse{
			Customers: responseCustomers,
			Message:   "Customers loaded successfully",
		})
	}
}

func CreateBusinessCustomerRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("create business customer handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(customerFieldErrors(nil)))
			return
		}

		var payload createBusinessCustomerPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create business customer handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := customerFieldErrors(&payload); len(errs) > 0 {
			log.Printf("create business customer handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create business customer handler: auth lookup failed err=%v", err)
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

		isActive := true
		if payload.IsActive != nil {
			isActive = *payload.IsActive
		}

		createdCustomer, err := repo.CreateBusinessCustomerRepository(pool, repo.BusinessCustomerInput{
			BusinessID:         businessID,
			CreatedBy:          user.ID,
			ContactID:          derefString(payload.ContactID),
			CustomerCode:       derefString(payload.CustomerCode),
			FirstName:          derefString(payload.FirstName),
			MiddleName:         derefString(payload.MiddleName),
			LastName:           derefString(payload.LastName),
			CompanyName:        derefString(payload.CompanyName),
			Phone:              validation.NormalizePhoneNumber(derefString(payload.Phone)),
			Email:              derefString(payload.Email),
			Address:            derefString(payload.Address),
			ShippingAddress:    derefString(payload.ShippingAddress),
			TaxNumber:          derefString(payload.TaxNumber),
			OpeningBalance:     derefFloat64(payload.OpeningBalance),
			PayTermsType:       derefString(payload.PayTermsType),
			PayTermsValue:      derefInt(payload.PayTermsValue),
			CreditLimit:        derefFloat64(payload.CreditLimit),
			CustomerGroup:      derefString(payload.CustomerGroup),
			AdvanceBalance:     derefFloat64(payload.AdvanceBalance),
			TotalSaleDue:       derefFloat64(payload.TotalSaleDue),
			TotalSellReturnDue: derefFloat64(payload.TotalSellReturnDue),
			CustomField1:       derefString(payload.CustomField1),
			CustomField2:       derefString(payload.CustomField2),
			CustomField3:       derefString(payload.CustomField3),
			CustomField4:       derefString(payload.CustomField4),
			CustomField5:       derefString(payload.CustomField5),
			Notes:              derefString(payload.Notes),
			IsActive:           isActive,
		})
		if err != nil {
			switch {
			case errors.Is(err, repo.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case errors.Is(err, repo.ErrInvalidBusinessCustomerInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			case errors.Is(err, repo.ErrBusinessCustomerCodeAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Customer code already exists for this business."})
			default:
				log.Printf("create business customer handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create customer"})
			}
			return
		}

		response := toCustomerResponse(*createdCustomer)
		response.Message = "Customer created successfully"
		c.JSON(http.StatusCreated, response)
	}
}

func UpdateBusinessCustomerRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business customer handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		customerID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || customerID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Customer is required."}))
			return
		}

		var payload createBusinessCustomerPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := customerFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		isActive := true
		if payload.IsActive != nil {
			isActive = *payload.IsActive
		}

		updatedCustomer, err := repo.UpdateBusinessCustomerRepository(pool, repo.BusinessCustomerInput{
			BusinessID:         businessID,
			CreatedBy:          user.ID,
			ContactID:          derefString(payload.ContactID),
			CustomerCode:       derefString(payload.CustomerCode),
			FirstName:          derefString(payload.FirstName),
			MiddleName:         derefString(payload.MiddleName),
			LastName:           derefString(payload.LastName),
			CompanyName:        derefString(payload.CompanyName),
			Phone:              validation.NormalizePhoneNumber(derefString(payload.Phone)),
			Email:              derefString(payload.Email),
			Address:            derefString(payload.Address),
			ShippingAddress:    derefString(payload.ShippingAddress),
			TaxNumber:          derefString(payload.TaxNumber),
			OpeningBalance:     derefFloat64(payload.OpeningBalance),
			PayTermsType:       derefString(payload.PayTermsType),
			PayTermsValue:      derefInt(payload.PayTermsValue),
			CreditLimit:        derefFloat64(payload.CreditLimit),
			CustomerGroup:      derefString(payload.CustomerGroup),
			AdvanceBalance:     derefFloat64(payload.AdvanceBalance),
			TotalSaleDue:       derefFloat64(payload.TotalSaleDue),
			TotalSellReturnDue: derefFloat64(payload.TotalSellReturnDue),
			CustomField1:       derefString(payload.CustomField1),
			CustomField2:       derefString(payload.CustomField2),
			CustomField3:       derefString(payload.CustomField3),
			CustomField4:       derefString(payload.CustomField4),
			CustomField5:       derefString(payload.CustomField5),
			Notes:              derefString(payload.Notes),
			IsActive:           isActive,
		}, customerID)
		if err != nil {
			switch {
			case errors.Is(err, repo.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case errors.Is(err, repo.ErrInvalidBusinessCustomerInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			case errors.Is(err, repo.ErrBusinessCustomerCodeAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Customer code already exists for this business."})
			default:
				log.Printf("update business customer handler: repository failed business_id=%s customer_id=%s err=%v", businessID, customerID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update customer"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Customer updated successfully",
			"customer": toCustomerResponse(*updatedCustomer),
		})
	}
}

func DeleteBusinessCustomerRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		customerID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || customerID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Customer is required."}))
			return
		}

		if err := repo.DeleteBusinessCustomerRepository(pool, businessID, customerID, user.ID); err != nil {
			if errors.Is(err, repo.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
				return
			}
			log.Printf("delete business customer handler: repository failed business_id=%s customer_id=%s err=%v", businessID, customerID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete customer"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Customer deleted successfully",
			"id":      customerID,
		})
	}
}

func customerFieldErrors(payload *createBusinessCustomerPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.CustomerCode == nil || strings.TrimSpace(*payload.CustomerCode) == "" {
		errs["customer_code"] = "Customer code is required."
	}

	if payload == nil || payload.Phone == nil || strings.TrimSpace(*payload.Phone) == "" {
		errs["phone"] = "Phone number is required."
	} else if normalized := validation.NormalizePhoneNumber(strings.TrimSpace(*payload.Phone)); normalized == "" {
		errs["phone"] = "Phone number is invalid."
	}

	if payload != nil && payload.Email != nil && strings.TrimSpace(*payload.Email) != "" && !strings.Contains(strings.TrimSpace(*payload.Email), "@") {
		errs["email"] = "Email must be a valid email address."
	}

	if payload != nil && payload.PayTermsType != nil && strings.TrimSpace(*payload.PayTermsType) != "" {
		termType := strings.ToLower(strings.TrimSpace(*payload.PayTermsType))
		if termType != "days" && termType != "months" {
			errs["pay_terms_type"] = "Pay terms type must be either days or months."
		}
	}

	if payload != nil && payload.PayTermsValue != nil && *payload.PayTermsValue < 0 {
		errs["pay_terms_value"] = "Pay terms value cannot be negative."
	}

	if payload != nil && payload.OpeningBalance != nil && *payload.OpeningBalance < 0 {
		errs["opening_balance"] = "Opening balance cannot be negative."
	}

	if payload != nil && payload.CreditLimit != nil && *payload.CreditLimit < 0 {
		errs["credit_limit"] = "Credit limit cannot be negative."
	}

	if payload != nil {
		shippingAddress := strings.TrimSpace(derefString(payload.ShippingAddress))
		if shippingAddress != "" && len(shippingAddress) < 2 {
			errs["shipping_address"] = "Shipping address must be at least 2 characters long."
		}
	}

	hasName := false
	if payload != nil {
		hasName = strings.TrimSpace(derefString(payload.CompanyName)) != "" ||
			strings.TrimSpace(derefString(payload.FirstName)) != "" ||
			strings.TrimSpace(derefString(payload.LastName)) != ""
	}
	if !hasName {
		errs["name"] = "Customer name or company name is required."
	}

	return errs
}

func toCustomerResponse(customer models.BusinessCustomer) customerResponse {
	return customerResponse{
		ID:                 customer.ID,
		BusinessID:         customer.BusinessID,
		ContactID:          customer.ContactID,
		CustomerCode:       customer.CustomerCode,
		FirstName:          customer.FirstName,
		MiddleName:         customer.MiddleName,
		LastName:           customer.LastName,
		CompanyName:        customer.CompanyName,
		Phone:              customer.Phone,
		Email:              customer.Email,
		Address:            customer.Address,
		ShippingAddress:    customer.ShippingAddress,
		TaxNumber:          customer.TaxNumber,
		OpeningBalance:     customer.OpeningBalance,
		PayTermsType:       customer.PayTermsType,
		PayTermsValue:      customer.PayTermsValue,
		CreditLimit:        customer.CreditLimit,
		CustomerGroup:      customer.CustomerGroup,
		AdvanceBalance:     customer.AdvanceBalance,
		TotalSaleDue:       customer.TotalSaleDue,
		TotalSellReturnDue: customer.TotalSellReturnDue,
		CustomField1:       customer.CustomField1,
		CustomField2:       customer.CustomField2,
		CustomField3:       customer.CustomField3,
		CustomField4:       customer.CustomField4,
		CustomField5:       customer.CustomField5,
		Notes:              customer.Notes,
		IsActive:           customer.IsActive,
		CreatedBy:          customer.CreatedBy,
		Deleted:            customer.Deleted,
		DeletedAt:          customer.DeletedAt,
		DeletedBy:          customer.DeletedBy,
		CreatedAt:          customer.CreatedAt,
		UpdatedAt:          customer.UpdatedAt,
		Name:               customer.Name,
		DisplayName:        customer.DisplayName,
	}
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
		if strings.EqualFold(role.Name, "business") || strings.EqualFold(role.Code, "business") {
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
