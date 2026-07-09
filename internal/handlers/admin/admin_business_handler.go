package admin

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	repoadmin "pos/internal/repository/admin"
)

type RegisterBusinessRequest struct {
	Name               string                   `json:"name" binding:"required"`
	BusinessEmail      string                   `json:"business_email" binding:"required,email"`
	BusinessPhone      string                   `json:"business_phone"`
	RegistrationNumber string                   `json:"registration_number"`
	Industry           string                   `json:"industry"`
	OwnerName          string                   `json:"owner_name"`
	SubscriptionPlan   string                   `json:"subscription_plan"`
	ManagerID          string                   `json:"manager_id"`
	Manager            *RegisterBusinessManager `json:"manager,omitempty"`
}

type RegisterBusinessManager struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type RegisterBusinessResponse struct {
	BusinessID   string `json:"business_id"`
	BusinessName string `json:"business_name"`
	ManagerID    string `json:"manager_id"`
	Message      string `json:"message"`
}

func CreateBusinessRequestHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterBusinessRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Validation failed",
				"errors":  map[string]string{"form": err.Error()},
			})
			return
		}

		req.normalize()

		if err := validateRegisterBusinessRequest(&req); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		result, err := repoadmin.CreateBusinessRepository(pool, repoadmin.CreateBusinessInput{
			Name:               req.Name,
			BusinessEmail:      req.BusinessEmail,
			BusinessPhone:      req.BusinessPhone,
			RegistrationNumber: req.RegistrationNumber,
			Industry:           req.Industry,
			OwnerName:          req.OwnerName,
			SubscriptionPlan:   req.SubscriptionPlan,
			ExistingManagerID:  req.ManagerID,
			Manager:            toCreateBusinessManagerInput(req.Manager),
		})
		if err != nil {
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
			case errors.Is(err, repoadmin.ErrInvalidManagerInput), errors.Is(err, repoadmin.ErrInvalidBusinessInput):
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Validation failed",
					"errors":  map[string]string{"form": err.Error()},
				})
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

func (r *RegisterBusinessRequest) normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.BusinessEmail = strings.ToLower(strings.TrimSpace(r.BusinessEmail))
	r.BusinessPhone = strings.TrimSpace(r.BusinessPhone)
	r.RegistrationNumber = strings.TrimSpace(r.RegistrationNumber)
	r.Industry = strings.TrimSpace(r.Industry)
	r.OwnerName = strings.TrimSpace(r.OwnerName)
	r.SubscriptionPlan = strings.TrimSpace(r.SubscriptionPlan)
	r.ManagerID = strings.TrimSpace(r.ManagerID)

	if r.Manager != nil {
		r.Manager.Username = strings.TrimSpace(r.Manager.Username)
		r.Manager.Email = strings.ToLower(strings.TrimSpace(r.Manager.Email))
		r.Manager.Password = strings.TrimSpace(r.Manager.Password)
		r.Manager.FullName = strings.TrimSpace(r.Manager.FullName)
		r.Manager.Phone = strings.TrimSpace(r.Manager.Phone)
	}
}

func validateRegisterBusinessRequest(req *RegisterBusinessRequest) gin.H {
	errorsMap := map[string]string{}

	if req.Name == "" {
		errorsMap["name"] = "Name is required."
	}

	if req.BusinessEmail == "" {
		errorsMap["business_email"] = "Business email is required."
	} else if _, err := mail.ParseAddress(req.BusinessEmail); err != nil {
		errorsMap["business_email"] = "Enter a valid business email address."
	}

	if req.ManagerID != "" && req.Manager != nil {
		errorsMap["manager"] = "Provide either manager_id or manager details, not both."
	}

	if req.ManagerID == "" {
		if req.Manager == nil {
			errorsMap["manager"] = "Manager details or manager_id are required."
		} else {
			if req.Manager.Username == "" {
				errorsMap["manager.username"] = "Username is required."
			}
			if req.Manager.Email == "" {
				errorsMap["manager.email"] = "Email is required."
			} else if _, err := mail.ParseAddress(req.Manager.Email); err != nil {
				errorsMap["manager.email"] = "Enter a valid email address."
			}
			if req.Manager.Password == "" {
				errorsMap["manager.password"] = "Password is required."
			} else if len(req.Manager.Password) < 8 {
				errorsMap["manager.password"] = "Password must be at least 8 characters."
			}
			if req.Manager.FullName == "" {
				errorsMap["manager.full_name"] = "Full name is required."
			}
		}
	}

	if len(errorsMap) == 0 {
		return nil
	}

	return gin.H{
		"message": "Validation failed",
		"errors":  errorsMap,
	}
}

func toCreateBusinessManagerInput(req *RegisterBusinessManager) *repoadmin.CreateBusinessManagerInput {
	if req == nil {
		return nil
	}

	return &repoadmin.CreateBusinessManagerInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
	}
}
