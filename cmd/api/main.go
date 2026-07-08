package main

import (
	"log"
	"net/http"
	"pos/internal/auth"
	"pos/internal/config"
	"pos/internal/database"
	"strings"

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
	router.Use(corsMiddleware(conf.FrontendOrigin))

	api := router.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	authService := auth.NewService(pool, conf.CookieSecure)
	authService.RegisterRoutes(api.Group("/auth"))

	router.Run(":" + conf.Port)
}

func corsMiddleware(frontendOrigin string) gin.HandlerFunc {
	allowed := map[string]struct{}{
		frontendOrigin:          {},
		"http://localhost:5173": {},
		"http://127.0.0.1:5173": {},
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok || strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Vary", "Origin")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
