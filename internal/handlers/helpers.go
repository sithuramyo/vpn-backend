package handlers

import (
	"net"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vpn-backend/internal/repositories"
	"vpn-backend/pkg/response"
)

func paginationFromQuery(c *gin.Context) repositories.Pagination {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	return repositories.Pagination{Page: page, PageSize: pageSize}
}

func metaFromPagination(p repositories.Pagination, total int64) response.Meta {
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	return response.Meta{Page: page, PageSize: pageSize, TotalItems: total, TotalPages: totalPages}
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		response.Error(c, 400, "INVALID_ID", "Invalid identifier.")
		return uuid.Nil, false
	}
	return id, true
}

// clientIP prefers the parsed remote address over X-Forwarded-For so a
// client cannot spoof the audit trail's IP through request headers; set
// TrustedProxies in main.go for deployments that do sit behind a real proxy.
func clientIP(c *gin.Context) string {
	if ip := c.ClientIP(); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
