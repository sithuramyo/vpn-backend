package handlers

import (
	"github.com/gin-gonic/gin"

	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

// MetricsSummaryHandler backs GET /api/v1/metrics/summary, the aggregate
// counts the admin dashboard renders as stat tiles. This is distinct from
// GET /metrics (Prometheus text exposition, see router.go).
type MetricsSummaryHandler struct {
	users      *services.UserService
	devices    *services.DeviceService
	accessKeys *services.AccessKeyService
	servers    *services.ServerService
}

func NewMetricsSummaryHandler(users *services.UserService, devices *services.DeviceService, accessKeys *services.AccessKeyService, servers *services.ServerService) *MetricsSummaryHandler {
	return &MetricsSummaryHandler{users: users, devices: devices, accessKeys: accessKeys, servers: servers}
}

type dashboardSummary struct {
	TotalUsers          int64 `json:"total_users"`
	ActiveUsers         int64 `json:"active_users"`
	ActiveDevices       int64 `json:"active_devices"`
	ActiveAccessKeys    int64 `json:"active_access_keys"`
}

func (h *MetricsSummaryHandler) Summary(c *gin.Context) {
	total, err := h.users.CountTotal()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	active, err := h.users.CountActive()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	activeDevices, err := h.devices.CountActive()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	activeKeys, err := h.accessKeys.CountActive()
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.OK(c, 200, dashboardSummary{
		TotalUsers:       total,
		ActiveUsers:      active,
		ActiveDevices:    activeDevices,
		ActiveAccessKeys: activeKeys,
	})
}
