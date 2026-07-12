package admin

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"pos/internal/auth"
	"pos/internal/validation"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	repoadmin "pos/internal/repository/admin"
)

type RegisterBusinessRequest struct {
	Name               *string                  `json:"name"`
	BusinessEmail      *string                  `json:"business_email"`
	BusinessPhone      *string                  `json:"business_phone"`
	RegistrationNumber *string                  `json:"registration_number"`
	Industry           *string                  `json:"industry"`
	OwnerName          *string                  `json:"owner_name"`
	SubscriptionPlan   *string                  `json:"subscription_plan"`
	ManagerID          *string                  `json:"manager_id"`
	Manager            *RegisterBusinessManager `json:"manager,omitempty"`
}

type RegisterBusinessManager struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
}

type RegisterBusinessResponse struct {
	BusinessID   string `json:"business_id"`
	BusinessName string `json:"business_name"`
	ManagerID    string `json:"manager_id"`
	Message      string `json:"message"`
}

type BusinessCatalogItemResponse struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	LegalName          string   `json:"legalName"`
	EIN                string   `json:"ein"`
	Email              string   `json:"email"`
	Phone              string   `json:"phone"`
	Website            string   `json:"website"`
	Address            string   `json:"address"`
	Industry           string   `json:"industry"`
	Status             string   `json:"status"`
	Tier               string   `json:"tier"`
	SubscriptionStatus string   `json:"subscriptionStatus"`
	TotalUsers         int      `json:"totalUsers"`
	TotalLocations     int      `json:"totalLocations"`
	TotalProducts      int      `json:"totalProducts"`
	TotalOrders        int      `json:"totalOrders"`
	MonthlyRevenue     float64  `json:"monthlyRevenue"`
	CreatedAt          string   `json:"createdAt"`
	LastActive         string   `json:"lastActive"`
	IsVerified         bool     `json:"isVerified"`
	IsFeatured         bool     `json:"isFeatured"`
	Flags              []string `json:"flags"`
	SupportTickets     int      `json:"supportTickets"`
	ApiCalls           int      `json:"apiCalls"`
}

type BusinessCatalogResponse struct {
	Businesses []BusinessCatalogItemResponse `json:"businesses"`
	Message    string                        `json:"message"`
}

type SyncBusinessModulesResponse struct {
	BusinessID         string `json:"business_id"`
	InsertedModules    int    `json:"inserted_modules"`
	InsertedSubmodules int    `json:"inserted_submodules"`
	Message            string `json:"message"`
}

func CreateBusinessRequestHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterBusinessRequest
		log.Printf("create business handler: request received remote_ip=%s content_length=%d content_type=%s",
			c.ClientIP(), c.Request.ContentLength, c.GetHeader("Content-Type"))

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("create business handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, businessValidationFailed(map[string]string{
				"form": "Unable to read request body.",
			}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			log.Printf("create business handler: empty request body, validating missing fields")
			c.JSON(http.StatusBadRequest, businessValidationFailed(businessFieldErrors(nil)))
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			log.Printf("create business handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, businessValidationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		req.normalize()

		if errs := businessFieldErrors(&req); len(errs) > 0 {
			log.Printf("create business handler: validation failed missing_or_invalid_fields=%v", errs)
			c.JSON(http.StatusBadRequest, businessValidationFailed(errs))
			return
		}

		log.Printf("create business handler: calling repository business=%q email=%q manager_id=%q", derefString(req.Name), derefString(req.BusinessEmail), derefString(req.ManagerID))
		result, err := repoadmin.CreateBusinessRepository(pool, repoadmin.CreateBusinessInput{
			Name:               derefString(req.Name),
			BusinessEmail:      derefString(req.BusinessEmail),
			BusinessPhone:      validation.NormalizePhoneNumber(derefString(req.BusinessPhone)),
			RegistrationNumber: derefString(req.RegistrationNumber),
			Industry:           derefString(req.Industry),
			OwnerName:          derefString(req.OwnerName),
			SubscriptionPlan:   derefString(req.SubscriptionPlan),
			ExistingManagerID:  derefString(req.ManagerID),
			Manager:            toCreateBusinessManagerInput(req.Manager),
		})
		if err != nil {
			log.Printf("create business handler: repository failed business=%q email=%q err=%v", derefString(req.Name), derefString(req.BusinessEmail), err)
			switch {
			case errors.Is(err, repoadmin.ErrBusinessAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{
					"message": "Business already exists",
				})
			case errors.Is(err, repoadmin.ErrManagerAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{
					"message": "Manager username or email already exists",
				})
			case errors.Is(err, repoadmin.ErrBusinessManagerAlreadyLinked):
				c.JSON(http.StatusConflict, gin.H{
					"message": "This manager is already linked to the business",
				})
			case errors.Is(err, repoadmin.ErrManagerNotFound):
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Manager not found",
				})
			case errors.Is(err, repoadmin.ErrPackageNotFound):
				c.JSON(http.StatusBadRequest, businessValidationFailed(map[string]string{
					"subscription_plan": "Selected package slug does not exist.",
				}))
			case errors.Is(err, repoadmin.ErrInvalidManagerInput), errors.Is(err, repoadmin.ErrInvalidBusinessInput):
				c.JSON(http.StatusBadRequest, businessValidationFailed(map[string]string{"form": err.Error()}))
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to create business",
				})
			}
			return
		}

		message := "Business created successfully"
		if result.CreatedUser {
			message = "Business and manager created successfully"
		} else {
			message = "Business created and manager linked successfully"
		}

		c.JSON(http.StatusCreated, RegisterBusinessResponse{
			BusinessID:   result.BusinessID,
			BusinessName: result.BusinessName,
			ManagerID:    result.ManagerID,
			Message:      message,
		})
	}
}

func ListBusinessesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list businesses handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasAdminRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Admin access is required"})
			return
		}

		businesses, err := repoadmin.ListBusinessesRepository(pool)
		if err != nil {
			log.Printf("list businesses handler: repository failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load businesses"})
			return
		}

		response := make([]BusinessCatalogItemResponse, 0, len(businesses))
		for _, business := range businesses {
			response = append(response, BusinessCatalogItemResponse{
				ID:                 business.ID,
				Name:               business.Name,
				LegalName:          business.LegalName,
				EIN:                business.EIN,
				Email:              business.Email,
				Phone:              business.Phone,
				Website:            business.Website,
				Address:            business.Address,
				Industry:           business.Industry,
				Status:             business.Status,
				Tier:               business.Tier,
				SubscriptionStatus: business.SubscriptionStatus,
				TotalUsers:         business.TotalUsers,
				TotalLocations:     business.TotalLocations,
				TotalProducts:      business.TotalProducts,
				TotalOrders:        business.TotalOrders,
				MonthlyRevenue:     business.MonthlyRevenue,
				CreatedAt:          business.CreatedAt,
				LastActive:         business.LastActive,
				IsVerified:         business.IsVerified,
				IsFeatured:         business.IsFeatured,
				Flags:              business.Flags,
				SupportTickets:     business.SupportTickets,
				ApiCalls:           business.ApiCalls,
			})
		}

		c.JSON(http.StatusOK, BusinessCatalogResponse{
			Businesses: response,
			Message:    "Businesses loaded successfully",
		})
	}
}

func SyncBusinessModulesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("sync business modules handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasAdminRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Admin access is required"})
			return
		}

		businessID := strings.TrimSpace(c.Param("id"))
		if businessID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Business id is required"})
			return
		}

		result, err := repoadmin.SyncBusinessModulesRepository(pool, businessID)
		if err != nil {
			switch {
			case errors.Is(err, repoadmin.ErrBusinessNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Business not found"})
			case errors.Is(err, repoadmin.ErrInvalidBusinessInput):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid business id"})
			default:
				log.Printf("sync business modules handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to sync business modules"})
			}
			return
		}

		c.JSON(http.StatusOK, SyncBusinessModulesResponse{
			BusinessID:         result.BusinessID,
			InsertedModules:    result.InsertedModules,
			InsertedSubmodules: result.InsertedSubmodules,
			Message:            "Business modules synced successfully",
		})
	}
}

func (r *RegisterBusinessRequest) normalize() {
	r.Name = normalizeStringPtr(r.Name, false)
	r.BusinessEmail = normalizeStringPtr(r.BusinessEmail, true)
	r.BusinessPhone = normalizeStringPtr(r.BusinessPhone, false)
	r.RegistrationNumber = normalizeStringPtr(r.RegistrationNumber, false)
	r.Industry = normalizeStringPtr(r.Industry, false)
	r.OwnerName = normalizeStringPtr(r.OwnerName, false)
	r.SubscriptionPlan = normalizeStringPtr(r.SubscriptionPlan, false)
	r.ManagerID = normalizeStringPtr(r.ManagerID, false)

	if r.Manager != nil {
		r.Manager.Username = normalizeStringPtr(r.Manager.Username, false)
		r.Manager.Email = normalizeStringPtr(r.Manager.Email, true)
		r.Manager.Password = normalizeStringPtr(r.Manager.Password, false)
		r.Manager.FullName = normalizeStringPtr(r.Manager.FullName, false)
		r.Manager.Phone = normalizeStringPtr(r.Manager.Phone, false)
	}
}

