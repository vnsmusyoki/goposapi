package warranty

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
	repowarranty "pos/internal/repository/business/warranty"
)

type warrantyPayload struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	DurationValue *int    `json:"duration_value"`
	DurationUnit  *string `json:"duration_unit"`
}

type WarrantyResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	DurationValue int    `json:"durationValue"`
	DurationUnit  string `json:"durationUnit"`
	AddedBy       string `json:"addedBy"`
	AddedAt       string `json:"addedAt"`
	Message       string `json:"message"`
}

type DeleteWarrantyResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type WarrantyListItemResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	DurationValue int    `json:"durationValue"`
	DurationUnit  string `json:"durationUnit"`
	AddedBy       string `json:"addedBy"`
	AddedAt       string `json:"addedAt"`
}

type ListWarrantiesResponse struct {
	Warranties []WarrantyListItemResponse `json:"warranties"`
	Message    string                     `json:"message"`
}

func ListWarrantiesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list warranties handler: auth lookup failed err=%v", err)
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

		warranties, err := repowarranty.ListWarrantiesRepository(pool, businessID)
		if err != nil {
			switch {
			case errors.Is(err, repowarranty.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list warranties handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load warranties"})
			}
			return
		}

		items := make([]WarrantyListItemResponse, 0, len(warranties))
		for _, warranty := range warranties {
			items = append(items, WarrantyListItemResponse{
				ID:            warranty.ID,
				Name:          warranty.Name,
				Description:   warranty.Description,
				DurationValue: warranty.DurationValue,
				DurationUnit:  warranty.DurationUnit,
				AddedBy:       displayAddedBy(warranty.AddedBy),
				AddedAt:       warranty.AddedAt,
			})
		}

		c.JSON(http.StatusOK, ListWarrantiesResponse{
			Warranties: items,
			Message:    "Warranties loaded successfully",
		})
	}
}

func CreateWarrantyRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create warranty handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(warrantyFieldErrors(nil)))
			return
		}

		var payload warrantyPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}
		if errs := warrantyFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		warranty, err := repowarranty.CreateWarrantyRepository(pool, repowarranty.CreateWarrantyInput{
			BusinessID:    businessID,
			Name:          strings.TrimSpace(*payload.Name),
			Description:   derefString(payload.Description),
			DurationValue: intValue(payload.DurationValue, 0),
			DurationUnit:  derefString(payload.DurationUnit),
			AddedByID:     user.ID,
			AddedBy:       strings.TrimSpace(user.FullName),
		})
		if err != nil {
			switch {
			case errors.Is(err, repowarranty.ErrWarrantyAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Warranty already exists"})
			case errors.Is(err, repowarranty.ErrInvalidWarrantyInput), errors.Is(err, repowarranty.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("create warranty handler: repository failed business_id=%s name=%q err=%v", businessID, strings.TrimSpace(*payload.Name), err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create warranty"})
			}
			return
		}

		c.JSON(http.StatusCreated, WarrantyResponse{
			ID:            warranty.ID,
			Name:          warranty.Name,
			Description:   warranty.Description,
			DurationValue: warranty.DurationValue,
			DurationUnit:  warranty.DurationUnit,
			AddedBy:       displayAddedBy(warranty.AddedBy),
			AddedAt:       warranty.AddedAt,
			Message:       "Warranty created successfully",
		})
	}
}

func UpdateWarrantyRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update warranty handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		warrantyID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || warrantyID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(warrantyFieldErrors(nil)))
			return
		}

		var payload warrantyPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}
		if errs := warrantyFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		warranty, err := repowarranty.UpdateWarrantyRepository(pool, repowarranty.UpdateWarrantyInput{
			ID:            warrantyID,
			BusinessID:    businessID,
			Name:          strings.TrimSpace(*payload.Name),
			Description:   derefString(payload.Description),
			DurationValue: intValue(payload.DurationValue, 0),
			DurationUnit:  derefString(payload.DurationUnit),
		})
		if err != nil {
			switch {
			case errors.Is(err, repowarranty.ErrWarrantyAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Warranty already exists"})
			case errors.Is(err, repowarranty.ErrWarrantyNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Warranty not found"})
			case errors.Is(err, repowarranty.ErrInvalidWarrantyInput), errors.Is(err, repowarranty.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("update warranty handler: repository failed business_id=%s id=%s err=%v", businessID, warrantyID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update warranty"})
			}
			return
		}

		c.JSON(http.StatusOK, WarrantyResponse{
			ID:            warranty.ID,
			Name:          warranty.Name,
			Description:   warranty.Description,
			DurationValue: warranty.DurationValue,
			DurationUnit:  warranty.DurationUnit,
			AddedBy:       displayAddedBy(warranty.AddedBy),
			AddedAt:       warranty.AddedAt,
			Message:       "Warranty updated successfully",
		})
	}
}

func DeleteWarrantyRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete warranty handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		warrantyID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || warrantyID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		if err := repowarranty.DeleteWarrantyRepository(pool, businessID, warrantyID, user.ID); err != nil {
			switch {
			case errors.Is(err, repowarranty.ErrWarrantyNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Warranty not found"})
			case errors.Is(err, repowarranty.ErrInvalidWarrantyInput), errors.Is(err, repowarranty.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("delete warranty handler: repository failed business_id=%s id=%s err=%v", businessID, warrantyID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete warranty"})
			}
			return
		}

		c.JSON(http.StatusOK, DeleteWarrantyResponse{
			ID:      warrantyID,
			Message: "Warranty deleted successfully",
		})
	}
}

func warrantyFieldErrors(payload *warrantyPayload) map[string]string {
	errs := map[string]string{}
	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Warranty name is required."
	}
	if payload == nil || payload.DurationValue == nil || *payload.DurationValue < 0 {
		errs["durationValue"] = "Duration must be zero or greater."
	}
	if payload == nil || payload.DurationUnit == nil || strings.TrimSpace(*payload.DurationUnit) == "" {
		errs["durationUnit"] = "Duration unit is required."
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
