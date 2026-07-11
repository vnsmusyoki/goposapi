package admin

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	repoadmin "pos/internal/repository/admin"
)

type createModulePayload struct {
	RoleID        *string `json:"role_id"`
	Code          *string `json:"code"`
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	Icon          *string `json:"icon"`
	Path          *string `json:"path"`
	HasSubModules *bool   `json:"has_sub_modules"`
	AccessLevel   *int    `json:"access_level"`
	SortOrder     *int    `json:"sort_order"`
	Active        *bool   `json:"active"`
}

type createSubmodulePayload struct {
	ModuleID    *string `json:"module_id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	URL         *string `json:"url"`
	AccessLevel *int    `json:"access_level"`
	SortOrder   *int    `json:"sort_order"`
	Active      *bool   `json:"active"`
}

type updateSubmodulePayload struct {
	ModuleID    *string `json:"module_id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	URL         *string `json:"url"`
	AccessLevel *int    `json:"access_level"`
	SortOrder   *int    `json:"sort_order"`
	Active      *bool   `json:"active"`
}

type reorderSubmodulesPayload struct {
	ModuleID            *string  `json:"module_id"`
	OrderedSubmoduleIDs []string `json:"ordered_submodule_ids"`
}

type reorderModulesPayload struct {
	RoleCode         *string  `json:"role_code"`
	OrderedModuleIDs []string `json:"ordered_module_ids"`
}

type CreateModuleResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type CreateSubmoduleResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type UpdateSubmoduleResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type ReorderSubmodulesResponse struct {
	Message string `json:"message"`
}

type ReorderModulesResponse struct {
	Message string `json:"message"`
}

func CreateModuleRequestHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(moduleFieldErrors(nil)))
			return
		}

		var payload createModulePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := moduleFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		module, err := repoadmin.CreateModuleRepository(pool, repoadmin.CreateModuleRequest{
			RoleID:        strings.TrimSpace(derefString(payload.RoleID)),
			Code:          strings.TrimSpace(derefString(payload.Code)),
			Name:          strings.TrimSpace(derefString(payload.Name)),
			Description:   strings.TrimSpace(derefString(payload.Description)),
			Icon:          strings.TrimSpace(derefString(payload.Icon)),
			Path:          strings.TrimSpace(derefString(payload.Path)),
			HasSubModules: derefBool(payload.HasSubModules, true),
			AccessLevel:   derefInt(payload.AccessLevel),
			SortOrder:     derefInt(payload.SortOrder),
			Active:        derefBool(payload.Active, true),
		})
		if err != nil {
			switch {
			case errors.Is(err, repoadmin.ErrModuleAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "Module code already exists"})
			case errors.Is(err, repoadmin.ErrInvalidBusinessInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Invalid module data."}))
			default:
				log.Printf("create module handler: repository failed err=%v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create module"})
			}
			return
		}

		c.JSON(http.StatusCreated, CreateModuleResponse{
			ID:      module.ID,
			Name:    module.Name,
			Message: "Module created successfully",
		})
	}
}

func CreateSubmoduleRequestHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(submoduleFieldErrors(nil)))
			return
		}

		var payload createSubmodulePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := submoduleFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		submodule, err := repoadmin.CreateSubmoduleRepository(pool, repoadmin.CreateSubmoduleRequest{
			ModuleID:    strings.TrimSpace(derefString(payload.ModuleID)),
			Name:        strings.TrimSpace(derefString(payload.Name)),
			Description: strings.TrimSpace(derefString(payload.Description)),
			Icon:        strings.TrimSpace(derefString(payload.Icon)),
			URL:         strings.TrimSpace(derefString(payload.URL)),
			AccessLevel: derefInt(payload.AccessLevel),
			SortOrder:   derefInt(payload.SortOrder),
			Active:      derefBool(payload.Active, true),
		})
		if err != nil {
			switch {
			case errors.Is(err, repoadmin.ErrSubmoduleAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "A submodule with that name already exists for this role"})
			case errors.Is(err, repoadmin.ErrInvalidBusinessInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Invalid submodule data."}))
			default:
				log.Printf("create submodule handler: repository failed err=%v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create submodule"})
			}
			return
		}

		c.JSON(http.StatusCreated, CreateSubmoduleResponse{
			ID:      submodule.ID,
			Name:    submodule.Name,
			Message: "Submodule created successfully",
		})
	}
}

func UpdateSubmoduleRequestHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		submoduleID := strings.TrimSpace(c.Param("id"))
		if submoduleID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Submodule id is required."}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(submoduleFieldErrors(nil)))
			return
		}

		var payload updateSubmodulePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := updateSubmoduleFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		submodule, err := repoadmin.UpdateSubmoduleRepository(pool, submoduleID, repoadmin.UpdateSubmoduleRequest{
			ModuleID:    strings.TrimSpace(derefString(payload.ModuleID)),
			Name:        strings.TrimSpace(derefString(payload.Name)),
			Description: strings.TrimSpace(derefString(payload.Description)),
			Icon:        strings.TrimSpace(derefString(payload.Icon)),
			URL:         strings.TrimSpace(derefString(payload.URL)),
			AccessLevel: derefInt(payload.AccessLevel),
			SortOrder:   derefInt(payload.SortOrder),
			Active:      derefBool(payload.Active, true),
		})
		if err != nil {
			switch {
			case errors.Is(err, repoadmin.ErrSubmoduleAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{"message": "A submodule with that name already exists for this role"})
			case errors.Is(err, repoadmin.ErrSubmoduleNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Submodule not found"})
			case errors.Is(err, repoadmin.ErrInvalidBusinessInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Invalid submodule data."}))
			default:
				log.Printf("update submodule handler: repository failed err=%v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update submodule"})
			}
			return
		}

		c.JSON(http.StatusOK, UpdateSubmoduleResponse{
			ID:      submodule.ID,
			Name:    submodule.Name,
			Message: "Submodule updated successfully",
		})
	}
}

func ReorderSubmodulesRequestHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body is required."}))
			return
		}

		var payload reorderSubmodulesPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		moduleID := strings.TrimSpace(derefString(payload.ModuleID))
		if moduleID == "" || len(payload.OrderedSubmoduleIDs) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Module and ordered submodules are required."}))
			return
		}

		cleanedIDs := make([]string, 0, len(payload.OrderedSubmoduleIDs))
		seen := make(map[string]struct{}, len(payload.OrderedSubmoduleIDs))
		for _, id := range payload.OrderedSubmoduleIDs {
			cleanedID := strings.TrimSpace(id)
			if cleanedID == "" {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"ordered_submodule_ids": "Submodule ids must be valid."}))
				return
			}
			if _, exists := seen[cleanedID]; exists {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"ordered_submodule_ids": "Submodule ids must be unique."}))
				return
			}
			seen[cleanedID] = struct{}{}
			cleanedIDs = append(cleanedIDs, cleanedID)
		}

		if err := repoadmin.ReorderSubmodulesRepository(pool, moduleID, cleanedIDs); err != nil {
			switch {
			case errors.Is(err, repoadmin.ErrModuleNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Module not found"})
			case errors.Is(err, repoadmin.ErrInvalidBusinessInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Invalid reorder payload."}))
			default:
				log.Printf("reorder submodules handler: repository failed err=%v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to reorder submodules"})
			}
			return
		}

		c.JSON(http.StatusOK, ReorderSubmodulesResponse{
			Message: "Submodules reordered successfully",
		})
	}
}

func ReorderModulesRequestHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body is required."}))
			return
		}

		var payload reorderModulesPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		roleCode := strings.TrimSpace(derefString(payload.RoleCode))
		if roleCode == "" || len(payload.OrderedModuleIDs) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Role and ordered modules are required."}))
			return
		}

		cleanedIDs := make([]string, 0, len(payload.OrderedModuleIDs))
		seen := make(map[string]struct{}, len(payload.OrderedModuleIDs))
		for _, id := range payload.OrderedModuleIDs {
			cleanedID := strings.TrimSpace(id)
			if cleanedID == "" {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"ordered_module_ids": "Module ids must be valid."}))
				return
			}
			if _, exists := seen[cleanedID]; exists {
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"ordered_module_ids": "Module ids must be unique."}))
				return
			}
			seen[cleanedID] = struct{}{}
			cleanedIDs = append(cleanedIDs, cleanedID)
		}

		if err := repoadmin.ReorderModulesRepository(pool, roleCode, cleanedIDs); err != nil {
			switch {
			case errors.Is(err, repoadmin.ErrRoleNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Role not found"})
			case errors.Is(err, repoadmin.ErrInvalidBusinessInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Invalid reorder payload."}))
			default:
				log.Printf("reorder modules handler: repository failed err=%v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to reorder modules"})
			}
			return
		}

		c.JSON(http.StatusOK, ReorderModulesResponse{
			Message: "Modules reordered successfully",
		})
	}
}

func moduleFieldErrors(payload *createModulePayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.RoleID == nil || strings.TrimSpace(*payload.RoleID) == "" {
		errs["role_id"] = "Role is required."
	}
	if payload == nil || payload.Code == nil || strings.TrimSpace(*payload.Code) == "" {
		errs["code"] = "Module code is required."
	}
	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Module name is required."
	}
	if payload == nil || payload.Path == nil || strings.TrimSpace(*payload.Path) == "" {
		errs["path"] = "Module path is required."
	}
	if payload == nil || payload.HasSubModules == nil {
		errs["has_sub_modules"] = "Please specify whether the module has sub modules."
	}
	if payload != nil && payload.AccessLevel != nil && *payload.AccessLevel != 1 && *payload.AccessLevel != 2 {
		errs["access_level"] = "Access level must be 1 or 2."
	}
	if payload != nil && payload.SortOrder != nil && *payload.SortOrder < 0 {
		errs["sort_order"] = "Sort order cannot be negative."
	}

	return errs
}

func submoduleFieldErrors(payload *createSubmodulePayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.ModuleID == nil || strings.TrimSpace(*payload.ModuleID) == "" {
		errs["module_id"] = "Parent module is required."
	}
	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Submodule name is required."
	}
	if payload == nil || payload.URL == nil || strings.TrimSpace(*payload.URL) == "" {
		errs["url"] = "Submodule URL is required."
	}
	if payload == nil || payload.Icon == nil || strings.TrimSpace(*payload.Icon) == "" {
		errs["icon"] = "Icon is required."
	}
	if payload != nil && payload.AccessLevel != nil && *payload.AccessLevel != 1 && *payload.AccessLevel != 2 {
		errs["access_level"] = "Access level must be 1 or 2."
	}
	if payload != nil && payload.SortOrder != nil && *payload.SortOrder < 0 {
		errs["sort_order"] = "Sort order cannot be negative."
	}

	return errs
}

func updateSubmoduleFieldErrors(payload *updateSubmodulePayload) map[string]string {
	errs := map[string]string{}

	if payload == nil || payload.ModuleID == nil || strings.TrimSpace(*payload.ModuleID) == "" {
		errs["module_id"] = "Parent module is required."
	}
	if payload == nil || payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Submodule name is required."
	}
	if payload == nil || payload.URL == nil || strings.TrimSpace(*payload.URL) == "" {
		errs["url"] = "Submodule URL is required."
	}
	if payload == nil || payload.Icon == nil || strings.TrimSpace(*payload.Icon) == "" {
		errs["icon"] = "Icon is required."
	}
	if payload != nil && payload.AccessLevel != nil && *payload.AccessLevel != 1 && *payload.AccessLevel != 2 {
		errs["access_level"] = "Access level must be 1 or 2."
	}
	if payload != nil && payload.SortOrder != nil && *payload.SortOrder < 0 {
		errs["sort_order"] = "Sort order cannot be negative."
	}

	return errs
}

func derefBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
