package ws

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, you should check the origin
		return true
	},
}

// Handler handles WebSocket connections
type Handler struct {
	hub        *Hub
	jwtManager *utils.JWTManager
	logger     *zap.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, jwtManager *utils.JWTManager, logger *zap.Logger) *Handler {
	return &Handler{
		hub:        hub,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// ServeWS handles WebSocket connection requests
// @Summary WebSocket 連線
// @Description 建立 WebSocket 連線進行即時通訊
// @Tags WebSocket
// @Param token query string true "JWT Token"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} map[string]string
// @Router /ws [get]
func (h *Handler) ServeWS(c *gin.Context) {
	// Get token from query parameter or header
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少認證 Token"})
		return
	}

	// Validate token
	claims, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		h.logger.Warn("Invalid token for WebSocket",
			zap.Error(err),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 Token"})
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket",
			zap.Error(err),
		)
		return
	}

	// Create client
	client := NewClient(h.hub, conn, claims.UserID, claims.Username, h.logger)

	// Register client
	h.hub.register <- client

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()
}

// GetStats returns WebSocket hub statistics
// @Summary 獲取 WebSocket 統計資訊
// @Description 獲取 WebSocket 連線統計資訊
// @Tags WebSocket
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]int
// @Router /api/v1/ws/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats := h.hub.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetOnlineUsers returns online users
// @Summary 獲取在線用戶
// @Description 獲取當前在線的用戶列表
// @Tags WebSocket
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string][]string
// @Router /api/v1/ws/online [get]
func (h *Handler) GetOnlineUsers(c *gin.Context) {
	users := h.hub.GetOnlineUsers()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users": users,
			"count": len(users),
		},
	})
}

// IsUserOnline checks if a specific user is online
// @Summary 檢查用戶是否在線
// @Description 檢查指定用戶是否在線
// @Tags WebSocket
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "用戶 ID"
// @Success 200 {object} map[string]bool
// @Router /api/v1/ws/online/{user_id} [get]
func (h *Handler) IsUserOnline(c *gin.Context) {
	userID := c.Param("user_id")
	online := h.hub.IsUserOnline(userID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id": userID,
			"online":  online,
		},
	})
}
