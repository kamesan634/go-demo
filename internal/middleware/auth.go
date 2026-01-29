package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/dto/response"
	"github.com/go-demo/chat/internal/pkg/utils"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	UserIDKey           = "user_id"
	UsernameKey         = "username"
	ClaimsKey           = "claims"
)

// Auth creates a JWT authentication middleware
func Auth(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			response.Unauthorized(c, "缺少認證 Token")
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			response.Unauthorized(c, "無效的認證格式")
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, BearerPrefix)
		if token == "" {
			response.Unauthorized(c, "Token 不能為空")
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			if err == utils.ErrExpiredToken {
				response.Unauthorized(c, "Token 已過期")
			} else {
				response.Unauthorized(c, "無效的 Token")
			}
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Set(ClaimsKey, claims)

		c.Next()
	}
}

// OptionalAuth creates an optional JWT authentication middleware
// It doesn't fail if no token is provided, but validates if one is present
func OptionalAuth(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, BearerPrefix)
		if token == "" {
			c.Next()
			return
		}

		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			// Token is invalid but optional, continue without user info
			c.Next()
			return
		}

		// Store user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Set(ClaimsKey, claims)

		c.Next()
	}
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetUsername retrieves username from context
func GetUsername(c *gin.Context) string {
	username, exists := c.Get(UsernameKey)
	if !exists {
		return ""
	}
	return username.(string)
}

// GetClaims retrieves JWT claims from context
func GetClaims(c *gin.Context) *utils.Claims {
	claims, exists := c.Get(ClaimsKey)
	if !exists {
		return nil
	}
	return claims.(*utils.Claims)
}

// IsAuthenticated checks if user is authenticated
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get(UserIDKey)
	return exists
}
