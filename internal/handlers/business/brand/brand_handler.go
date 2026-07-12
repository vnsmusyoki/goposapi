package brand

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
	repobrand "pos/internal/repository/business/brand"
)

type brandPayload struct {
	Name             *string `json:"name"`
	ShortDescription *string `json:"short_description"`
}

type BrandResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ShortDescription string `json:"shortDescription"`
	AddedBy          string `json:"addedBy"`
	AddedAt          string `json:"addedAt"`
	Message          string `json:"message"`
}

type DeleteBrandResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type BrandListItemResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ShortDescription string `json:"shortDescription"`
	AddedBy          string `json:"addedBy"`
	AddedAt          string `json:"addedAt"`
}

type ListBrandsResponse struct {
	Brands  []BrandListItemResponse `json:"brands"`
	Message string                  `json:"message"`
}

func ListBrandsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list brands handler: auth lookup failed err=%v", err)
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

		brands, err := repobrand.ListBrandsRepository(pool, businessID)
		if err != nil {
			switch {
			case errors.Is(err, repobrand.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list brands handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load brands"})
			}
			return
		}

		items := make([]BrandListItemResponse, 0, len(brands))
		for _, brand := range brands {
			items = append(items, BrandListItemResponse{
				ID:               brand.ID,
				Name:             brand.Name,
				ShortDescription: brand.ShortDescription,
				AddedBy:          displayAddedBy(brand.AddedBy),
				AddedAt:          brand.AddedAt,
			})
		}

		c.JSON(http.StatusOK, ListBrandsResponse{
			Brands:  items,
			Message: "Brands loaded successfully",
		})
	}
}

func CreateBrandRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create brand handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(brandFieldErrors(nil)))
			return
		}

		var payload brandPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}
		if errs := brandFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		brand, err := repobrand.CreateBrandRepository(pool, repobrand.CreateBrandInput{
			BusinessID:       businessID,
			Name:             strings.TrimSpace(*payload.Name),
			ShortDescription: derefString(payload.ShortDescription),
			AddedByID:        user.ID,
			AddedBy:          strings.TrimSpace(user.FullName),
		})
		if err != nil {
			switch {
			case errors.Is(err, repobrand.ErrBrandAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Brand already exists"})
			case errors.Is(err, repobrand.ErrInvalidBrandInput), errors.Is(err, repobrand.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("create brand handler: repository failed business_id=%s name=%q err=%v", businessID, strings.TrimSpace(*payload.Name), err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create brand"})
			}
			return
		}

		c.JSON(http.StatusCreated, BrandResponse{
			ID:               brand.ID,
			Name:             brand.Name,
			ShortDescription: brand.ShortDescription,
			AddedBy:          displayAddedBy(brand.AddedBy),
			AddedAt:          brand.AddedAt,
			Message:          "Brand created successfully",
		})
	}
}

func UpdateBrandRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update brand handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		brandID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || brandID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(brandFieldErrors(nil)))
			return
		}

		var payload brandPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}
		if errs := brandFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		brand, err := repobrand.UpdateBrandRepository(pool, repobrand.UpdateBrandInput{
			ID:               brandID,
			BusinessID:       businessID,
			Name:             strings.TrimSpace(*payload.Name),
			ShortDescription: derefString(payload.ShortDescription),
		})
		if err != nil {
			switch {
			case errors.Is(err, repobrand.ErrBrandAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Brand already exists"})
			case errors.Is(err, repobrand.ErrBrandNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Brand not found"})
			case errors.Is(err, repobrand.ErrInvalidBrandInput), errors.Is(err, repobrand.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("update brand handler: repository failed business_id=%s id=%s err=%v", businessID, brandID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update brand"})
			}
			return
		}

		c.JSON(http.StatusOK, BrandResponse{
			ID:               brand.ID,
			Name:             brand.Name,
			ShortDescription: brand.ShortDescription,
			AddedBy:          displayAddedBy(brand.AddedBy),
			AddedAt:          brand.AddedAt,
			Message:          "Brand updated successfully",
		})
	}
}

func DeleteBrandRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete brand handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		brandID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || brandID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		if err := repobrand.DeleteBrandRepository(pool, businessID, brandID, user.ID); err != nil {
			switch {
			case errors.Is(err, repobrand.ErrBrandNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Brand not found"})
			case errors.Is(err, repobrand.ErrInvalidBrandInput), errors.Is(err, repobrand.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("delete brand handler: repository failed business_id=%s id=%s err=%v", businessID, brandID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete brand"})
			}
			return
		}

		c.JSON(http.StatusOK, DeleteBrandResponse{
			ID:      brandID,
			Message: "Brand deleted successfully",
		})
	}
}

func brandFieldErrors(payload *brandPayload) map[string]string {
	errs := map[string]string{}
	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Brand name is required."
	}
	return errs
}

func displayAddedBy(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Current User"
	}
	return value
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
