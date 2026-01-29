package ws

import (
	"encoding/json"
	"time"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Client -> Server messages
	MessageTypeJoinRoom     MessageType = "join_room"
	MessageTypeLeaveRoom    MessageType = "leave_room"
	MessageTypeSendMessage  MessageType = "send_message"
	MessageTypeTyping       MessageType = "typing"
	MessageTypeStopTyping   MessageType = "stop_typing"
	MessageTypePing         MessageType = "ping"
	MessageTypeMarkRead     MessageType = "mark_read"

	// Server -> Client messages
	MessageTypeRoomJoined   MessageType = "room_joined"
	MessageTypeRoomLeft     MessageType = "room_left"
	MessageTypeNewMessage   MessageType = "new_message"
	MessageTypeUserTyping   MessageType = "user_typing"
	MessageTypeUserStopTyping MessageType = "user_stop_typing"
	MessageTypePong         MessageType = "pong"
	MessageTypeUserOnline   MessageType = "user_online"
	MessageTypeUserOffline  MessageType = "user_offline"
	MessageTypeError        MessageType = "error"
	MessageTypeAck          MessageType = "ack"

	// Direct message types
	MessageTypeSendDM       MessageType = "send_dm"
	MessageTypeNewDM        MessageType = "new_dm"
	MessageTypeDMRead       MessageType = "dm_read"

	// Notification types
	MessageTypeNotification MessageType = "notification"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	RequestID string          `json:"request_id,omitempty"`
}

// JoinRoomPayload represents join room payload
type JoinRoomPayload struct {
	RoomID string `json:"room_id"`
}

// LeaveRoomPayload represents leave room payload
type LeaveRoomPayload struct {
	RoomID string `json:"room_id"`
}

// SendMessagePayload represents send message payload
type SendMessagePayload struct {
	RoomID    string `json:"room_id"`
	Content   string `json:"content"`
	Type      string `json:"type,omitempty"` // text, image, file
	ReplyToID string `json:"reply_to_id,omitempty"`
}

// TypingPayload represents typing indicator payload
type TypingPayload struct {
	RoomID string `json:"room_id"`
}

// SendDMPayload represents send direct message payload
type SendDMPayload struct {
	ReceiverID string `json:"receiver_id"`
	Content    string `json:"content"`
	Type       string `json:"type,omitempty"`
}

// MarkReadPayload represents mark as read payload
type MarkReadPayload struct {
	RoomID    string `json:"room_id,omitempty"`
	SenderID  string `json:"sender_id,omitempty"` // For DM
	MessageID string `json:"message_id,omitempty"`
}

// RoomJoinedPayload represents room joined response
type RoomJoinedPayload struct {
	RoomID      string `json:"room_id"`
	RoomName    string `json:"room_name"`
	MemberCount int    `json:"member_count"`
}

// NewMessagePayload represents new message broadcast
type NewMessagePayload struct {
	ID          string `json:"id"`
	RoomID      string `json:"room_id"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	ReplyToID   string `json:"reply_to_id,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// UserTypingPayload represents user typing broadcast
type UserTypingPayload struct {
	RoomID      string `json:"room_id"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// UserStatusPayload represents user online/offline status
type UserStatusPayload struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

// NewDMPayload represents new direct message
type NewDMPayload struct {
	ID                string `json:"id"`
	SenderID          string `json:"sender_id"`
	SenderUsername    string `json:"sender_username"`
	SenderDisplayName string `json:"sender_display_name"`
	SenderAvatarURL   string `json:"sender_avatar_url"`
	Content           string `json:"content"`
	Type              string `json:"type"`
	CreatedAt         string `json:"created_at"`
}

// DMReadPayload represents DM read notification
type DMReadPayload struct {
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	ReadAt     string `json:"read_at"`
}

// ErrorPayload represents error message
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NotificationPayload represents a notification
type NotificationPayload struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Title         string `json:"title"`
	Content       string `json:"content,omitempty"`
	ReferenceID   string `json:"reference_id,omitempty"`
	ReferenceType string `json:"reference_type,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// AckPayload represents acknowledgement
type AckPayload struct {
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
	}, nil
}

// NewErrorMessage creates a new error message
func NewErrorMessage(code int, message string) (*Message, error) {
	return NewMessage(MessageTypeError, &ErrorPayload{
		Code:    code,
		Message: message,
	})
}

// ParsePayload parses message payload into the given type
func (m *Message) ParsePayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}
