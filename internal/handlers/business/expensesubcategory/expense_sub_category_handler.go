package expensesubcategory

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
	repoexpensesubcategory "pos/internal/repository/business/expensesubcategory"
)

type expenseSubCategoryPayload struct {
	ExpenseCategoryID *string `json:"expense_category_id"`
	Name              *string `json:"name"`
	Code              *string `json:"code"`
	Description       *string `json:"description"`
	Active            *bool   `json:"active"`
	SortOrder         *int    `json:"sort_order"`
}

type ExpenseSubCategoryResponse struct {
	ID                  string `json:"id"`
	BusinessID          string `json:"businessId"`
	ExpenseCategoryID   string `json:"expenseCategoryId"`
	ExpenseCategoryName string `json:"expenseCategoryName"`
	Name                string `json:"name"`
	Code                string `json:"code"`
	Description         string `json:"description"`
	Active              bool   `json:"active"`
	SortOrder           int    `json:"sortOrder"`
	AddedBy             string `json:"addedBy"`
	AddedAt             string `json:"addedAt"`
	Message             string `json:"message"`
}

type ExpenseSubCategoryListResponse struct {
	ExpenseSubCategories []ExpenseSubCategoryResponse `json:"expenseSubCategories"`
	Message              string                       `json:"message"`
}

func ListExpenseSubCategoriesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list expense sub categories handler: auth lookup failed err=%v", err)
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

		items, err := repoexpensesubcategory.ListExpenseSubCategoriesRepository(pool, businessID)
		if err != nil {
			switch {
			case errors.Is(err, repoexpensesubcategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list expense sub categories handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load expense sub categories"})
			}
			return
		}

		response := make([]ExpenseSubCategoryResponse, 0, len(items))
		for _, item := range items {
			response = append(response, ExpenseSubCategoryResponse{
				ID:                  item.ID,
				BusinessID:          item.BusinessID,
				ExpenseCategoryID:   item.ExpenseCategoryID,
				ExpenseCategoryName: item.ExpenseCategoryName,
				Name:                item.Name,
				Code:                item.Code,
				Description:         item.Description,
				Active:              item.Active,
				SortOrder:           item.SortOrder,
				AddedBy:             displayAddedBy(item.CreatedBy),
				AddedAt:             item.AddedAt,
			})
		}

		c.JSON(http.StatusOK, ExpenseSubCategoryListResponse{
			ExpenseSubCategories: response,
			Message:              "Expense sub categories loaded successfully",
		})
	}
}

func CreateExpenseSubCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create expense sub category handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(expenseSubCategoryFieldErrors(nil)))
			return
		}

		var payload expenseSubCategoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}
		if errs := expenseSubCategoryFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		item, err := repoexpensesubcategory.CreateExpenseSubCategoryRepository(pool, repoexpensesubcategory.CreateExpenseSubCategoryInput{
			BusinessID:        businessID,
			ExpenseCategoryID: derefString(payload.ExpenseCategoryID),
			Name:              strings.TrimSpace(*payload.Name),
			Code:              derefString(payload.Code),
			Description:       derefString(payload.Description),
			Active:            boolValue(payload.Active, true),
			SortOrder:         intValue(payload.SortOrder, 0),
			CreatedByID:       user.ID,
			CreatedBy:         strings.TrimSpace(user.FullName),
			UpdatedByID:       user.ID,
			UpdatedBy:         strings.TrimSpace(user.FullName),
		})
		if err != nil {
			switch {
			case errors.Is(err, repoexpensesubcategory.ErrExpenseSubCategoryAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Expense sub category already exists"})
			case errors.Is(err, repoexpensesubcategory.ErrInvalidExpenseSubCategoryInput), errors.Is(err, repoexpensesubcategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("create expense sub category handler: repository failed business_id=%s name=%q err=%v", businessID, strings.TrimSpace(*payload.Name), err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create expense sub category"})
			}
			return
		}

		c.JSON(http.StatusCreated, ExpenseSubCategoryResponse{
			ID:                  item.ID,
			BusinessID:          item.BusinessID,
			ExpenseCategoryID:   item.ExpenseCategoryID,
			ExpenseCategoryName: item.ExpenseCategoryName,
			Name:                item.Name,
			Code:                item.Code,
			Description:         item.Description,
			Active:              item.Active,
			SortOrder:           item.SortOrder,
			AddedBy:             displayAddedBy(item.CreatedBy),
			AddedAt:             item.AddedAt,
			Message:             "Expense sub category created successfully",
		})
	}
}

func UpdateExpenseSubCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update expense sub category handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		expenseSubCategoryID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || expenseSubCategoryID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(expenseSubCategoryFieldErrors(nil)))
			return
		}

		var payload expenseSubCategoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}
		if errs := expenseSubCategoryFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		item, err := repoexpensesubcategory.UpdateExpenseSubCategoryRepository(pool, repoexpensesubcategory.UpdateExpenseSubCategoryInput{
			ID:                expenseSubCategoryID,
			BusinessID:        businessID,
			ExpenseCategoryID: derefString(payload.ExpenseCategoryID),
			Name:              strings.TrimSpace(*payload.Name),
			Code:              derefString(payload.Code),
			Description:       derefString(payload.Description),
			Active:            boolValue(payload.Active, true),
			SortOrder:         intValue(payload.SortOrder, 0),
			UpdatedByID:       user.ID,
			UpdatedBy:         strings.TrimSpace(user.FullName),
		})
		if err != nil {
			switch {
			case errors.Is(err, repoexpensesubcategory.ErrExpenseSubCategoryAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Expense sub category already exists"})
			case errors.Is(err, repoexpensesubcategory.ErrExpenseSubCategoryNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Expense sub category not found"})
			case errors.Is(err, repoexpensesubcategory.ErrInvalidExpenseSubCategoryInput), errors.Is(err, repoexpensesubcategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("update expense sub category handler: repository failed business_id=%s id=%s err=%v", businessID, expenseSubCategoryID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update expense sub category"})
			}
			return
		}

		c.JSON(http.StatusOK, ExpenseSubCategoryResponse{
			ID:                  item.ID,
			BusinessID:          item.BusinessID,
			ExpenseCategoryID:   item.ExpenseCategoryID,
			ExpenseCategoryName: item.ExpenseCategoryName,
			Name:                item.Name,
			Code:                item.Code,
			Description:         item.Description,
			Active:              item.Active,
			SortOrder:           item.SortOrder,
			AddedBy:             displayAddedBy(item.CreatedBy),
			AddedAt:             item.AddedAt,
			Message:             "Expense sub category updated successfully",
		})
	}
}

func DeleteExpenseSubCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete expense sub category handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		expenseSubCategoryID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || expenseSubCategoryID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		if err := repoexpensesubcategory.DeleteExpenseSubCategoryRepository(pool, businessID, expenseSubCategoryID); err != nil {
			switch {
			case errors.Is(err, repoexpensesubcategory.ErrExpenseSubCategoryNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Expense sub category not found"})
			case errors.Is(err, repoexpensesubcategory.ErrInvalidExpenseSubCategoryInput), errors.Is(err, repoexpensesubcategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("delete expense sub category handler: repository failed business_id=%s id=%s err=%v", businessID, expenseSubCategoryID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete expense sub category"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":      expenseSubCategoryID,
			"message": "Expense sub category deleted successfully",
		})
	}
}

func expenseSubCategoryFieldErrors(payload *expenseSubCategoryPayload) map[string]string {
	errs := map[string]string{}
	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Expense sub category name is required."
	}
	if payload == nil || payload.ExpenseCategoryID == nil || strings.TrimSpace(*payload.ExpenseCategoryID) == "" {
		errs["expenseCategoryId"] = "Expense category is required."
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
