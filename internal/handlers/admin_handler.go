package handlers

import (
	"github.com/gin-gonic/gin"

	"vpn-backend/internal/middleware"
	"vpn-backend/internal/models"
	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

type AdminHandler struct {
	admins *services.AdminService
	audit  *services.AuditService
}

func NewAdminHandler(admins *services.AdminService, audit *services.AuditService) *AdminHandler {
	return &AdminHandler{admins: admins, audit: audit}
}

func (h *AdminHandler) List(c *gin.Context) {
	p := paginationFromQuery(c)
	admins, total, err := h.admins.List(p)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OKWithMeta(c, 200, admins, metaFromPagination(p, total))
}

func (h *AdminHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	admin, err := h.admins.Get(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, admin)
}

type updateAdminRequest struct {
	Role   *models.AdminRole   `json:"role"`
	Status *models.AdminStatus `json:"status"`
}

func (h *AdminHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req updateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 422, "VALIDATION_ERROR", "Invalid request body.")
		return
	}

	admin, err := h.admins.Update(id, services.UpdateAdminInput{Role: req.Role, Status: req.Status})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	action := models.ActionUserUpdated
	if req.Status != nil && *req.Status == models.AdminStatusDisabled {
		action = models.ActionAdminDisabled
	}
	_ = h.audit.Record(&actor.ID, action, "admin", &admin.ID, clientIP(c), map[string]any{"admin_email": admin.Email})

	response.OK(c, 200, admin)
}
