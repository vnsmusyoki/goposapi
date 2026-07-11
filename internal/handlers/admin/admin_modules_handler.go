package admin

import (
	"log"
	"net/http"
	"strings"

	"pos/internal/auth"
	repoadmin "pos/internal/repository/admin"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ModuleCatalogSubmoduleResponse struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	AccessLevel int    `json:"accessLevel"`
	SortOrder   int    `json:"sortOrder"`
	Active      bool   `json:"active"`
}

type ModuleCatalogModuleResponse struct {
	ID            string                           `json:"id"`
	Code          string                           `json:"code"`
	Name          string                           `json:"name"`
	Description   string                           `json:"description"`
	Icon          string                           `json:"icon"`
	Path          string                           `json:"path"`
	HasSubModules bool                             `json:"hasSubModules"`
	AccessLevel   int                              `json:"accessLevel"`
	RoleCode      string                           `json:"roleCode"`
	RoleName      string                           `json:"roleName"`
	SortOrder     int                              `json:"sortOrder"`
	Active        bool                             `json:"active"`
	Submodules    []ModuleCatalogSubmoduleResponse `json:"submodules"`
}

type ModuleCatalogTabResponse struct {
	Key     string                        `json:"key"`
	Label   string                        `json:"label"`
	Modules []ModuleCatalogModuleResponse `json:"modules"`
}

type ModuleCatalogResponse struct {
	Tabs    []ModuleCatalogTabResponse `json:"tabs"`
	Message string                     `json:"message"`
}

func ListModulesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list modules handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasAdminRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Admin access is required"})
			return
		}

		tabs, err := repoadmin.ListModulesRepository(pool)
		if err != nil {
			log.Printf("list modules handler: repository failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load modules"})
			return
		}

		responseTabs := make([]ModuleCatalogTabResponse, 0, len(tabs))
		for _, tab := range tabs {
			if len(tab.Modules) == 0 {
				continue
			}

			moduleResponses := make([]ModuleCatalogModuleResponse, 0, len(tab.Modules))
			for _, module := range tab.Modules {
				submoduleResponses := make([]ModuleCatalogSubmoduleResponse, 0, len(module.Submodules))
				for _, submodule := range module.Submodules {
					submoduleResponses = append(submoduleResponses, ModuleCatalogSubmoduleResponse{
						ID:          submodule.ID,
						Code:        submodule.Code,
						Name:        submodule.Name,
						URL:         submodule.URL,
						Icon:        submodule.Icon,
						Description: submodule.Description,
						AccessLevel: submodule.AccessLevel,
						SortOrder:   submodule.SortOrder,
						Active:      submodule.Active,
					})
				}

				moduleResponses = append(moduleResponses, ModuleCatalogModuleResponse{
					ID:            module.ID,
					Code:          module.Code,
					Name:          module.Name,
					Description:   module.Description,
					Icon:          module.Icon,
					Path:          module.Path,
					HasSubModules: module.HasSubModules,
					AccessLevel:   module.AccessLevel,
					RoleCode:      module.RoleCode,
					RoleName:      module.RoleName,
					SortOrder:     module.SortOrder,
					Active:        module.Active,
					Submodules:    submoduleResponses,
				})
			}

			label := strings.TrimSpace(tab.Label)
			if label == "" {
				label = strings.Title(tab.Key)
			}

			responseTabs = append(responseTabs, ModuleCatalogTabResponse{
				Key:     tab.Key,
				Label:   label,
				Modules: moduleResponses,
			})
		}

		c.JSON(http.StatusOK, ModuleCatalogResponse{
			Tabs:    responseTabs,
			Message: "Modules loaded successfully",
		})
	}
}

func hasAdminRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Code), "admin") {
			return true
		}
	}

	return false
}
