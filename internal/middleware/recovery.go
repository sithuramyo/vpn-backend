package middleware

import (
	"log"

	"github.com/gin-gonic/gin"

	"vpn-backend/pkg/response"
)

// Recovery converts a panic into a clean 500 JSON response instead of
// letting Gin's default recovery dump a stack trace (which can leak
// internal details) to the client.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered: %v", r)
				response.Error(c, 500, "INTERNAL_ERROR", "An unexpected error occurred.")
			}
		}()
		c.Next()
	}
}
