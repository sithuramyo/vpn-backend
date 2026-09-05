package handlers

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

// handleServiceError maps a service-layer error to the right HTTP status.
// It logs unexpected (500-class) errors server-side but only ever returns
// a generic message to the client.
func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		response.Error(c, 404, "NOT_FOUND", "The requested resource was not found.")
	case errors.Is(err, services.ErrConflict):
		response.Error(c, 409, "CONFLICT", "This resource already exists or conflicts with an existing one.")
	case errors.Is(err, services.ErrValidation):
		response.Error(c, 422, "VALIDATION_ERROR", "Validation failed.")
	case errors.Is(err, services.ErrUnauthorized):
		response.Error(c, 401, "UNAUTHORIZED", "Authentication failed.")
	case errors.Is(err, services.ErrNotAuthorizedGoogleAccount):
		response.Error(c, 403, "FORBIDDEN", "Your Google account is not authorized to access this system.")
	default:
		log.Printf("internal error: %v", err)
		response.Error(c, 500, "INTERNAL_ERROR", "An unexpected error occurred.")
	}
}
