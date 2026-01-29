package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 4096

	// Send buffer size
	sendBufferSize = 256
)

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	username string
	rooms    map[string]bool // Subscribed rooms
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewClient creates a new client
func NewClient(hub *Hub, conn *websocket.Conn, userID, username string, logger *zap.Logger) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, sendBufferSize),
		userID:   userID,
		username: username,
		rooms:    make(map[string]bool),
		logger:   logger,
	}
}

// GetUserID returns client's user ID
func (c *Client) GetUserID() string {
	return c.userID
}

// GetUsername returns client's username
func (c *Client) GetUsername() string {
	return c.username
}

// GetRooms returns client's subscribed rooms
func (c *Client) GetRooms() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rooms := make([]string, 0, len(c.rooms))
	for roomID := range c.rooms {
		rooms = append(rooms, roomID)
	}
	return rooms
}

// IsInRoom checks if client is in a room
func (c *Client) IsInRoom(roomID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rooms[roomID]
}

// JoinRoom adds client to a room
func (c *Client) JoinRoom(roomID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rooms[roomID] = true
}

// LeaveRoom removes client from a room
func (c *Client) LeaveRoom(roomID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rooms, roomID)
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("WebSocket read error",
					zap.String("user_id", c.userID),
					zap.Error(err),
				)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Warn("Failed to parse message",
				zap.String("user_id", c.userID),
				zap.Error(err),
			)
			c.sendError(400, "無效的訊息格式")
			continue
		}

		c.handleMessage(&msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming messages based on type
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MessageTypeJoinRoom:
		c.handleJoinRoom(msg)
	case MessageTypeLeaveRoom:
		c.handleLeaveRoom(msg)
	case MessageTypeSendMessage:
		c.handleSendMessage(msg)
	case MessageTypeSendDM:
		c.handleSendDM(msg)
	case MessageTypeTyping:
		c.handleTyping(msg)
	case MessageTypeStopTyping:
		c.handleStopTyping(msg)
	case MessageTypePing:
		c.handlePing(msg)
	case MessageTypeMarkRead:
		c.handleMarkRead(msg)
	default:
		c.sendError(400, "未知的訊息類型")
	}
}

func (c *Client) handleJoinRoom(msg *Message) {
	var payload JoinRoomPayload
	if err := msg.ParsePayload(&payload); err != nil {
		c.sendError(400, "無效的請求參數")
		return
	}

	c.hub.JoinRoom(c, payload.RoomID)
}

func (c *Client) handleLeaveRoom(msg *Message) {
	var payload LeaveRoomPayload
	if err := msg.ParsePayload(&payload); err != nil {
		c.sendError(400, "無效的請求參數")
		return
	}

	c.hub.LeaveRoom(c, payload.RoomID)
}

func (c *Client) handleSendMessage(msg *Message) {
	var payload SendMessagePayload
	if err := msg.ParsePayload(&payload); err != nil {
		c.sendError(400, "無效的請求參數")
		return
	}

	c.hub.SendMessage(c, payload, msg.RequestID)
}

func (c *Client) handleSendDM(msg *Message) {
	var payload SendDMPayload
	if err := msg.ParsePayload(&payload); err != nil {
		c.sendError(400, "無效的請求參數")
		return
	}

	c.hub.SendDirectMessage(c, payload, msg.RequestID)
}

func (c *Client) handleTyping(msg *Message) {
	var payload TypingPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return
	}

	c.hub.BroadcastTyping(c, payload.RoomID, true)
}

func (c *Client) handleStopTyping(msg *Message) {
	var payload TypingPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return
	}

	c.hub.BroadcastTyping(c, payload.RoomID, false)
}

func (c *Client) handlePing(msg *Message) {
	pongMsg, _ := NewMessage(MessageTypePong, nil)
	c.SendMessage(pongMsg)
}

func (c *Client) handleMarkRead(msg *Message) {
	var payload MarkReadPayload
	if err := msg.ParsePayload(&payload); err != nil {
		return
	}

	c.hub.MarkAsRead(c, payload)
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		c.logger.Error("Failed to marshal message",
			zap.String("user_id", c.userID),
			zap.Error(err),
		)
		return
	}

	select {
	case c.send <- data:
	default:
		// Channel is full, client is slow
		c.logger.Warn("Client send buffer full",
			zap.String("user_id", c.userID),
		)
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(code int, message string) {
	errMsg, _ := NewErrorMessage(code, message)
	c.SendMessage(errMsg)
}

// Close closes the client connection
func (c *Client) Close() {
	close(c.send)
}
