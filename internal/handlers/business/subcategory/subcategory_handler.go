package subcategory

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
	reposubcategory "pos/internal/repository/business/subcategory"
)

type subCategoryPayload struct {
	Name             *string `json:"name"`
	SubCategoryCode  *string `json:"sub_category_code"`
	Description      *string `json:"description"`
	ParentCategoryID *string `json:"parent_category_id"`
	MetaTitle        *string `json:"meta_title"`
	MetaDescription  *string `json:"meta_description"`
	ImageURL         *string `json:"image_url"`
	Active           *bool   `json:"active"`
	Featured         *bool   `json:"featured"`
	SortOrder        *int    `json:"sort_order"`
}

type SubCategoryResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	SubCategoryCode string `json:"sub_category_code"`
	Message         string `json:"message"`
}

type DeleteSubCategoryResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type SubCategoryProductCounts struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
}

type SubCategoryMetadata struct {
	MetaTitle       string `json:"metaTitle"`
	MetaDescription string `json:"metaDescription"`
}

type SubCategoryListItemResponse struct {
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	SubCategoryCode    string                    `json:"subCategoryCode"`
	Description        string                    `json:"description"`
	Icon               string                    `json:"icon"`
	Slug               string                    `json:"slug"`
	ParentCategoryID   string                    `json:"parentCategoryId"`
	ParentCategoryName string                    `json:"parentCategoryName"`
	Level              int                       `json:"level"`
	ProductCount       int                       `json:"productCount"`
	Active             bool                      `json:"active"`
	Featured           bool                      `json:"featured"`
	SortOrder          int                       `json:"sortOrder"`
	CreatedAt          string                    `json:"createdAt"`
	UpdatedAt          string                    `json:"updatedAt"`
	CreatedBy          string                    `json:"createdBy"`
	ProductCounts      *SubCategoryProductCounts `json:"productCounts,omitempty"`
	Metadata           *SubCategoryMetadata      `json:"metadata,omitempty"`
}

type ListSubCategoryResponse struct {
	Categories []SubCategoryListItemResponse `json:"categories"`
	Message    string                        `json:"message"`
}

func ListSubCategoriesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list sub categories handler: auth lookup failed err=%v", err)
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

		items, err := reposubcategory.ListSubCategoriesRepository(pool, businessID)
		if err != nil {
			switch err {
			case reposubcategory.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list sub categories handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load sub-categories"})
			}
			return
		}

		response := make([]SubCategoryListItemResponse, 0, len(items))
		for _, item := range items {
			response = append(response, SubCategoryListItemResponse{
				ID:                 item.ID,
				Name:               item.Name,
				SubCategoryCode:    item.SubCategoryCode,
				Description:        item.Description,
				Icon:               "FolderTree",
				Slug:               slugifySubCategoryName(item.Name),
				ParentCategoryID:   item.ParentCategoryID,
				ParentCategoryName: item.ParentCategoryName,
				Level:              1,
				ProductCount:       0,
				Active:             item.Active,
				Featured:           item.Featured,
				SortOrder:          item.SortOrder,
				CreatedAt:          item.CreatedAt,
				UpdatedAt:          item.UpdatedAt,
				CreatedBy:          "Current User",
				ProductCounts: &SubCategoryProductCounts{
					Total:    0,
					Active:   0,
					Inactive: 0,
				},
				Metadata: &SubCategoryMetadata{
					MetaTitle:       item.MetaTitle,
					MetaDescription: item.MetaDescription,
				},
			})
		}

		c.JSON(http.StatusOK, ListSubCategoryResponse{
			Categories: response,
			Message:    "Sub-categories loaded successfully",
		})
	}
}

func CreateSubCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Validation failed."}))
			return
		}

		var payload subCategoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := subCategoryFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

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
		if businessID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		item, err := reposubcategory.CreateSubCategoryRepository(pool, reposubcategory.CreateSubCategoryInput{
			BusinessID:       businessID,
			ParentCategoryID: derefString(payload.ParentCategoryID),
			SubCategoryCode:  derefString(payload.SubCategoryCode),
			Name:             strings.TrimSpace(*payload.Name),
			Description:      derefString(payload.Description),
			MetaTitle:        derefString(payload.MetaTitle),
			MetaDescription:  derefString(payload.MetaDescription),
			ImageURL:         derefString(payload.ImageURL),
			Active:           boolValue(payload.Active, true),
			Featured:         boolValue(payload.Featured, false),
			SortOrder:        intValue(payload.SortOrder, 0),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposubcategory.ErrSubCategoryAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Sub-category already exists"})
			case errors.Is(err, reposubcategory.ErrInvalidSubCategoryInput), errors.Is(err, reposubcategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("create sub category handler: repository failed business_id=%s name=%q err=%v", businessID, derefString(payload.Name), err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create sub-category"})
			}
			return
		}

		c.JSON(http.StatusCreated, SubCategoryResponse{
			ID:              item.ID,
			Name:            item.Name,
			SubCategoryCode: item.SubCategoryCode,
			Message:         "Sub-category created successfully",
		})
	}
}

func UpdateSubCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Validation failed."}))
			return
		}

		var payload subCategoryPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := subCategoryFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

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
		subCategoryID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || subCategoryID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		item, err := reposubcategory.UpdateSubCategoryRepository(pool, reposubcategory.UpdateSubCategoryInput{
			ID:               subCategoryID,
			BusinessID:       businessID,
			ParentCategoryID: derefString(payload.ParentCategoryID),
			SubCategoryCode:  derefString(payload.SubCategoryCode),
			Name:             strings.TrimSpace(*payload.Name),
			Description:      derefString(payload.Description),
			MetaTitle:        derefString(payload.MetaTitle),
			MetaDescription:  derefString(payload.MetaDescription),
			ImageURL:         derefString(payload.ImageURL),
			Active:           boolValue(payload.Active, true),
			Featured:         boolValue(payload.Featured, false),
			SortOrder:        intValue(payload.SortOrder, 0),
		})
		if err != nil {
			switch {
			case errors.Is(err, reposubcategory.ErrSubCategoryAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Sub-category already exists"})
			case errors.Is(err, reposubcategory.ErrSubCategoryNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Sub-category not found"})
			case errors.Is(err, reposubcategory.ErrInvalidSubCategoryInput), errors.Is(err, reposubcategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("update sub category handler: repository failed business_id=%s id=%s err=%v", businessID, subCategoryID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update sub-category"})
			}
			return
		}

		c.JSON(http.StatusOK, SubCategoryResponse{
			ID:              item.ID,
			Name:            item.Name,
			SubCategoryCode: item.SubCategoryCode,
			Message:         "Sub-category updated successfully",
		})
	}
}

func DeleteSubCategoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
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
		subCategoryID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || subCategoryID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		if err := reposubcategory.DeleteSubCategoryRepository(pool, businessID, subCategoryID, user.ID); err != nil {
			switch {
			case errors.Is(err, reposubcategory.ErrSubCategoryNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Sub-category not found"})
			case errors.Is(err, reposubcategory.ErrInvalidSubCategoryInput), errors.Is(err, reposubcategory.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("delete sub category handler: repository failed business_id=%s id=%s err=%v", businessID, subCategoryID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete sub-category"})
			}
			return
		}

		c.JSON(http.StatusOK, DeleteSubCategoryResponse{
			ID:      subCategoryID,
			Message: "Sub-category deleted successfully",
		})
	}
}

func subCategoryFieldErrors(payload *subCategoryPayload) map[string]string {
	errs := map[string]string{}
	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Sub-category name is required."
	}
	if payload == nil || payload.ParentCategoryID == nil || strings.TrimSpace(*payload.ParentCategoryID) == "" {
		errs["parentCategoryId"] = "Parent category is required."
	}
	if payload != nil && payload.ImageURL != nil && strings.TrimSpace(*payload.ImageURL) != "" {
		if len(strings.TrimSpace(*payload.ImageURL)) == 0 {
			errs["imageUrl"] = "Image is invalid."
		}
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

func slugifySubCategoryName(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.Trim(slug, "-")
	return slug
}
