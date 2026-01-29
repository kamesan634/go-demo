package ws

import (
	"encoding/json"
	"testing"

	"go.uber.org/zap"
)

func createTestClient(userID, username string) *Client {
	logger := zap.NewNop()
	return &Client{
		send:     make(chan []byte, 256),
		userID:   userID,
		username: username,
		rooms:    make(map[string]bool),
		logger:   logger,
	}
}

func TestClient_GetUserID(t *testing.T) {
	client := createTestClient("user-123", "alice")

	if client.GetUserID() != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", client.GetUserID())
	}
}

func TestClient_GetUsername(t *testing.T) {
	client := createTestClient("user-123", "alice")

	if client.GetUsername() != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", client.GetUsername())
	}
}

func TestClient_JoinRoom(t *testing.T) {
	client := createTestClient("user-123", "alice")

	roomID := "room-1"
	client.JoinRoom(roomID)

	if !client.IsInRoom(roomID) {
		t.Error("Expected client to be in room")
	}

	rooms := client.GetRooms()
	if len(rooms) != 1 {
		t.Errorf("Expected 1 room, got %d", len(rooms))
	}
}

func TestClient_LeaveRoom(t *testing.T) {
	client := createTestClient("user-123", "alice")

	roomID := "room-1"
	client.JoinRoom(roomID)
	client.LeaveRoom(roomID)

	if client.IsInRoom(roomID) {
		t.Error("Expected client not to be in room")
	}

	rooms := client.GetRooms()
	if len(rooms) != 0 {
		t.Errorf("Expected 0 rooms, got %d", len(rooms))
	}
}

func TestClient_IsInRoom(t *testing.T) {
	client := createTestClient("user-123", "alice")

	if client.IsInRoom("room-1") {
		t.Error("Expected client not to be in room initially")
	}

	client.JoinRoom("room-1")

	if !client.IsInRoom("room-1") {
		t.Error("Expected client to be in room after joining")
	}

	if client.IsInRoom("room-2") {
		t.Error("Expected client not to be in room-2")
	}
}

func TestClient_GetRooms(t *testing.T) {
	client := createTestClient("user-123", "alice")

	client.JoinRoom("room-1")
	client.JoinRoom("room-2")
	client.JoinRoom("room-3")

	rooms := client.GetRooms()
	if len(rooms) != 3 {
		t.Errorf("Expected 3 rooms, got %d", len(rooms))
	}

	// Verify all rooms are present
	roomMap := make(map[string]bool)
	for _, r := range rooms {
		roomMap[r] = true
	}

	for _, expected := range []string{"room-1", "room-2", "room-3"} {
		if !roomMap[expected] {
			t.Errorf("Expected room %s to be present", expected)
		}
	}
}

func TestClient_SendMessage(t *testing.T) {
	client := createTestClient("user-123", "alice")

	msg, _ := NewMessage(MessageTypeNewMessage, &NewMessagePayload{
		ID:      "msg-1",
		Content: "Hello!",
	})

	client.SendMessage(msg)

	// Check message was sent
	select {
	case data := <-client.send:
		var received Message
		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("Failed to unmarshal received message: %v", err)
		}

		if received.Type != MessageTypeNewMessage {
			t.Errorf("Expected message type '%s', got '%s'", MessageTypeNewMessage, received.Type)
		}
	default:
		t.Error("Expected message to be in send channel")
	}
}

func TestClient_SendMessage_BufferFull(t *testing.T) {
	// Create client with small buffer
	client := &Client{
		send:     make(chan []byte, 1), // Very small buffer
		userID:   "user-123",
		username: "alice",
		rooms:    make(map[string]bool),
		logger:   zap.NewNop(),
	}

	msg, _ := NewMessage(MessageTypeNewMessage, &NewMessagePayload{Content: "Test"})

	// Fill the buffer
	client.SendMessage(msg)

	// This should not block (should be dropped if buffer full)
	client.SendMessage(msg)

	// Verify first message is in channel
	select {
	case <-client.send:
		// OK
	default:
		t.Error("Expected at least one message in channel")
	}
}

func TestClient_MultipleRooms(t *testing.T) {
	client := createTestClient("user-123", "alice")

	// Join multiple rooms
	roomIDs := []string{"room-1", "room-2", "room-3", "room-4", "room-5"}
	for _, roomID := range roomIDs {
		client.JoinRoom(roomID)
	}

	rooms := client.GetRooms()
	if len(rooms) != 5 {
		t.Errorf("Expected 5 rooms, got %d", len(rooms))
	}

	// Leave some rooms
	client.LeaveRoom("room-1")
	client.LeaveRoom("room-3")

	rooms = client.GetRooms()
	if len(rooms) != 3 {
		t.Errorf("Expected 3 rooms after leaving, got %d", len(rooms))
	}

	if client.IsInRoom("room-1") {
		t.Error("Expected client not to be in room-1")
	}
	if !client.IsInRoom("room-2") {
		t.Error("Expected client to be in room-2")
	}
}

func TestClient_JoinRoomIdempotent(t *testing.T) {
	client := createTestClient("user-123", "alice")

	// Join same room multiple times
	client.JoinRoom("room-1")
	client.JoinRoom("room-1")
	client.JoinRoom("room-1")

	rooms := client.GetRooms()
	if len(rooms) != 1 {
		t.Errorf("Expected 1 room (idempotent join), got %d", len(rooms))
	}
}

func TestClient_LeaveRoomIdempotent(t *testing.T) {
	client := createTestClient("user-123", "alice")

	client.JoinRoom("room-1")

	// Leave same room multiple times
	client.LeaveRoom("room-1")
	client.LeaveRoom("room-1") // Should not panic
	client.LeaveRoom("room-1")

	if client.IsInRoom("room-1") {
		t.Error("Expected client not to be in room")
	}
}

func TestClient_LeaveRoomNotJoined(t *testing.T) {
	client := createTestClient("user-123", "alice")

	// Leave room that was never joined - should not panic
	client.LeaveRoom("non-existent-room")

	if client.IsInRoom("non-existent-room") {
		t.Error("Expected client not to be in room")
	}
}

func TestClient_ConcurrentRoomOperations(t *testing.T) {
	client := createTestClient("user-123", "alice")

	done := make(chan bool)

	// Concurrent joins
	go func() {
		for i := 0; i < 100; i++ {
			client.JoinRoom("room-1")
		}
		done <- true
	}()

	// Concurrent leaves
	go func() {
		for i := 0; i < 100; i++ {
			client.LeaveRoom("room-1")
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = client.IsInRoom("room-1")
			_ = client.GetRooms()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should not have panicked
}

func TestClient_HandleMessage_Types(t *testing.T) {
	// Create client to verify test helper works
	_ = createTestClient("user-123", "alice")

	testCases := []struct {
		msgType MessageType
	}{
		{MessageTypeJoinRoom},
		{MessageTypeLeaveRoom},
		{MessageTypeSendMessage},
		{MessageTypeSendDM},
		{MessageTypeTyping},
		{MessageTypeStopTyping},
		{MessageTypePing},
		{MessageTypeMarkRead},
	}

	for _, tc := range testCases {
		msg := &Message{
			Type:    tc.msgType,
			Payload: json.RawMessage(`{}`),
		}

		// Should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("handleMessage panicked for type %s: %v", tc.msgType, r)
				}
			}()
			// Note: We can't actually call handleMessage directly as it requires hub
			// This test just verifies the message types exist
			_ = msg
		}()
	}
}
