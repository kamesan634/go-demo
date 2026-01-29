package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-demo/chat/internal/model"
	"github.com/go-demo/chat/internal/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	RoomID  string
	Message *Message
	Sender  *Client // nil for system messages
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Clients by room: roomID -> clients
	rooms map[string]map[*Client]bool

	// Clients by user: userID -> clients (supports multiple connections)
	users map[string]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to room
	broadcast chan *BroadcastMessage

	// Direct message to user
	directMessage chan *DirectMessageBroadcast

	// Mutex for thread-safe access
	mu sync.RWMutex

	// Services
	roomService    *service.RoomService
	messageService *service.MessageService
	dmService      *service.DirectMessageService
	userService    *service.UserService

	// Redis for Pub/Sub (horizontal scaling)
	redis *redis.Client

	// Logger
	logger *zap.Logger
}

// DirectMessageBroadcast represents a DM to send
type DirectMessageBroadcast struct {
	ReceiverID string
	Message    *Message
}

// NewHub creates a new Hub
func NewHub(
	roomService *service.RoomService,
	messageService *service.MessageService,
	dmService *service.DirectMessageService,
	userService *service.UserService,
	redisClient *redis.Client,
	logger *zap.Logger,
) *Hub {
	return &Hub{
		clients:        make(map[*Client]bool),
		rooms:          make(map[string]map[*Client]bool),
		users:          make(map[string]map[*Client]bool),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcast:      make(chan *BroadcastMessage, 256),
		directMessage:  make(chan *DirectMessageBroadcast, 256),
		roomService:    roomService,
		messageService: messageService,
		dmService:      dmService,
		userService:    userService,
		redis:          redisClient,
		logger:         logger,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	// Start Redis subscriber in goroutine
	go h.subscribeRedis()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.broadcast:
			h.broadcastToRoom(msg)

		case dm := <-h.directMessage:
			h.sendToUser(dm.ReceiverID, dm.Message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	// Add to users map
	if h.users[client.userID] == nil {
		h.users[client.userID] = make(map[*Client]bool)
	}
	h.users[client.userID][client] = true

	h.logger.Info("Client connected",
		zap.String("user_id", client.userID),
		zap.String("username", client.username),
		zap.Int("total_clients", len(h.clients)),
	)

	// Update user status
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.userService.UpdateStatus(ctx, client.userID, model.UserStatusOnline)
	}()

	// Broadcast user online
	go h.broadcastUserStatus(client, true)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()

	if _, ok := h.clients[client]; !ok {
		h.mu.Unlock()
		return
	}

	delete(h.clients, client)

	// Remove from users map
	if userClients, ok := h.users[client.userID]; ok {
		delete(userClients, client)
		if len(userClients) == 0 {
			delete(h.users, client.userID)
		}
	}

	// Remove from all rooms
	for roomID := range client.rooms {
		if roomClients, ok := h.rooms[roomID]; ok {
			delete(roomClients, client)
			if len(roomClients) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}

	h.mu.Unlock()

	client.Close()

	h.logger.Info("Client disconnected",
		zap.String("user_id", client.userID),
		zap.String("username", client.username),
	)

	// Check if user has no more connections
	h.mu.RLock()
	hasOtherConnections := len(h.users[client.userID]) > 0
	h.mu.RUnlock()

	if !hasOtherConnections {
		// Update user status
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			h.userService.UpdateStatus(ctx, client.userID, model.UserStatusOffline)
		}()

		// Broadcast user offline
		go h.broadcastUserStatus(client, false)
	}
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(client *Client, roomID string) {
	// Check if user is member of the room
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	isMember, err := h.roomService.IsMember(ctx, roomID, client.userID)
	if err != nil {
		client.sendError(500, "伺服器錯誤")
		return
	}

	if !isMember {
		client.sendError(403, "您不是該聊天室的成員")
		return
	}

	h.mu.Lock()
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]bool)
	}
	h.rooms[roomID][client] = true
	h.mu.Unlock()

	client.JoinRoom(roomID)

	// Get room info
	room, err := h.roomService.GetByIDWithDetails(ctx, roomID)
	if err != nil {
		return
	}

	// Send room joined confirmation
	joinedMsg, _ := NewMessage(MessageTypeRoomJoined, &RoomJoinedPayload{
		RoomID:      roomID,
		RoomName:    room.Name,
		MemberCount: room.MemberCount,
	})
	client.SendMessage(joinedMsg)

	h.logger.Debug("Client joined room",
		zap.String("user_id", client.userID),
		zap.String("room_id", roomID),
	)
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(client *Client, roomID string) {
	h.mu.Lock()
	if roomClients, ok := h.rooms[roomID]; ok {
		delete(roomClients, client)
		if len(roomClients) == 0 {
			delete(h.rooms, roomID)
		}
	}
	h.mu.Unlock()

	client.LeaveRoom(roomID)

	// Send room left confirmation
	leftMsg, _ := NewMessage(MessageTypeRoomLeft, &LeaveRoomPayload{RoomID: roomID})
	client.SendMessage(leftMsg)

	h.logger.Debug("Client left room",
		zap.String("user_id", client.userID),
		zap.String("room_id", roomID),
	)
}

