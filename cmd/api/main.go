package main

import (
	"log"
	"net/http"
	"pos/internal/auth"
	"pos/internal/config"
	"pos/internal/database"
	adminhandler "pos/internal/handlers/admin"
	categoryhandler "pos/internal/handlers/business/category"
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

	router.Run(":" + conf.Port)
}
