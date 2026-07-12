package category

import (
	"encoding/csv"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repocategory "pos/internal/repository/business/category"
)

type CategoryListItemResponse struct {
	ID            string                     `json:"id"`
	Name          string                     `json:"name"`
	CategoryCode  string                     `json:"categoryCode"`
	Description   string                     `json:"description"`
	Icon          string                     `json:"icon"`
	Slug          string                     `json:"slug"`
	ParentID      *string                    `json:"parentId"`
	Level         int                        `json:"level"`
	ProductCount  int                        `json:"productCount"`
	Active        bool                       `json:"active"`
	Featured      bool                       `json:"featured"`
	SortOrder     int                        `json:"sortOrder"`
	CreatedAt     string                     `json:"createdAt"`
	UpdatedAt     string                     `json:"updatedAt"`
	CreatedBy     string                     `json:"createdBy"`
	SubCategories []CategoryListItemResponse `json:"subCategories"`
}

type ListCategoryResponse struct {
	Categories []CategoryListItemResponse `json:"categories"`
	Message    string                     `json:"message"`
}

func ListCategoriesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list categories handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Active business is required.",
			})
			return
		}

		categories, err := repocategory.ListCategoriesRepository(pool, businessID)
		if err != nil {
			switch err {
			case repocategory.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Active business is required.",
				})
			default:
				log.Printf("list categories handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to load categories",
				})
			}
			return
		}

		responseCategories := make([]CategoryListItemResponse, 0, len(categories))
		for _, category := range categories {
			slug := slugifyCategoryName(category.Name)
			responseCategories = append(responseCategories, CategoryListItemResponse{
				ID:            category.ID,
				Name:          category.Name,
				CategoryCode:  category.CategoryCode,
				Description:   category.Description,
				Icon:          "FolderTree",
				Slug:          slug,
				ParentID:      nil,
				Level:         0,
				ProductCount:  0,
				Active:        category.Active,
				Featured:      category.Featured,
				SortOrder:     category.SortOrder,
				CreatedAt:     category.CreatedAt,
				UpdatedAt:     category.UpdatedAt,
				CreatedBy:     "Current User",
				SubCategories: []CategoryListItemResponse{},
			})
		}

		c.JSON(http.StatusOK, ListCategoryResponse{
			Categories: responseCategories,
			Message:    "Categories loaded successfully",
		})
	}
}

func ExportCategoriesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("export categories handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Active business is required.",
			})
			return
		}

		categories, err := repocategory.ListCategoriesRepository(pool, businessID)
		if err != nil {
			switch err {
			case repocategory.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "Active business is required.",
				})
			default:
				log.Printf("export categories handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to export categories",
				})
			}
			return
		}

		filename := "categories_" + time.Now().Format("2006-01-02") + ".csv"
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Cache-Control", "no-store")
		c.Status(http.StatusOK)

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		if err := writer.Write([]string{"ID", "Category Code", "Name", "Description", "Created At", "Updated At"}); err != nil {
			log.Printf("export categories handler: write header failed business_id=%s err=%v", businessID, err)
			return
		}

		for _, category := range categories {
			if err := writer.Write([]string{
				category.ID,
				category.CategoryCode,
				category.Name,
				category.Description,
				category.CreatedAt,
				category.UpdatedAt,
			}); err != nil {
				log.Printf("export categories handler: write row failed business_id=%s category_id=%s err=%v", businessID, category.ID, err)
				return
			}
		}
	}
}

func slugifyCategoryName(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.Trim(slug, "-")
	return slug
}
