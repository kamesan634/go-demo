package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept",
			"Accept-Encoding",
			"Authorization",
			"X-Request-ID",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
			"X-Request-ID",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// CORS creates a CORS middleware with default configuration
func CORS() gin.HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig creates a CORS middleware with custom configuration
func CORSWithConfig(config *CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// Check if origin is allowed
		allowed := false
		for _, o := range config.AllowOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if !allowed {
			c.Next()
			return
		}

		// Set CORS headers
		if config.AllowOrigins[0] == "*" && !config.AllowCredentials {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", joinStrings(config.AllowMethods))
		c.Header("Access-Control-Allow-Headers", joinStrings(config.AllowHeaders))
		c.Header("Access-Control-Expose-Headers", joinStrings(config.ExposeHeaders))

		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if config.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", formatDuration(config.MaxAge))
		}

		// Handle preflight request
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func joinStrings(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += ", " + strs[i]
	}
	return result
}

func formatDuration(d time.Duration) string {
	return string(rune(int(d.Seconds())))
}
