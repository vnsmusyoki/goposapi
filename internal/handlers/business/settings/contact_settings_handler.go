package settings

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
	reposettings "pos/internal/repository/business/settings"
)

type updateBusinessContactSettingsPayload struct {
	DefaultCreditLimit *float64 `json:"defaultCreditLimit"`
}

type BusinessContactSettingsResponse struct {
	ID                 string   `json:"id"`
	DefaultCreditLimit *float64 `json:"defaultCreditLimit,omitempty"`
	Message            string   `json:"message"`
}

func GetBusinessContactSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get business contact settings handler: auth lookup failed err=%v", err)
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

		settings, err := reposettings.GetBusinessContactSettingsRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, reposettings.ErrBusinessNotResolved) {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
				return
			}

			log.Printf("get business contact settings handler: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load contact settings"})
			return
		}

		c.JSON(http.StatusOK, toBusinessContactSettingsResponse(settings, "Contact settings loaded successfully"))
	}
}

func UpdateBusinessContactSettingsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update business contact settings handler: auth lookup failed err=%v", err)
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
			log.Printf("update business contact settings handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(contactSettingsFieldErrors(nil)))
			return
		}

		var payload updateBusinessContactSettingsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update business contact settings handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := contactSettingsFieldErrors(&payload); len(errs) > 0 {
			log.Printf("update business contact settings handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		settings, err := reposettings.UpdateBusinessContactSettingsRepository(pool, reposettings.UpdateBusinessContactSettingsInput{
			BusinessID:         businessID,
			DefaultCreditLimit: *payload.DefaultCreditLimit,
		})
		if err != nil {
			switch {
			case errors.Is(err, reposettings.ErrInvalidBusinessSettingsInput):
				c.JSON(http.StatusBadRequest, validationFailed(contactSettingsFieldErrors(&payload)))
			default:
				log.Printf("update business contact settings handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save contact settings"})
			}
			return
		}

		c.JSON(http.StatusOK, toBusinessContactSettingsResponse(settings, "Contact settings saved successfully"))
	}
}

func contactSettingsFieldErrors(payload *updateBusinessContactSettingsPayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.DefaultCreditLimit == nil {
		errs["defaultCreditLimit"] = "Default credit limit is required."
	} else if *payload.DefaultCreditLimit < 0 {
		errs["defaultCreditLimit"] = "Default credit limit must be zero or more."
	}

	return errs
}

func toBusinessContactSettingsResponse(settings *models.BusinessContactSettings, message string) BusinessContactSettingsResponse {
	response := BusinessContactSettingsResponse{
		ID:      settings.ID,
		Message: message,
	}

	if settings.DefaultCreditLimit != nil {
		value := *settings.DefaultCreditLimit
		response.DefaultCreditLimit = &value
	}

	return response
}
