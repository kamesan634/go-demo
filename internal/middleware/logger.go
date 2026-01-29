package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

// RequestID generates a request ID for each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID retrieves request ID from context
func GetRequestID(c *gin.Context) string {
	requestID, exists := c.Get(RequestIDKey)
	if !exists {
		return ""
	}
	return requestID.(string)
}

// Logger creates a logging middleware
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request info
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		requestID := GetRequestID(c)
		userID := GetUserID(c)

		// Build log fields
		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("user_agent", userAgent),
			zap.String("request_id", requestID),
		}

		if userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}

		// Check for errors
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				fields = append(fields, zap.String("error", e.Error()))
			}
		}

		// Log based on status code
		switch {
		case statusCode >= 500:
			logger.Error("Server error", fields...)
		case statusCode >= 400:
			logger.Warn("Client error", fields...)
		case statusCode >= 300:
			logger.Info("Redirect", fields...)
		default:
			logger.Info("Request completed", fields...)
		}
	}
}

// Recovery creates a recovery middleware that logs panics
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := GetRequestID(c)

				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("request_id", requestID),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("ip", c.ClientIP()),
				)

				c.AbortWithStatusJSON(500, gin.H{
					"success": false,
					"error": gin.H{
						"code":    500,
						"message": "伺服器內部錯誤",
					},
				})
			}
		}()

		c.Next()
	}
}
