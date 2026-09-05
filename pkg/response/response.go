// Package response provides a consistent JSON envelope for API responses.
package response

import "github.com/gin-gonic/gin"

type Meta struct {
	Page       int   `json:"page,omitempty"`
	PageSize   int   `json:"page_size,omitempty"`
	TotalItems int64 `json:"total_items,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func OK(c *gin.Context, status int, data any) {
	c.JSON(status, envelope{Data: data})
}

func OKWithMeta(c *gin.Context, status int, data any, meta Meta) {
	c.JSON(status, envelope{Data: data, Meta: &meta})
}

func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, envelope{Error: &apiError{Code: code, Message: message}})
}

func ErrorWithDetails(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, envelope{Error: &apiError{Code: code, Message: message, Details: details}})
}
