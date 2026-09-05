package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vpn-backend/internal/middleware"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

type DeviceHandler struct {
	devices *services.DeviceService
	audit   *services.AuditService
}

func NewDeviceHandler(devices *services.DeviceService, audit *services.AuditService) *DeviceHandler {
	return &DeviceHandler{devices: devices, audit: audit}
}

func (h *DeviceHandler) List(c *gin.Context) {
	p := paginationFromQuery(c)
	f := repositories.DeviceFilter{Status: c.Query("status"), Platform: c.Query("platform")}
	if userID := c.Query("user_id"); userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			f.UserID = &id
		}
	}

	devices, total, err := h.devices.List(f, p)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OKWithMeta(c, 200, devices, metaFromPagination(p, total))
}

func (h *DeviceHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	device, err := h.devices.Get(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, device)
}

type updateDeviceRequest struct {
	Name   *string             `json:"name"`
	Status *models.DeviceStatus `json:"status"`
}

func (h *DeviceHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req updateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 422, "VALIDATION_ERROR", "Invalid request body.")
		return
	}
	device, err := h.devices.Update(id, services.UpdateDeviceInput{Name: req.Name, Status: req.Status})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, device)
}

func (h *DeviceHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.devices.Delete(id); err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionDeviceRevoked, "vpn_device", &id, clientIP(c), nil)

	c.Status(204)
}
