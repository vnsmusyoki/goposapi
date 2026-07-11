package category

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
	Message      string `json:"message"`
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
			Message:      "Category created successfully",
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

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Code), "business") {
			return true
		}
	}
	return false
}
