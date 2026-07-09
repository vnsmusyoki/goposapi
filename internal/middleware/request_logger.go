package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		contentType := c.GetHeader("Content-Type")
		contentLength := c.Request.ContentLength

		log.Printf("request start method=%s path=%s remote_ip=%s content_type=%s content_length=%d",
			method, path, c.ClientIP(), contentType, contentLength)

		c.Next()

		log.Printf("request end method=%s path=%s status=%d duration=%s errors=%d",
			method, path, c.Writer.Status(), time.Since(start), len(c.Errors))
	}
}
