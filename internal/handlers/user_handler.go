package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"vpn-backend/internal/middleware"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

type UserHandler struct {
	users *services.UserService
	audit *services.AuditService
}

func NewUserHandler(users *services.UserService, audit *services.AuditService) *UserHandler {
	return &UserHandler{users: users, audit: audit}
}

func (h *UserHandler) List(c *gin.Context) {
	p := paginationFromQuery(c)
	f := repositories.UserFilter{Status: c.Query("status"), Search: c.Query("search")}
	users, total, err := h.users.List(f, p)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OKWithMeta(c, 200, users, metaFromPagination(p, total))
}

func (h *UserHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	user, err := h.users.Get(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, user)
}

type createUserRequest struct {
	Name              string     `json:"name" binding:"required"`
	Email             string     `json:"email" binding:"required,email"`
	ExpiresAt         *time.Time `json:"expires_at"`
	TrafficLimitBytes int64      `json:"traffic_limit_bytes"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 422, "VALIDATION_ERROR", "Name and a valid email are required.")
		return
	}

	user, err := h.users.Create(services.CreateUserInput{
		Name:              req.Name,
		Email:             req.Email,
		ExpiresAt:         req.ExpiresAt,
		TrafficLimitBytes: req.TrafficLimitBytes,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionUserCreated, "vpn_user", &user.ID, clientIP(c), map[string]any{"email": user.Email})

	response.OK(c, 201, user)
}

type updateUserRequest struct {
	Name              *string    `json:"name"`
	Email             *string    `json:"email"`
	ExpiresAt         *time.Time `json:"expires_at"`
	ClearExpiresAt    bool       `json:"clear_expires_at"`
	TrafficLimitBytes *int64     `json:"traffic_limit_bytes"`
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 422, "VALIDATION_ERROR", "Invalid request body.")
		return
	}

	input := services.UpdateUserInput{
		Name:              req.Name,
		Email:             req.Email,
		TrafficLimitBytes: req.TrafficLimitBytes,
	}
	if req.ClearExpiresAt {
		var nilTime *time.Time
		input.ExpiresAt = &nilTime
	} else if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt
		input.ExpiresAt = &expiresAt
	}

	user, err := h.users.Update(id, input)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionUserUpdated, "vpn_user", &user.ID, clientIP(c), nil)

	response.OK(c, 200, user)
}

func (h *UserHandler) Disable(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	user, err := h.users.Disable(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionUserDisabled, "vpn_user", &user.ID, clientIP(c), nil)

	response.OK(c, 200, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.users.Delete(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionUserDisabled, "vpn_user", &id, clientIP(c), map[string]any{"deleted": true})

	c.Status(204)
}
