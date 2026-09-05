package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"vpn-backend/internal/repositories"
	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

type AuditLogHandler struct {
	audit *services.AuditService
}

func NewAuditLogHandler(audit *services.AuditService) *AuditLogHandler {
	return &AuditLogHandler{audit: audit}
}

func (h *AuditLogHandler) List(c *gin.Context) {
	p := paginationFromQuery(c)
	f := repositories.AuditLogFilter{
		AdminID:      c.Query("admin_id"),
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		ResourceID:   c.Query("resource_id"),
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			f.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			f.To = &t
		}
	}

	logs, total, err := h.audit.List(f, p)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OKWithMeta(c, 200, logs, metaFromPagination(p, total))
}
