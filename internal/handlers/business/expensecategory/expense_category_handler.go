package expensecategory

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
	repoexpensecategory "pos/internal/repository/business/expensecategory"
)

type expenseCategoryPayload struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Description *string `json:"description"`
	ParentID    *string `json:"parent_id"`
	Active      *bool   `json:"active"`
	SortOrder   *int    `json:"sort_order"`
}

type ExpenseCategoryResponse struct {
	ID          string `json:"id"`
	BusinessID  string `json:"businessId"`
	ParentID    string `json:"parentId"`
	ParentName  string `json:"parentName"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	SortOrder   int    `json:"sortOrder"`
	AddedBy     string `json:"addedBy"`
	AddedAt     string `json:"addedAt"`
	Message     string `json:"message"`
}

type ExpenseCategoryListResponse struct {
	ExpenseCategories []ExpenseCategoryResponse `json:"expenseCategories"`
	Message           string                    `json:"message"`
}

func ListExpenseCategoriesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list expense categories handler: auth lookup failed err=%v", err)
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

		items, err := repoexpensecategory.ListExpenseCategoriesRepository(pool, businessID)
		if err != nil {
			switch {
			case errors.Is(err, repoexpensecategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list expense categories handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load expense categories"})
			}
			return
		}

		response := make([]ExpenseCategoryResponse, 0, len(items))
		for _, item := range items {
			response = append(response, ExpenseCategoryResponse{
				ID:          item.ID,
				BusinessID:  item.BusinessID,
				ParentID:    item.ParentID,
				ParentName:  item.ParentName,
				Name:        item.Name,
				Code:        item.Code,
				Description: item.Description,
				Active:      item.Active,
				SortOrder:   item.SortOrder,
				AddedBy:     displayAddedBy(item.CreatedBy),
				AddedAt:     item.AddedAt,
			})
		}

		c.JSON(http.StatusOK, ExpenseCategoryListResponse{
			ExpenseCategories: response,
			Message:           "Expense categories loaded successfully",
		})
	}
}

func CreateExpenseCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create expense category handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(expenseCategoryFieldErrors(nil)))
			return
		}

		var payload expenseCategoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}
		if errs := expenseCategoryFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		item, err := repoexpensecategory.CreateExpenseCategoryRepository(pool, repoexpensecategory.CreateExpenseCategoryInput{
			BusinessID:  businessID,
			ParentID:    derefString(payload.ParentID),
			Name:        strings.TrimSpace(*payload.Name),
			Code:        derefString(payload.Code),
			Description: derefString(payload.Description),
			Active:      boolValue(payload.Active, true),
			SortOrder:   intValue(payload.SortOrder, 0),
			CreatedByID: user.ID,
			CreatedBy:   strings.TrimSpace(user.FullName),
			UpdatedByID: user.ID,
			UpdatedBy:   strings.TrimSpace(user.FullName),
		})
		if err != nil {
			switch {
			case errors.Is(err, repoexpensecategory.ErrExpenseCategoryAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Expense category already exists"})
			case errors.Is(err, repoexpensecategory.ErrInvalidExpenseCategoryInput), errors.Is(err, repoexpensecategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("create expense category handler: repository failed business_id=%s name=%q err=%v", businessID, strings.TrimSpace(*payload.Name), err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create expense category"})
			}
			return
		}

		c.JSON(http.StatusCreated, ExpenseCategoryResponse{
			ID:          item.ID,
			BusinessID:  item.BusinessID,
			ParentID:    item.ParentID,
			ParentName:  item.ParentName,
			Name:        item.Name,
			Code:        item.Code,
			Description: item.Description,
			Active:      item.Active,
			SortOrder:   item.SortOrder,
			AddedBy:     displayAddedBy(item.CreatedBy),
			AddedAt:     item.AddedAt,
			Message:     "Expense category created successfully",
		})
	}
}

func UpdateExpenseCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update expense category handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		expenseCategoryID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || expenseCategoryID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(expenseCategoryFieldErrors(nil)))
			return
		}

		var payload expenseCategoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}
		if errs := expenseCategoryFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		item, err := repoexpensecategory.UpdateExpenseCategoryRepository(pool, repoexpensecategory.UpdateExpenseCategoryInput{
			ID:          expenseCategoryID,
			BusinessID:  businessID,
			ParentID:    derefString(payload.ParentID),
			Name:        strings.TrimSpace(*payload.Name),
			Code:        derefString(payload.Code),
			Description: derefString(payload.Description),
			Active:      boolValue(payload.Active, true),
			SortOrder:   intValue(payload.SortOrder, 0),
			UpdatedByID: user.ID,
			UpdatedBy:   strings.TrimSpace(user.FullName),
		})
		if err != nil {
			switch {
			case errors.Is(err, repoexpensecategory.ErrExpenseCategoryAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Expense category already exists"})
			case errors.Is(err, repoexpensecategory.ErrExpenseCategoryNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Expense category not found"})
			case errors.Is(err, repoexpensecategory.ErrInvalidExpenseCategoryInput), errors.Is(err, repoexpensecategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("update expense category handler: repository failed business_id=%s id=%s err=%v", businessID, expenseCategoryID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update expense category"})
			}
			return
		}

		c.JSON(http.StatusOK, ExpenseCategoryResponse{
			ID:          item.ID,
			BusinessID:  item.BusinessID,
			ParentID:    item.ParentID,
			ParentName:  item.ParentName,
			Name:        item.Name,
			Code:        item.Code,
			Description: item.Description,
			Active:      item.Active,
			SortOrder:   item.SortOrder,
			AddedBy:     displayAddedBy(item.CreatedBy),
			AddedAt:     item.AddedAt,
			Message:     "Expense category updated successfully",
		})
	}
}

func DeleteExpenseCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete expense category handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		expenseCategoryID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || expenseCategoryID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		if err := repoexpensecategory.DeleteExpenseCategoryRepository(pool, businessID, expenseCategoryID); err != nil {
			switch {
			case errors.Is(err, repoexpensecategory.ErrExpenseCategoryNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Expense category not found"})
			case errors.Is(err, repoexpensecategory.ErrInvalidExpenseCategoryInput), errors.Is(err, repoexpensecategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("delete expense category handler: repository failed business_id=%s id=%s err=%v", businessID, expenseCategoryID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete expense category"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":      expenseCategoryID,
			"message": "Expense category deleted successfully",
		})
	}
}

func expenseCategoryFieldErrors(payload *expenseCategoryPayload) map[string]string {
	errs := map[string]string{}
	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Expense category name is required."
	}
	return errs
}

func validationFailed(errorsMap map[string]string) gin.H {
	if len(errorsMap) == 0 {
		errorsMap = map[string]string{"form": "Validation failed."}
	}
	return gin.H{"message": "Validation failed", "errors": errorsMap}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Code), "business") {
			return true
		}
	}
	return false
}

func displayAddedBy(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Current User"
	}
	return value
}
