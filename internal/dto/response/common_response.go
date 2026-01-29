package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/go-demo/chat/internal/pkg/errors"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Success sends a success response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage sends a success response with a message
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created sends a 201 created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// NoContent sends a 204 no content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error sends an error response
func Error(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if e, ok := err.(*apperrors.AppError); ok {
		appErr = e
	} else {
		appErr = apperrors.ErrInternal
	}

	c.JSON(appErr.Code, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
	})
}

// ErrorWithStatus sends an error response with a specific status code
func ErrorWithStatus(c *gin.Context, status int, message string) {
	c.JSON(status, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    status,
			Message: message,
		},
	})
}

// BadRequest sends a 400 bad request response
func BadRequest(c *gin.Context, message string) {
	ErrorWithStatus(c, http.StatusBadRequest, message)
}

// Unauthorized sends a 401 unauthorized response
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "未授權的請求"
	}
	ErrorWithStatus(c, http.StatusUnauthorized, message)
}

// Forbidden sends a 403 forbidden response
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "禁止存取"
	}
	ErrorWithStatus(c, http.StatusForbidden, message)
}

// NotFound sends a 404 not found response
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "資源不存在"
	}
	ErrorWithStatus(c, http.StatusNotFound, message)
}

// InternalError sends a 500 internal server error response
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "伺服器內部錯誤"
	}
	ErrorWithStatus(c, http.StatusInternalServerError, message)
}

// ValidationError sends a 400 response with validation errors
func ValidationError(c *gin.Context, details interface{}) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    http.StatusBadRequest,
			Message: "驗證失敗",
			Details: details,
		},
	})
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// NewPaginatedResponse creates a paginated response
func NewPaginatedResponse(items interface{}, total, page, limit int) *PaginatedResponse {
	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
}
