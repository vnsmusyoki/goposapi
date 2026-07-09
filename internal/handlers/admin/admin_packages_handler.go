package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	repoadmin "pos/internal/repository/admin"
)

type createPackagePayload struct {
	Name        *string  `json:"name"`
	Slug        *string  `json:"slug"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	Currency    *string  `json:"currency"`
	TrialDays   *int     `json:"trial_days"`
	MaxUsers    *int     `json:"max_users"`
	MaxBranches *int     `json:"max_branches"`
	MaxProducts *int     `json:"max_products"`
}

type CreatePackageResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func CreatePackageRequestHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Unable to read request body.",
			}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(packageFieldErrors(nil)))
			return
		}

		var payload createPackagePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"form": "Request body must be valid JSON.",
			}))
			return
		}

		if errs := packageFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		req := repoadmin.CreatePackageRequest{
			Name:        strings.TrimSpace(*payload.Name),
			Slug:        strings.TrimSpace(*payload.Slug),
			Description: strings.TrimSpace(*payload.Description),
			Price:       *payload.Price,
			Currency:    strings.TrimSpace(*payload.Currency),
			TrialDays:   *payload.TrialDays,
			MaxUsers:    *payload.MaxUsers,
			MaxBranches: *payload.MaxBranches,
			MaxProducts: *payload.MaxProducts,
		}

		pkg, err := repoadmin.CreatePackageRepository(pool, req)

		if err != nil {
			if errors.Is(err, repoadmin.ErrPackageAlreadyExists) {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Package name or slug already exists",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create package",
			})
			return
		}

		c.JSON(http.StatusCreated, CreatePackageResponse{
			ID:      strconv.Itoa(pkg.Id),
			Name:    pkg.Name,
			Message: "Package created successfully",
		})
	}
}

func packageFieldErrors(payload *createPackagePayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Name is required."
	}
	if payload == nil || payload.Slug == nil || strings.TrimSpace(*payload.Slug) == "" {
		errs["slug"] = "Slug is required."
	}
	if payload == nil || payload.Description == nil || strings.TrimSpace(*payload.Description) == "" {
		errs["description"] = "Description is required."
	}
	if payload == nil || payload.Price == nil {
		errs["price"] = "Price is required."
	}
	if payload == nil || payload.Currency == nil || strings.TrimSpace(*payload.Currency) == "" {
		errs["currency"] = "Currency is required."
	}
	if payload == nil || payload.TrialDays == nil {
		errs["trial_days"] = "Trial days is required."
	}
	if payload == nil || payload.MaxUsers == nil {
		errs["max_users"] = "Max users is required."
	}
	if payload == nil || payload.MaxBranches == nil {
		errs["max_branches"] = "Max branches is required."
	}
	if payload == nil || payload.MaxProducts == nil {
		errs["max_products"] = "Max products is required."
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
