package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vpn-backend/internal/metrics"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repositories"
	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

type AccessKeyHandler struct {
	keys  *services.AccessKeyService
	audit *services.AuditService
}

func NewAccessKeyHandler(keys *services.AccessKeyService, audit *services.AuditService) *AccessKeyHandler {
	return &AccessKeyHandler{keys: keys, audit: audit}
}

func (h *AccessKeyHandler) List(c *gin.Context) {
	p := paginationFromQuery(c)
	f := repositories.AccessKeyFilter{Status: c.Query("status")}
	if userID := c.Query("user_id"); userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			f.UserID = &id
		}
	}
	if serverID := c.Query("server_id"); serverID != "" {
		if id, err := uuid.Parse(serverID); err == nil {
			f.ServerID = &id
		}
	}

	keys, total, err := h.keys.List(f, p)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OKWithMeta(c, 200, keys, metaFromPagination(p, total))
}

func (h *AccessKeyHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	key, err := h.keys.Get(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, key)
}

type createAccessKeyRequest struct {
	VPNUserID         uuid.UUID  `json:"vpn_user_id" binding:"required"`
	VPNServerID       uuid.UUID  `json:"vpn_server_id" binding:"required"`
	Name              string     `json:"name" binding:"required"`
	ExpiresAt         *time.Time `json:"expires_at"`
	TrafficLimitBytes int64      `json:"traffic_limit_bytes"`
	TCPEnabled        bool       `json:"tcp_enabled"`
	UDPEnabled        bool       `json:"udp_enabled"`
	WebSocketEnabled  bool       `json:"websocket_enabled"`
}

func (h *AccessKeyHandler) Create(c *gin.Context) {
	var req createAccessKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 422, "VALIDATION_ERROR", "vpn_user_id, vpn_server_id and name are required.")
		return
	}

	key, err := h.keys.Create(c.Request.Context(), services.CreateAccessKeyInput{
		VPNUserID:         req.VPNUserID,
		VPNServerID:       req.VPNServerID,
		Name:              req.Name,
		ExpiresAt:         req.ExpiresAt,
		TrafficLimitBytes: req.TrafficLimitBytes,
		TCPEnabled:        req.TCPEnabled,
		UDPEnabled:        req.UDPEnabled,
		WebSocketEnabled:  req.WebSocketEnabled,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}
	metrics.ConfigOperationsTotal.WithLabelValues("create").Inc()

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionAccessKeyCreated, "access_key", &key.ID, clientIP(c), map[string]any{"name": key.Name})

	response.OK(c, 201, key)
}

func (h *AccessKeyHandler) Revoke(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	key, err := h.keys.Revoke(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	metrics.ConfigOperationsTotal.WithLabelValues("revoke").Inc()

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionAccessKeyRevoked, "access_key", &key.ID, clientIP(c), nil)

	response.OK(c, 200, key)
}

func (h *AccessKeyHandler) Rotate(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	key, err := h.keys.Rotate(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	metrics.ConfigOperationsTotal.WithLabelValues("rotate").Inc()

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionAccessKeyRotated, "access_key", &key.ID, clientIP(c), nil)

	response.OK(c, 200, key)
}

// GetConfig returns the client configuration, including the decrypted
// secret. It is intentionally never logged and only reachable by an
// authenticated, authorized administrator explicitly requesting it.
func (h *AccessKeyHandler) GetConfig(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	cfg, err := h.keys.GetConfig(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, cfg)
}

func (h *AccessKeyHandler) GetQR(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	png, err := h.keys.GetQRPNG(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.Data(200, "image/png", png)
}

func (h *AccessKeyHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.keys.Delete(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionAccessKeyRevoked, "access_key", &id, clientIP(c), map[string]any{"deleted": true})

	c.Status(204)
}
