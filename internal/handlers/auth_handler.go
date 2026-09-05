package handlers

import (
	"github.com/gin-gonic/gin"

	"vpn-backend/internal/middleware"
	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type googleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// Login exchanges a Google ID token (obtained by the frontend's own OAuth
// flow) for this backend's session token. It never accepts a role or
// status from the caller.
func (h *AuthHandler) Login(c *gin.Context) {
	var req googleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 422, "VALIDATION_ERROR", "id_token is required.")
		return
	}

	result, err := h.auth.LoginWithGoogle(req.IDToken, clientIP(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.OK(c, 200, gin.H{
		"token":      result.Token,
		"expires_at": result.ExpiresAt,
		"admin":      result.Admin,
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	admin := middleware.CurrentAdmin(c)
	if admin == nil {
		response.Error(c, 401, "UNAUTHORIZED", "Authentication required.")
		return
	}
	response.OK(c, 200, admin)
}

// Logout is a client-driven action (drop the token) since sessions are
// stateless JWTs; it still returns 204 so the frontend has a symmetric
// endpoint to call for its own cookie/session cleanup.
func (h *AuthHandler) Logout(c *gin.Context) {
	c.Status(204)
}
