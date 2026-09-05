package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"vpn-backend/internal/middleware"
	"vpn-backend/internal/models"
	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

type ServerHandler struct {
	servers *services.ServerService
	audit   *services.AuditService
}

func NewServerHandler(servers *services.ServerService, audit *services.AuditService) *ServerHandler {
	return &ServerHandler{servers: servers, audit: audit}
}

func (h *ServerHandler) List(c *gin.Context) {
	p := paginationFromQuery(c)
	servers, total, err := h.servers.List(p)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OKWithMeta(c, 200, servers, metaFromPagination(p, total))
}

func (h *ServerHandler) Get(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	server, err := h.servers.Get(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, server)
}

type createServerRequest struct {
	Name     string `json:"name" binding:"required"`
	Hostname string `json:"hostname" binding:"required"`
	PublicIP string `json:"public_ip"`
	Country  string `json:"country"`
	City     string `json:"city"`
	VPNPort  int    `json:"vpn_port"`
	TLSPort  int    `json:"tls_port"`
}

func (h *ServerHandler) Create(c *gin.Context) {
	var req createServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 422, "VALIDATION_ERROR", "name and hostname are required.")
		return
	}

	server, err := h.servers.Create(services.CreateServerInput{
		Name:     req.Name,
		Hostname: req.Hostname,
		PublicIP: req.PublicIP,
		Country:  req.Country,
		City:     req.City,
		VPNPort:  req.VPNPort,
		TLSPort:  req.TLSPort,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionServerCreated, "vpn_server", &server.ID, clientIP(c), map[string]any{"name": server.Name})

	response.OK(c, 201, server)
}

type updateServerRequest struct {
	Name    *string             `json:"name"`
	Status  *models.ServerStatus `json:"status"`
	VPNPort *int                `json:"vpn_port"`
	TLSPort *int                `json:"tls_port"`
}

func (h *ServerHandler) Update(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req updateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 422, "VALIDATION_ERROR", "Invalid request body.")
		return
	}

	server, err := h.servers.Update(id, services.UpdateServerInput{
		Name: req.Name, Status: req.Status, VPNPort: req.VPNPort, TLSPort: req.TLSPort,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionServerUpdated, "vpn_server", &server.ID, clientIP(c), nil)

	response.OK(c, 200, server)
}

func (h *ServerHandler) Delete(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.servers.Delete(id); err != nil {
		handleServiceError(c, err)
		return
	}

	actor := middleware.CurrentAdmin(c)
	_ = h.audit.Record(&actor.ID, models.ActionServerDeleted, "vpn_server", &id, clientIP(c), nil)

	c.Status(204)
}

func (h *ServerHandler) Health(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	report, err := h.servers.Health(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, report)
}

// MetricsHistory backs the CPU/memory/bandwidth/connection charts on the
// server detail page. ?window accepts a Go duration string (default 24h).
func (h *ServerHandler) MetricsHistory(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	window := 24 * time.Hour
	if raw := c.Query("window"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			window = parsed
		}
	}

	history, err := h.servers.MetricsHistory(id, window)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, history)
}
