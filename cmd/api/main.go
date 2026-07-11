package main

import (
	"log"
	"net/http"
	"pos/internal/auth"
	"pos/internal/config"
	"pos/internal/database"
	adminhandler "pos/internal/handlers/admin"
	categoryhandler "pos/internal/handlers/business/category"
	locationhandler "pos/internal/handlers/business/location"
	settingshandler "pos/internal/handlers/business/settings"
	"pos/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {
	var conf *config.Config
	var err error
	conf, err = config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var pool *pgxpool.Pool
	pool, err = database.Connect(conf.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer pool.Close()

	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(middleware.CORS(conf.FrontendOrigin))
	router.Use(middleware.RequestLogger())

	api := router.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	authService := auth.NewService(pool, conf.CookieSecure)
	authService.RegisterRoutes(api.Group("/auth"))
	api.POST("/businesses", adminhandler.CreateBusinessRequestHandler(pool))
	api.GET("/admin/businesses", adminhandler.ListBusinessesRequestHandler(pool, authService))
	api.POST("/admin/businesses/:id/sync-modules", adminhandler.SyncBusinessModulesRequestHandler(pool, authService))
	api.POST("/packages", adminhandler.CreatePackageRequestHandler(pool))
	api.GET("/admin/roles", adminhandler.ListRolesRequestHandler(pool, authService))
	api.GET("/admin/modules", adminhandler.ListModulesRequestHandler(pool, authService))
	api.POST("/admin/modules", adminhandler.CreateModuleRequestHandler(pool))
	api.PATCH("/admin/modules/reorder", adminhandler.ReorderModulesRequestHandler(pool))
	api.POST("/admin/modules/submodules", adminhandler.CreateSubmoduleRequestHandler(pool))
	api.PATCH("/admin/modules/submodules/:id", adminhandler.UpdateSubmoduleRequestHandler(pool))
	api.PATCH("/admin/modules/submodules/reorder", adminhandler.ReorderSubmodulesRequestHandler(pool))
	api.POST("/categories", categoryhandler.CreateCategoryRequestHandler(pool, authService))
	api.GET("/categories", categoryhandler.ListCategoriesRequestHandler(pool, authService))
	api.GET("/business/locations", locationhandler.GetBusinessLocationsRequestHandler(pool, authService))
	api.POST("/business/locations", locationhandler.CreateBusinessLocationRequestHandler(pool, authService))
	api.DELETE("/business/locations/:id", locationhandler.DeleteBusinessLocationRequestHandler(pool, authService))
	api.GET("/business/settings", settingshandler.GetBusinessSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings", settingshandler.UpdateBusinessSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/product", settingshandler.GetBusinessProductSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/product", settingshandler.UpdateBusinessProductSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/contact", settingshandler.GetBusinessContactSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/contact", settingshandler.UpdateBusinessContactSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/sale", settingshandler.GetBusinessSaleSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/sale", settingshandler.UpdateBusinessSaleSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/pos", settingshandler.GetBusinessPosSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/pos", settingshandler.UpdateBusinessPosSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/purchases", settingshandler.GetBusinessPurchasesSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/purchases", settingshandler.UpdateBusinessPurchasesSettingsRequestHandler(pool, authService))

	router.Run(":" + conf.Port)
}