func businessFieldErrors(req *RegisterBusinessRequest) map[string]string {
	errorsMap := map[string]string{}
	hasManagerDetails := req != nil && req.Manager != nil

	if req == nil || req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		errorsMap["name"] = "Name is required."
	}

	if req == nil || req.BusinessEmail == nil || strings.TrimSpace(*req.BusinessEmail) == "" {
		errorsMap["business_email"] = "Business email is required."
	} else if _, err := mail.ParseAddress(*req.BusinessEmail); err != nil {
		errorsMap["business_email"] = "Enter a valid business email address."
	}

	if req == nil || req.BusinessPhone == nil || strings.TrimSpace(*req.BusinessPhone) == "" {
		errorsMap["business_phone"] = "Business phone is required."
	} else if err := validation.ValidatePhoneNumber(*req.BusinessPhone, true); err != nil {
		errorsMap["business_phone"] = "Business phone must start with 0 and contain 10 digits."
	}

	if req == nil || req.RegistrationNumber == nil || strings.TrimSpace(*req.RegistrationNumber) == "" {
		errorsMap["registration_number"] = "Registration number is required."
	}

	if req == nil || req.Industry == nil || strings.TrimSpace(*req.Industry) == "" {
		errorsMap["industry"] = "Industry is required."
	}

	if req == nil || req.OwnerName == nil || strings.TrimSpace(*req.OwnerName) == "" {
		errorsMap["owner_name"] = "Owner name is required."
	}

	if req == nil || req.SubscriptionPlan == nil || strings.TrimSpace(*req.SubscriptionPlan) == "" {
		errorsMap["subscription_plan"] = "Subscription plan is required."
	}

	if !hasManagerDetails && (req == nil || req.ManagerID == nil || strings.TrimSpace(*req.ManagerID) == "") {
		errorsMap["manager_id"] = "Manager ID is required when manager details are not provided."
	}

	if req != nil && req.ManagerID != nil && strings.TrimSpace(*req.ManagerID) != "" && hasManagerDetails {
		errorsMap["manager"] = "Provide either manager_id or manager details, not both."
	}

	if hasManagerDetails {
		if req.Manager.Username == nil || strings.TrimSpace(*req.Manager.Username) == "" {
			errorsMap["manager.username"] = "Username is required."
		}
		if req.Manager.Email == nil || strings.TrimSpace(*req.Manager.Email) == "" {
			errorsMap["manager.email"] = "Email is required."
		} else if _, err := mail.ParseAddress(*req.Manager.Email); err != nil {
			errorsMap["manager.email"] = "Enter a valid email address."
		}
		if req.Manager.Password == nil || strings.TrimSpace(*req.Manager.Password) == "" {
			errorsMap["manager.password"] = "Password is required."
		} else if len(strings.TrimSpace(*req.Manager.Password)) < 8 {
			errorsMap["manager.password"] = "Password must be at least 8 characters."
		}
		if req.Manager.FullName == nil || strings.TrimSpace(*req.Manager.FullName) == "" {
			errorsMap["manager.full_name"] = "Full name is required."
		}
		if req.Manager.Phone == nil || strings.TrimSpace(*req.Manager.Phone) == "" {
			errorsMap["manager.phone"] = "Phone is required."
		} else if err := validation.ValidatePhoneNumber(*req.Manager.Phone, true); err != nil {
			errorsMap["manager.phone"] = "Phone must start with 0 and contain 10 digits."
		}
	} else if req == nil || req.ManagerID == nil || strings.TrimSpace(*req.ManagerID) == "" {
		errorsMap["manager.username"] = "Username is required."
		errorsMap["manager.email"] = "Email is required."
		errorsMap["manager.password"] = "Password is required."
		errorsMap["manager.full_name"] = "Full name is required."
		errorsMap["manager.phone"] = "Phone is required."
	}

	return errorsMap
}

func toCreateBusinessManagerInput(req *RegisterBusinessManager) *repoadmin.CreateBusinessManagerInput {
	if req == nil {
		return nil
	}

	return &repoadmin.CreateBusinessManagerInput{
		Username: derefString(req.Username),
		Email:    derefString(req.Email),
		Password: derefString(req.Password),
		FullName: derefString(req.FullName),
		Phone:    validation.NormalizePhoneNumber(derefString(req.Phone)),
	}
}

func businessValidationFailed(errorsMap map[string]string) gin.H {
	if len(errorsMap) == 0 {
		errorsMap = map[string]string{"form": "Validation failed."}
	}

	return gin.H{
		"message": "Validation failed",
		"errors":  errorsMap,
	}
}

func normalizeStringPtr(value *string, lower bool) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if lower {
		trimmed = strings.ToLower(trimmed)
	}

	return &trimmed
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}
