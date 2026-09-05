package middleware

import "github.com/gin-gonic/gin"

// SecureHeaders sets conservative defaults for an API that is never
// rendered in a browser directly (it only ever answers JSON to the admin
// frontend), so a strict, no-surprises header set is safe here.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}
