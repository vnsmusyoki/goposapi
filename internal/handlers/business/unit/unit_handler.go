package unit

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
	"pos/internal/models"
	repounit "pos/internal/repository/business/unit"
)

type businessUnitPayload struct {
	Name              *string  `json:"name"`
	ShortName         *string  `json:"shortName"`
	AllowDecimal      *bool    `json:"allowDecimal"`
	IsMultipleOfOther *bool    `json:"isMultipleOfOther"`
	BaseUnitID        *string  `json:"baseUnitId"`
	ConversionRate    *float64 `json:"conversionRate"`
}

type BusinessUnitResponse struct {
	models.BusinessUnit
	Message string `json:"message,omitempty"`
}

type BusinessUnitsResponse struct {
	Units   []models.BusinessUnit `json:"units"`
	Message string                `json:"message"`
}

type BusinessUnitDeleteResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func GetBusinessUnitsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business units handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		units, err := repounit.ListBusinessUnitsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, repounit.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
				return
			}

			log.Printf("get business units handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load business units"})
			return
		}

		c.JSON(http.StatusOK, BusinessUnitsResponse{
			Units:   units,
			Message: "Business units loaded successfully",
		})
	}
}

func CreateBusinessUnitRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create business unit handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("create business unit handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(businessUnitFieldErrors(nil)))
			return
		}

		var payload businessUnitPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create business unit handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := businessUnitFieldErrors(&payload); len(errs) > 0 {
			log.Printf("create business unit handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		unit, err := repounit.CreateBusinessUnitRepository(pool, repounit.CreateBusinessUnitInput{
			BusinessID:        businessID,
			Name:              strings.TrimSpace(*payload.Name),
			ShortName:         strings.TrimSpace(*payload.ShortName),
			AllowDecimal:      boolValue(payload.AllowDecimal),
			IsMultipleOfOther: boolValue(payload.IsMultipleOfOther),
			BaseUnitID:        stringValue(payload.BaseUnitID),
			ConversionRate:    floatValue(payload.ConversionRate),
			CreatedByUserID:   user.ID,
			CreatedBy:         strings.TrimSpace(user.FullName),
		})
		if err != nil {
			switch {
			case errors.Is(err, repounit.ErrBusinessUnitAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Business unit already exists"})
			case errors.Is(err, repounit.ErrInvalidBusinessUnitInput), errors.Is(err, repounit.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(businessUnitFieldErrors(&payload)))
			default:
				log.Printf("create business unit handler: repository failed business_id=%s name=%q err=%v", businessID, strings.TrimSpace(*payload.Name), err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create business unit"})
			}
			return
		}

		c.JSON(http.StatusCreated, BusinessUnitResponse{
			BusinessUnit: *unit,
			Message:      "Business unit created successfully",
		})
	}
}

func UpdateBusinessUnitRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business unit handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		unitID := strings.TrimSpace(c.Param("id"))
		if unitID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"unit_id": "Unit id is required.",
			}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("update business unit handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(businessUnitFieldErrors(nil)))
			return
		}

		var payload businessUnitPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update business unit handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := businessUnitFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update business unit handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		unit, err := repounit.UpdateBusinessUnitRepository(pool, repounit.UpdateBusinessUnitInput{
			BusinessID:        businessID,
			ID:                unitID,
			Name:              strings.TrimSpace(*payload.Name),
			ShortName:         strings.TrimSpace(*payload.ShortName),
			AllowDecimal:      boolValue(payload.AllowDecimal),
			IsMultipleOfOther: boolValue(payload.IsMultipleOfOther),
			BaseUnitID:        stringValue(payload.BaseUnitID),
			ConversionRate:    floatValue(payload.ConversionRate),
		})
		if err != nil {
			switch {
			case errors.Is(err, repounit.ErrBusinessUnitAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Business unit already exists"})
			case errors.Is(err, repounit.ErrBusinessUnitNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Business unit not found"})
			case errors.Is(err, repounit.ErrInvalidBusinessUnitInput), errors.Is(err, repounit.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(businessUnitFieldErrors(&payload)))
			default:
				log.Printf("update business unit handler: repository failed business_id=%s unit_id=%s err=%v", businessID, unitID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update business unit"})
			}
			return
		}

		c.JSON(http.StatusOK, BusinessUnitResponse{
			BusinessUnit: *unit,
			Message:      "Business unit updated successfully",
		})
	}
}

func DeleteBusinessUnitRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete business unit handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		unitID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || unitID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		if err := repounit.DeleteBusinessUnitRepository(pool, businessID, unitID); err != nil {
			switch {
			case errors.Is(err, repounit.ErrBusinessUnitNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Business unit not found"})
			case errors.Is(err, repounit.ErrInvalidBusinessUnitInput), errors.Is(err, repounit.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			default:
				log.Printf("delete business unit handler: repository failed business_id=%s unit_id=%s err=%v", businessID, unitID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete business unit"})
			}
			return
		}

		c.JSON(http.StatusOK, BusinessUnitDeleteResponse{
			ID:      unitID,
			Message: "Business unit deleted successfully",
		})
	}
}

func businessUnitFieldErrors(payload *businessUnitPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Unit name is required."
	}

	if payload == nil || payload.ShortName == nil || strings.TrimSpace(*payload.ShortName) == "" {
		errs["shortName"] = "Short name is required."
	}

	if payload != nil && payload.ConversionRate != nil && *payload.ConversionRate < 0 {
		errs["conversionRate"] = "Conversion rate cannot be negative."
	}

	multiple := payload != nil && payload.IsMultipleOfOther != nil && *payload.IsMultipleOfOther
	if multiple {
		if payload == nil || payload.BaseUnitID == nil || strings.TrimSpace(*payload.BaseUnitID) == "" {
			errs["baseUnitId"] = "Base unit is required."
		}

		if payload == nil || payload.ConversionRate == nil || *payload.ConversionRate <= 0 {
			errs["conversionRate"] = "Conversion rate must be greater than zero."
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func floatValue(value *float64) float64 {
	if value == nil || *value < 0 {
		return 0
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