// SendMessage sends a message to a room
func (h *Hub) SendMessage(client *Client, payload SendMessagePayload, requestID string) {
	if !client.IsInRoom(payload.RoomID) {
		client.sendError(403, "您尚未加入該聊天室")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get user info for broadcast
	user, err := h.userService.GetByID(ctx, client.userID)
	if err != nil {
		client.sendError(500, "伺服器錯誤")
		return
	}

	// Save message
	msgType := model.MessageTypeText
	if payload.Type == "image" {
		msgType = model.MessageTypeImage
	} else if payload.Type == "file" {
		msgType = model.MessageTypeFile
	}

	msg, err := h.messageService.SendMessage(ctx, &service.SendMessageInput{
		RoomID:    payload.RoomID,
		UserID:    client.userID,
		Content:   payload.Content,
		Type:      msgType,
		ReplyToID: payload.ReplyToID,
	})
	if err != nil {
		client.sendError(500, "發送訊息失敗")
		return
	}

	// Send acknowledgement to sender
	ackMsg, _ := NewMessage(MessageTypeAck, &AckPayload{
		RequestID: requestID,
		Success:   true,
		MessageID: msg.ID,
	})
	client.SendMessage(ackMsg)

	// Broadcast to room
	broadcastPayload := &NewMessagePayload{
		ID:          msg.ID,
		RoomID:      msg.RoomID,
		UserID:      msg.UserID,
		Username:    user.Username,
		DisplayName: user.GetDisplayName(),
		AvatarURL:   user.GetAvatarURL(),
		Content:     msg.Content,
		Type:        string(msg.Type),
		ReplyToID:   msg.GetReplyToID(),
		CreatedAt:   msg.CreatedAt.Format(time.RFC3339),
	}

	broadcastMsg, _ := NewMessage(MessageTypeNewMessage, broadcastPayload)

	h.broadcast <- &BroadcastMessage{
		RoomID:  payload.RoomID,
		Message: broadcastMsg,
		Sender:  client,
	}

	// Publish to Redis for horizontal scaling
	h.publishToRedis("room:"+payload.RoomID, broadcastMsg)
}

// SendDirectMessage sends a direct message
func (h *Hub) SendDirectMessage(client *Client, payload SendDMPayload, requestID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get sender info
	sender, err := h.userService.GetByID(ctx, client.userID)
	if err != nil {
		client.sendError(500, "伺服器錯誤")
		return
	}

	// Save DM
	msgType := model.MessageTypeText
	if payload.Type == "image" {
		msgType = model.MessageTypeImage
	} else if payload.Type == "file" {
		msgType = model.MessageTypeFile
	}

	dm, err := h.dmService.SendMessage(ctx, &service.SendDMInput{
		SenderID:   client.userID,
		ReceiverID: payload.ReceiverID,
		Content:    payload.Content,
		Type:       msgType,
	})
	if err != nil {
		client.sendError(500, "發送訊息失敗")
		return
	}

	// Send acknowledgement
	ackMsg, _ := NewMessage(MessageTypeAck, &AckPayload{
		RequestID: requestID,
		Success:   true,
		MessageID: dm.ID,
	})
	client.SendMessage(ackMsg)

	// Send to receiver
	dmPayload := &NewDMPayload{
		ID:                dm.ID,
		SenderID:          dm.SenderID,
		SenderUsername:    sender.Username,
		SenderDisplayName: sender.GetDisplayName(),
		SenderAvatarURL:   sender.GetAvatarURL(),
		Content:           dm.Content,
		Type:              string(dm.Type),
		CreatedAt:         dm.CreatedAt.Format(time.RFC3339),
	}

	dmMsg, _ := NewMessage(MessageTypeNewDM, dmPayload)

	h.directMessage <- &DirectMessageBroadcast{
		ReceiverID: payload.ReceiverID,
		Message:    dmMsg,
	}

	// Also send to sender (for multi-device sync)
	client.SendMessage(dmMsg)

	// Publish to Redis
	h.publishToRedis("dm:"+payload.ReceiverID, dmMsg)
}

// BroadcastTyping broadcasts typing indicator
func (h *Hub) BroadcastTyping(client *Client, roomID string, isTyping bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.userService.GetByID(ctx, client.userID)
	if err != nil {
		return
	}

	payload := &UserTypingPayload{
		RoomID:      roomID,
		UserID:      client.userID,
		Username:    user.Username,
		DisplayName: user.GetDisplayName(),
	}

	var msgType MessageType
	if isTyping {
		msgType = MessageTypeUserTyping
	} else {
		msgType = MessageTypeUserStopTyping
	}

	msg, _ := NewMessage(msgType, payload)

	h.broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Message: msg,
		Sender:  client,
	}
}

