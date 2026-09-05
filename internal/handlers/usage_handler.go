package handlers

import (
	"github.com/gin-gonic/gin"

	"vpn-backend/internal/services"
	"vpn-backend/pkg/response"
)

type UsageHandler struct {
	usage *services.UsageService
}

func NewUsageHandler(usage *services.UsageService) *UsageHandler {
	return &UsageHandler{usage: usage}
}

func (h *UsageHandler) Summary(c *gin.Context) {
	summary, err := h.usage.Summary()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, summary)
}

func (h *UsageHandler) ByUser(c *gin.Context) {
	p := paginationFromQuery(c)
	usage, total, err := h.usage.ByUser(p)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OKWithMeta(c, 200, usage, metaFromPagination(p, total))
}

func (h *UsageHandler) ByServer(c *gin.Context) {
	usage, err := h.usage.ByServer()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.OK(c, 200, usage)
}
