package main

import (
	"log"
	"net/http"
	"pos/internal/auth"
	"pos/internal/config"
	"pos/internal/database"
	adminhandler "pos/internal/handlers/admin"
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


	router.Run(":" + conf.Port)
}
