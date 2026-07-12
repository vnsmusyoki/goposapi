package category

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repocategory "pos/internal/repository/business/category"
)

type createCategoryPayload struct {
	Name            *string `json:"name"`
	CategoryCode    *string `json:"category_code"`
	Description     *string `json:"description"`
	MetaTitle       *string `json:"meta_title"`
	MetaDescription *string `json:"meta_description"`
	ImageURL        *string `json:"image_url"`
	Active          *bool   `json:"active"`
	Featured        *bool   `json:"featured"`
	SortOrder       *int    `json:"sort_order"`
}

type CreateCategoryResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CategoryCode string `json:"category_code"`
	Active       bool   `json:"active"`
	Featured     bool   `json:"featured"`
	SortOrder    int    `json:"sort_order"`
	Message      string `json:"message"`
}

type UpdateCategoryResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CategoryCode string `json:"category_code"`
	Active       bool   `json:"active"`
	Featured     bool   `json:"featured"`
	SortOrder    int    `json:"sort_order"`
	Message      string `json:"message"`
}

type DeleteCategoryResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func CreateCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("create category handler: request started remote_ip=%s content_length=%d", c.ClientIP(), c.Request.ContentLength)

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("create category handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Unable to read request body.",
			}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			log.Printf("create category handler: empty request body")
			c.JSON(http.StatusBadRequest, validationFailed(categoryFieldErrors(nil)))
			return
		}

		var payload createCategoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create category handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		if errs := categoryFieldErrors(&payload); len(errs) > 0 {
			log.Printf("create category handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create category handler: auth lookup failed err=%v", err)
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
			log.Printf("create category handler: business not resolved user_id=%s", user.ID)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		req := repocategory.CreateCategoryInput{
			BusinessID:      businessID,
			Name:            strings.TrimSpace(*payload.Name),
			CategoryCode:    derefString(payload.CategoryCode),
			Description:     derefString(payload.Description),
			MetaTitle:       derefString(payload.MetaTitle),
			MetaDescription: derefString(payload.MetaDescription),
			ImageURL:        derefString(payload.ImageURL),
			Active:          boolValue(payload.Active, true),
			Featured:        boolValue(payload.Featured, false),
			SortOrder:       intValue(payload.SortOrder, 0),
		}

		category, err := repocategory.CreateCategoryRepository(pool, req)
		if err != nil {
			switch {
			case errors.Is(err, repocategory.ErrCategoryAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{
					"message": "Category already exists",
				})
			case errors.Is(err, repocategory.ErrInvalidCategoryInput), errors.Is(err, repocategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"form": err.Error(),
				}))
			default:
				log.Printf("create category handler: repository failed business_id=%s name=%q err=%v", businessID, req.Name, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to create category",
				})
			}
			return
		}

		log.Printf("create category handler: success id=%s business_id=%s code=%q name=%q", category.ID, category.BusinessID, category.CategoryCode, category.Name)
		c.JSON(http.StatusCreated, CreateCategoryResponse{
			ID:           category.ID,
			Name:         category.Name,
			CategoryCode: category.CategoryCode,
			Active:       category.Active,
			Featured:     category.Featured,
			SortOrder:    category.SortOrder,
			Message:      "Category created successfully",
		})
	}
}

func UpdateCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("update category handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Unable to read request body.",
			}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(categoryFieldErrors(nil)))
			return
		}

		var payload createCategoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update category handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		if errs := categoryFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update category handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update category handler: auth lookup failed err=%v", err)
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
		categoryID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || categoryID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		category, err := repocategory.UpdateCategoryRepository(pool, repocategory.UpdateCategoryInput{
			ID:              categoryID,
			BusinessID:      businessID,
			CategoryCode:    derefString(payload.CategoryCode),
			Name:            strings.TrimSpace(*payload.Name),
			Description:     derefString(payload.Description),
			MetaTitle:       derefString(payload.MetaTitle),
			MetaDescription: derefString(payload.MetaDescription),
			ImageURL:        derefString(payload.ImageURL),
			Active:          boolValue(payload.Active, true),
			Featured:        boolValue(payload.Featured, false),
			SortOrder:       intValue(payload.SortOrder, 0),
		})
		if err != nil {
			switch {
			case errors.Is(err, repocategory.ErrCategoryAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{
					"message": "Category already exists",
				})
			case errors.Is(err, repocategory.ErrInvalidCategoryInput), errors.Is(err, repocategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"form": err.Error(),
				}))
			default:
				log.Printf("update category handler: repository failed business_id=%s id=%s err=%v", businessID, categoryID, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to update category",
				})
			}
			return
		}

		c.JSON(http.StatusOK, UpdateCategoryResponse{
			ID:           category.ID,
			Name:         category.Name,
			CategoryCode: category.CategoryCode,
			Active:       category.Active,
			Featured:     category.Featured,
			SortOrder:    category.SortOrder,
			Message:      "Category updated successfully",
		})
	}
}

func DeleteCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete category handler: auth lookup failed err=%v", err)
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
		categoryID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || categoryID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		subCategoryCount, err := repocategory.DeleteCategoryRepository(pool, businessID, categoryID, user.ID)
		if err != nil {
			switch {
			case errors.Is(err, repocategory.ErrCategoryHasSubCategories):
				c.JSON(http.StatusConflict, gin.H{
					"message": fmt.Sprintf("Unable to delete category because it is linked to %d subcategories.", subCategoryCount),
				})
			case errors.Is(err, repocategory.ErrCategoryNotFound):
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Category not found",
				})
			case errors.Is(err, repocategory.ErrInvalidCategoryInput), errors.Is(err, repocategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"form": err.Error(),
				}))
			default:
				log.Printf("delete category handler: repository failed business_id=%s id=%s err=%v", businessID, categoryID, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to delete category",
				})
			}
			return
		}

		c.JSON(http.StatusOK, DeleteCategoryResponse{
			ID:      categoryID,
			Message: "Category deleted successfully",
		})
	}
}

func categoryFieldErrors(payload *createCategoryPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Category name is required."
	}

	if payload != nil && payload.ImageURL != nil && strings.TrimSpace(*payload.ImageURL) != "" {
		if err := repocategory.ValidateCategoryImageDataURL(*payload.ImageURL); err != nil {
			errs["image_url"] = repocategory.CategoryImageValidationMessage(err)
		}
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
