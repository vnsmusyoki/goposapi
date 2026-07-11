package admin

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repoadmin "pos/internal/repository/admin"
)

type RoleCatalogResponse struct {
	Roles   []repoadmin.RoleCatalogItem `json:"roles"`
	Message string                      `json:"message"`
}

func ListRolesRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list roles handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasAdminRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Admin access is required"})
			return
		}

		roles, err := repoadmin.ListRolesRepository(pool)
		if err != nil {
			log.Printf("list roles handler: repository failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load roles"})
			return
		}

		c.JSON(http.StatusOK, RoleCatalogResponse{
			Roles:   roles,
			Message: "Roles loaded successfully",
		})
	}
}
