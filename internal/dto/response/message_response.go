package response

import (
	"time"

	"github.com/go-demo/chat/internal/model"
)

// MessageResponse represents a message response
type MessageResponse struct {
	ID          string                `json:"id"`
	RoomID      string                `json:"room_id"`
	UserID      string                `json:"user_id"`
	Username    string                `json:"username"`
	DisplayName string                `json:"display_name"`
	AvatarURL   string                `json:"avatar_url"`
	Content     string                `json:"content"`
	Type        string                `json:"type"`
	ReplyToID   string                `json:"reply_to_id,omitempty"`
	IsEdited    bool                  `json:"is_edited"`
	IsDeleted   bool                  `json:"is_deleted"`
	Attachments []*AttachmentResponse `json:"attachments,omitempty"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
}

// NewMessageResponse creates a message response from model
func NewMessageResponse(m *model.MessageWithUser) *MessageResponse {
	displayName := m.Username
	if m.DisplayName.Valid && m.DisplayName.String != "" {
		displayName = m.DisplayName.String
	}

	avatarURL := ""
	if m.AvatarURL.Valid {
		avatarURL = m.AvatarURL.String
	}

	replyToID := ""
	if m.ReplyToID.Valid {
		replyToID = m.ReplyToID.String
	}

	return &MessageResponse{
		ID:          m.ID,
		RoomID:      m.RoomID,
		UserID:      m.UserID,
		Username:    m.Username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		Content:     m.Content,
		Type:        string(m.Type),
		ReplyToID:   replyToID,
		IsEdited:    m.IsEdited,
		IsDeleted:   m.IsDeleted,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
	}
}

// AttachmentResponse represents a message attachment response
type AttachmentResponse struct {
	ID        string `json:"id"`
	FileName  string `json:"file_name"`
	FileURL   string `json:"file_url"`
	FileType  string `json:"file_type"`
	FileSize  int64  `json:"file_size"`
	CreatedAt string `json:"created_at"`
}

// NewAttachmentResponse creates an attachment response from model
func NewAttachmentResponse(a *model.MessageAttachment) *AttachmentResponse {
	return &AttachmentResponse{
		ID:        a.ID,
		FileName:  a.FileName,
		FileURL:   a.FileURL,
		FileType:  a.FileType,
		FileSize:  a.FileSize,
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
	}
}

// DirectMessageResponse represents a direct message response
type DirectMessageResponse struct {
	ID                string `json:"id"`
	SenderID          string `json:"sender_id"`
	ReceiverID        string `json:"receiver_id"`
	SenderUsername    string `json:"sender_username"`
	SenderDisplayName string `json:"sender_display_name"`
	SenderAvatarURL   string `json:"sender_avatar_url"`
	Content           string `json:"content"`
	Type              string `json:"type"`
	IsRead            bool   `json:"is_read"`
	CreatedAt         string `json:"created_at"`
}

// NewDirectMessageResponse creates a direct message response from model
func NewDirectMessageResponse(m *model.DirectMessageWithUser) *DirectMessageResponse {
	senderDisplayName := m.SenderUsername
	if m.SenderDisplayName.Valid && m.SenderDisplayName.String != "" {
		senderDisplayName = m.SenderDisplayName.String
	}

	senderAvatarURL := ""
	if m.SenderAvatarURL.Valid {
		senderAvatarURL = m.SenderAvatarURL.String
	}

	return &DirectMessageResponse{
		ID:                m.ID,
		SenderID:          m.SenderID,
		ReceiverID:        m.ReceiverID,
		SenderUsername:    m.SenderUsername,
		SenderDisplayName: senderDisplayName,
		SenderAvatarURL:   senderAvatarURL,
		Content:           m.Content,
		Type:              string(m.Type),
		IsRead:            m.IsRead,
		CreatedAt:         m.CreatedAt.Format(time.RFC3339),
	}
}

// ConversationResponse represents a conversation response
type ConversationResponse struct {
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	DisplayName   string `json:"display_name"`
	AvatarURL     string `json:"avatar_url"`
	Status        string `json:"status"`
	LastMessage   string `json:"last_message"`
	LastMessageAt string `json:"last_message_at"`
	UnreadCount   int    `json:"unread_count"`
}

// NewConversationResponse creates a conversation response from model
func NewConversationResponse(c *model.Conversation) *ConversationResponse {
	return &ConversationResponse{
		UserID:        c.UserID,
		Username:      c.Username,
		DisplayName:   c.DisplayName,
		AvatarURL:     c.AvatarURL,
		Status:        c.Status,
		LastMessage:   c.LastMessage,
		LastMessageAt: c.LastMessageAt.Format(time.RFC3339),
		UnreadCount:   c.UnreadCount,
	}
}

// MessageListResponse represents a list of messages
type MessageListResponse struct {
	Messages []*MessageResponse `json:"messages"`
	Total    int                `json:"total"`
	HasMore  bool               `json:"has_more"`
}

// NewMessageListResponse creates a message list response
func NewMessageListResponse(messages []*model.MessageWithUser, total int, hasMore bool) *MessageListResponse {
	messageResponses := make([]*MessageResponse, len(messages))
	for i, msg := range messages {
		messageResponses[i] = NewMessageResponse(msg)
	}

	return &MessageListResponse{
		Messages: messageResponses,
		Total:    total,
		HasMore:  hasMore,
	}
}
