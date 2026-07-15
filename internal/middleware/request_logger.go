package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"pos/internal/auth"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		contentType := c.GetHeader("Content-Type")
		contentLength := c.Request.ContentLength
		cookieName := auth.SessionCookieName()
		cookieState := "missing"
		cookieFingerprint := ""
		if cookie, err := c.Request.Cookie(cookieName); err == nil && cookie.Value != "" {
			cookieState = "present"
			cookieFingerprint = auth.SessionTokenFingerprint(cookie.Value)
		}

		log.Printf("request start method=%s path=%s remote_ip=%s content_type=%s content_length=%d session_cookie_name=%s session_cookie_state=%s session_cookie_fingerprint=%s",
			method, path, c.ClientIP(), contentType, contentLength, cookieName, cookieState, cookieFingerprint)

		c.Next()

		log.Printf("request end method=%s path=%s status=%d duration=%s errors=%d",
			method, path, c.Writer.Status(), time.Since(start), len(c.Errors))
	}
}
