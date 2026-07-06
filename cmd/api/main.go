package main

import (
	"log"
	"net/http"
	"pos/internal/config"
	"pos/internal/database"

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
	
	var router *gin.Engine = gin.Default()
	router.SetTrustedProxies(nil)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	router.Run(":" + conf.Port)
	 
}
