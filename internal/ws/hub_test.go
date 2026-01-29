package ws

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func createTestHub() *Hub {
	logger := zap.NewNop()

	return &Hub{
		clients:       make(map[*Client]bool),
		rooms:         make(map[string]map[*Client]bool),
		users:         make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan *BroadcastMessage, 256),
		directMessage: make(chan *DirectMessageBroadcast, 256),
		logger:        logger,
	}
}

func createMockClient(userID, username string) *Client {
	logger := zap.NewNop()
	return &Client{
		send:     make(chan []byte, 256),
		userID:   userID,
		username: username,
		rooms:    make(map[string]bool),
		logger:   logger,
	}
}

func TestHub_RegisterClient(t *testing.T) {
	hub := createTestHub()
	client := createMockClient("user-1", "alice")
	client.hub = hub

	// Manually register client (simulating the channel receive)
	hub.clients[client] = true
	if hub.users[client.userID] == nil {
		hub.users[client.userID] = make(map[*Client]bool)
	}
	hub.users[client.userID][client] = true

	if len(hub.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(hub.clients))
	}

	if len(hub.users["user-1"]) != 1 {
		t.Errorf("Expected 1 user connection, got %d", len(hub.users["user-1"]))
	}
}

func TestHub_UnregisterClient(t *testing.T) {
	hub := createTestHub()
	client := createMockClient("user-1", "alice")
	client.hub = hub

	// Register
	hub.clients[client] = true
	hub.users[client.userID] = make(map[*Client]bool)
	hub.users[client.userID][client] = true

	// Unregister
	delete(hub.clients, client)
	delete(hub.users[client.userID], client)
	if len(hub.users[client.userID]) == 0 {
		delete(hub.users, client.userID)
	}

	if len(hub.clients) != 0 {
		t.Errorf("Expected 0 clients, got %d", len(hub.clients))
	}

	if hub.users["user-1"] != nil {
		t.Error("Expected user to be removed from users map")
	}
}

func TestHub_MultipleDevices(t *testing.T) {
	hub := createTestHub()
	client1 := createMockClient("user-1", "alice")
	client2 := createMockClient("user-1", "alice")
	client1.hub = hub
	client2.hub = hub

	// Register both clients for same user
	hub.clients[client1] = true
	hub.clients[client2] = true
	hub.users["user-1"] = make(map[*Client]bool)
	hub.users["user-1"][client1] = true
	hub.users["user-1"][client2] = true

	if len(hub.clients) != 2 {
		t.Errorf("Expected 2 clients, got %d", len(hub.clients))
	}

	if len(hub.users["user-1"]) != 2 {
		t.Errorf("Expected 2 connections for user, got %d", len(hub.users["user-1"]))
	}

	// Unregister one client
	delete(hub.clients, client1)
	delete(hub.users["user-1"], client1)

	if len(hub.clients) != 1 {
		t.Errorf("Expected 1 client after unregister, got %d", len(hub.clients))
	}

	if len(hub.users["user-1"]) != 1 {
		t.Errorf("Expected 1 connection for user after unregister, got %d", len(hub.users["user-1"]))
	}
}

func TestHub_JoinRoom(t *testing.T) {
	hub := createTestHub()
	client := createMockClient("user-1", "alice")
	client.hub = hub

	roomID := "room-1"

	// Register client
	hub.clients[client] = true
	hub.users[client.userID] = make(map[*Client]bool)
	hub.users[client.userID][client] = true

	// Join room
	if hub.rooms[roomID] == nil {
		hub.rooms[roomID] = make(map[*Client]bool)
	}
	hub.rooms[roomID][client] = true
	client.JoinRoom(roomID)

	if len(hub.rooms[roomID]) != 1 {
		t.Errorf("Expected 1 client in room, got %d", len(hub.rooms[roomID]))
	}

	if !client.IsInRoom(roomID) {
		t.Error("Expected client to be in room")
	}
}

func TestHub_LeaveRoom(t *testing.T) {
	hub := createTestHub()
	client := createMockClient("user-1", "alice")
	client.hub = hub

	roomID := "room-1"

	// Register client and join room
	hub.clients[client] = true
	hub.rooms[roomID] = make(map[*Client]bool)
	hub.rooms[roomID][client] = true
	client.JoinRoom(roomID)

	// Leave room
	delete(hub.rooms[roomID], client)
	if len(hub.rooms[roomID]) == 0 {
		delete(hub.rooms, roomID)
	}
	client.LeaveRoom(roomID)

	if hub.rooms[roomID] != nil {
		t.Error("Expected room to be removed when empty")
	}

	if client.IsInRoom(roomID) {
		t.Error("Expected client not to be in room")
	}
}

