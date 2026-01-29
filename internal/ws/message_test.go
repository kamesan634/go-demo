package ws

import (
	"encoding/json"
	"testing"
)

func TestNewMessage(t *testing.T) {
	payload := &JoinRoomPayload{RoomID: "room-123"}

	msg, err := NewMessage(MessageTypeJoinRoom, payload)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	if msg.Type != MessageTypeJoinRoom {
		t.Errorf("Expected type %s, got %s", MessageTypeJoinRoom, msg.Type)
	}

	if msg.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if len(msg.Payload) == 0 {
		t.Error("Expected payload to be set")
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg, err := NewErrorMessage(400, "Bad Request")
	if err != nil {
		t.Fatalf("Failed to create error message: %v", err)
	}

	if msg.Type != MessageTypeError {
		t.Errorf("Expected type %s, got %s", MessageTypeError, msg.Type)
	}

	var payload ErrorPayload
	if err := msg.ParsePayload(&payload); err != nil {
		t.Fatalf("Failed to parse payload: %v", err)
	}

	if payload.Code != 400 {
		t.Errorf("Expected code 400, got %d", payload.Code)
	}

	if payload.Message != "Bad Request" {
		t.Errorf("Expected message 'Bad Request', got '%s'", payload.Message)
	}
}

func TestMessage_ParsePayload(t *testing.T) {
	original := &SendMessagePayload{
		RoomID:    "room-123",
		Content:   "Hello, World!",
		Type:      "text",
		ReplyToID: "msg-456",
	}

	msg, _ := NewMessage(MessageTypeSendMessage, original)

	var parsed SendMessagePayload
	if err := msg.ParsePayload(&parsed); err != nil {
		t.Fatalf("Failed to parse payload: %v", err)
	}

	if parsed.RoomID != original.RoomID {
		t.Errorf("Expected RoomID %s, got %s", original.RoomID, parsed.RoomID)
	}

	if parsed.Content != original.Content {
		t.Errorf("Expected Content %s, got %s", original.Content, parsed.Content)
	}

	if parsed.Type != original.Type {
		t.Errorf("Expected Type %s, got %s", original.Type, parsed.Type)
	}

	if parsed.ReplyToID != original.ReplyToID {
		t.Errorf("Expected ReplyToID %s, got %s", original.ReplyToID, parsed.ReplyToID)
	}
}

func TestMessage_JSONSerialization(t *testing.T) {
	payload := &NewMessagePayload{
		ID:          "msg-123",
		RoomID:      "room-456",
		UserID:      "user-789",
		Username:    "testuser",
		DisplayName: "Test User",
		Content:     "Hello!",
		Type:        "text",
		CreatedAt:   "2024-01-01T00:00:00Z",
	}

	msg, _ := NewMessage(MessageTypeNewMessage, payload)

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Deserialize
	var parsed Message
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if parsed.Type != msg.Type {
		t.Errorf("Expected type %s, got %s", msg.Type, parsed.Type)
	}
}

func TestMessageTypes(t *testing.T) {
	expectedTypes := []MessageType{
		MessageTypeJoinRoom,
		MessageTypeLeaveRoom,
		MessageTypeSendMessage,
		MessageTypeTyping,
		MessageTypeStopTyping,
		MessageTypePing,
		MessageTypeMarkRead,
		MessageTypeRoomJoined,
		MessageTypeRoomLeft,
		MessageTypeNewMessage,
		MessageTypeUserTyping,
		MessageTypeUserStopTyping,
		MessageTypePong,
		MessageTypeUserOnline,
		MessageTypeUserOffline,
		MessageTypeError,
		MessageTypeAck,
		MessageTypeSendDM,
		MessageTypeNewDM,
		MessageTypeDMRead,
		MessageTypeNotification,
	}

	for _, msgType := range expectedTypes {
		if msgType == "" {
			t.Error("Message type should not be empty")
		}
	}
}

func TestJoinRoomPayload(t *testing.T) {
	payload := JoinRoomPayload{RoomID: "room-123"}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed JoinRoomPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.RoomID != payload.RoomID {
		t.Errorf("Expected %s, got %s", payload.RoomID, parsed.RoomID)
	}
}

func TestSendMessagePayload(t *testing.T) {
	payload := SendMessagePayload{
		RoomID:    "room-123",
		Content:   "Hello!",
		Type:      "text",
		ReplyToID: "msg-456",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed SendMessagePayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.RoomID != payload.RoomID {
		t.Errorf("Expected RoomID %s, got %s", payload.RoomID, parsed.RoomID)
	}
	if parsed.Content != payload.Content {
		t.Errorf("Expected Content %s, got %s", payload.Content, parsed.Content)
	}
}

func TestSendDMPayload(t *testing.T) {
	payload := SendDMPayload{
		ReceiverID: "user-456",
		Content:    "Private message",
		Type:       "text",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed SendDMPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.ReceiverID != payload.ReceiverID {
		t.Errorf("Expected ReceiverID %s, got %s", payload.ReceiverID, parsed.ReceiverID)
	}
}

func TestAckPayload(t *testing.T) {
	payload := AckPayload{
		RequestID: "req-123",
		Success:   true,
		MessageID: "msg-456",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed AckPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.RequestID != payload.RequestID {
		t.Errorf("Expected RequestID %s, got %s", payload.RequestID, parsed.RequestID)
	}
	if parsed.Success != payload.Success {
		t.Errorf("Expected Success %v, got %v", payload.Success, parsed.Success)
	}
}

func TestUserStatusPayload(t *testing.T) {
	payload := UserStatusPayload{
		UserID:      "user-123",
		Username:    "testuser",
		DisplayName: "Test User",
		Status:      "online",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed UserStatusPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.UserID != payload.UserID {
		t.Errorf("Expected UserID %s, got %s", payload.UserID, parsed.UserID)
	}
	if parsed.Status != payload.Status {
		t.Errorf("Expected Status %s, got %s", payload.Status, parsed.Status)
	}
}