// MarkAsRead handles mark as read
func (h *Hub) MarkAsRead(client *Client, payload MarkReadPayload) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if payload.RoomID != "" {
		// Room message read
		h.roomService.UpdateLastRead(ctx, payload.RoomID, client.userID)
	} else if payload.SenderID != "" {
		// DM read
		h.dmService.MarkAsRead(ctx, client.userID, payload.SenderID)

		// Notify sender
		readMsg, _ := NewMessage(MessageTypeDMRead, &DMReadPayload{
			SenderID:   payload.SenderID,
			ReceiverID: client.userID,
			ReadAt:     time.Now().Format(time.RFC3339),
		})

		h.directMessage <- &DirectMessageBroadcast{
			ReceiverID: payload.SenderID,
			Message:    readMsg,
		}
	}
}

func (h *Hub) broadcastToRoom(bm *BroadcastMessage) {
	h.mu.RLock()
	clients := h.rooms[bm.RoomID]
	h.mu.RUnlock()

	for client := range clients {
		// Skip sender for certain message types (they already have acknowledgement)
		if bm.Sender != nil && client == bm.Sender {
			// Still send to other devices of the same user
			if client.userID == bm.Sender.userID && client != bm.Sender {
				client.SendMessage(bm.Message)
			}
			continue
		}
		client.SendMessage(bm.Message)
	}
}

func (h *Hub) sendToUser(userID string, msg *Message) {
	h.mu.RLock()
	clients := h.users[userID]
	h.mu.RUnlock()

	for client := range clients {
		client.SendMessage(msg)
	}
}

func (h *Hub) broadcastUserStatus(client *Client, online bool) {
	status := "offline"
	if online {
		status = "online"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.userService.GetByID(ctx, client.userID)
	if err != nil {
		return
	}

	payload := &UserStatusPayload{
		UserID:      client.userID,
		Username:    user.Username,
		DisplayName: user.GetDisplayName(),
		Status:      status,
	}

	var msgType MessageType
	if online {
		msgType = MessageTypeUserOnline
	} else {
		msgType = MessageTypeUserOffline
	}

	msg, _ := NewMessage(msgType, payload)

	// Broadcast to all rooms the user is in
	for roomID := range client.rooms {
		h.broadcast <- &BroadcastMessage{
			RoomID:  roomID,
			Message: msg,
			Sender:  nil, // System message
		}
	}
}

// Redis Pub/Sub for horizontal scaling
func (h *Hub) publishToRedis(channel string, msg *Message) {
	if h.redis == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	ctx := context.Background()
	h.redis.Publish(ctx, channel, data)
}

func (h *Hub) subscribeRedis() {
	if h.redis == nil {
		return
	}

	ctx := context.Background()
	pubsub := h.redis.PSubscribe(ctx, "room:*", "dm:*")
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		var wsMsg Message
		if err := json.Unmarshal([]byte(msg.Payload), &wsMsg); err != nil {
			continue
		}

		// Check channel type and broadcast
		// This allows messages from other instances to be delivered
		// Implementation depends on your scaling strategy
	}
}

// GetOnlineUsers returns online user IDs
func (h *Hub) GetOnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := make([]string, 0, len(h.users))
	for userID := range h.users {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

// IsUserOnline checks if a user is online
func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users[userID]) > 0
}

// GetRoomClients returns the number of clients in a room
func (h *Hub) GetRoomClients(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]int{
		"total_clients":  len(h.clients),
		"online_users":   len(h.users),
		"active_rooms":   len(h.rooms),
	}
}
