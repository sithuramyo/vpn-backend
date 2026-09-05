package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"vpn-backend/internal/auth"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/pkg/response"
)

const contextAdminKey = "current_admin"

// RequireAuth validates the bearer session token, re-loads the admin from
// PostgreSQL on every request (so a disabled admin or role change is
// enforced immediately), and rejects anything but an ACTIVE admin.
func RequireAuth(sessions *auth.SessionManager, admins *repositories.AdminRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c)
		if tokenString == "" {
			response.Error(c, 401, "UNAUTHORIZED", "Missing or malformed Authorization header.")
			return
		}

		adminID, err := sessions.Verify(tokenString)
		if err != nil {
			response.Error(c, 401, "UNAUTHORIZED", "Session expired or invalid.")
			return
		}

		admin, err := admins.FindByID(adminID)
		if err != nil {
			response.Error(c, 401, "UNAUTHORIZED", "Session expired or invalid.")
			return
		}

		if admin.Status != models.AdminStatusActive {
			response.Error(c, 403, "FORBIDDEN", "Your Google account is not authorized to access this system.")
			return
		}

		c.Set(contextAdminKey, admin)
		c.Next()
	}
}

// RequireRole must run after RequireAuth. It authorizes purely against the
// role loaded from PostgreSQL in this same request — a role claimed by the
// client is never consulted.
func RequireRole(allowed ...models.AdminRole) gin.HandlerFunc {
	allowedSet := make(map[models.AdminRole]bool, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = true
	}

	return func(c *gin.Context) {
		admin := CurrentAdmin(c)
		if admin == nil {
			response.Error(c, 401, "UNAUTHORIZED", "Authentication required.")
			return
		}
		if !allowedSet[admin.Role] {
			response.Error(c, 403, "FORBIDDEN", "Permission denied.")
			return
		}
		c.Next()
	}
}

func CurrentAdmin(c *gin.Context) *models.Admin {
	v, ok := c.Get(contextAdminKey)
	if !ok {
		return nil
	}
	admin, ok := v.(*models.Admin)
	if !ok {
		return nil
	}
	return admin
}

func extractBearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