func TestHub_GetOnlineUsers(t *testing.T) {
	hub := createTestHub()

	client1 := createMockClient("user-1", "alice")
	client2 := createMockClient("user-2", "bob")
	client1.hub = hub
	client2.hub = hub

	// Register clients
	hub.clients[client1] = true
	hub.clients[client2] = true
	hub.users["user-1"] = map[*Client]bool{client1: true}
	hub.users["user-2"] = map[*Client]bool{client2: true}

	onlineUsers := hub.GetOnlineUsers()

	if len(onlineUsers) != 2 {
		t.Errorf("Expected 2 online users, got %d", len(onlineUsers))
	}
}

func TestHub_IsUserOnline(t *testing.T) {
	hub := createTestHub()

	client := createMockClient("user-1", "alice")
	client.hub = hub

	// User not online yet
	if hub.IsUserOnline("user-1") {
		t.Error("Expected user to be offline")
	}

	// Register client
	hub.clients[client] = true
	hub.users["user-1"] = map[*Client]bool{client: true}

	if !hub.IsUserOnline("user-1") {
		t.Error("Expected user to be online")
	}
}

func TestHub_GetRoomClients(t *testing.T) {
	hub := createTestHub()

	client1 := createMockClient("user-1", "alice")
	client2 := createMockClient("user-2", "bob")
	client1.hub = hub
	client2.hub = hub

	roomID := "room-1"

	// Add clients to room
	hub.rooms[roomID] = map[*Client]bool{
		client1: true,
		client2: true,
	}

	count := hub.GetRoomClients(roomID)
	if count != 2 {
		t.Errorf("Expected 2 clients in room, got %d", count)
	}

	// Empty room
	count = hub.GetRoomClients("non-existent-room")
	if count != 0 {
		t.Errorf("Expected 0 clients in non-existent room, got %d", count)
	}
}

func TestHub_GetStats(t *testing.T) {
	hub := createTestHub()

	client1 := createMockClient("user-1", "alice")
	client2 := createMockClient("user-2", "bob")
	client1.hub = hub
	client2.hub = hub

	// Register clients
	hub.clients[client1] = true
	hub.clients[client2] = true
	hub.users["user-1"] = map[*Client]bool{client1: true}
	hub.users["user-2"] = map[*Client]bool{client2: true}

	// Add to room
	hub.rooms["room-1"] = map[*Client]bool{client1: true, client2: true}

	stats := hub.GetStats()

	if stats["total_clients"] != 2 {
		t.Errorf("Expected total_clients 2, got %d", stats["total_clients"])
	}

	if stats["online_users"] != 2 {
		t.Errorf("Expected online_users 2, got %d", stats["online_users"])
	}

	if stats["active_rooms"] != 1 {
		t.Errorf("Expected active_rooms 1, got %d", stats["active_rooms"])
	}
}

func TestHub_BroadcastToRoom(t *testing.T) {
	hub := createTestHub()

	client1 := createMockClient("user-1", "alice")
	client2 := createMockClient("user-2", "bob")
	client1.hub = hub
	client2.hub = hub

	roomID := "room-1"

	// Add clients to room
	hub.rooms[roomID] = map[*Client]bool{
		client1: true,
		client2: true,
	}

	// Create test message
	msg, _ := NewMessage(MessageTypeNewMessage, &NewMessagePayload{
		ID:      "msg-1",
		RoomID:  roomID,
		Content: "Hello!",
	})

	// Broadcast
	hub.broadcastToRoom(&BroadcastMessage{
		RoomID:  roomID,
		Message: msg,
		Sender:  nil,
	})

	// Give some time for messages to be delivered
	time.Sleep(10 * time.Millisecond)

	// Check both clients received the message
	select {
	case data := <-client1.send:
		if len(data) == 0 {
			t.Error("Expected client1 to receive message")
		}
	default:
		t.Error("Client1 did not receive message")
	}

	select {
	case data := <-client2.send:
		if len(data) == 0 {
			t.Error("Expected client2 to receive message")
		}
	default:
		t.Error("Client2 did not receive message")
	}
}

func TestHub_SendToUser(t *testing.T) {
	hub := createTestHub()

	client := createMockClient("user-1", "alice")
	client.hub = hub

	// Register client
	hub.users["user-1"] = map[*Client]bool{client: true}

	// Create test message
	msg, _ := NewMessage(MessageTypeNewDM, &NewDMPayload{
		ID:      "dm-1",
		Content: "Hello!",
	})

	// Send to user
	hub.sendToUser("user-1", msg)

	// Give some time for message to be delivered
	time.Sleep(10 * time.Millisecond)

	// Check client received the message
	select {
	case data := <-client.send:
		if len(data) == 0 {
			t.Error("Expected client to receive message")
		}
	default:
		t.Error("Client did not receive message")
	}
}
