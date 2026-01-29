package model

import (
	"database/sql"
	"time"
)

type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeFile   MessageType = "file"
	MessageTypeSystem MessageType = "system"
)

type Message struct {
	ID        string         `db:"id" json:"id"`
	RoomID    string         `db:"room_id" json:"room_id"`
	UserID    string         `db:"user_id" json:"user_id"`
	Content   string         `db:"content" json:"content"`
	Type      MessageType    `db:"type" json:"type"`
	ReplyToID sql.NullString `db:"reply_to_id" json:"reply_to_id,omitempty"`
	IsEdited  bool           `db:"is_edited" json:"is_edited"`
	IsDeleted bool           `db:"is_deleted" json:"is_deleted"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt time.Time      `db:"updated_at" json:"updated_at"`
}

// GetReplyToID returns reply_to_id or empty string
func (m *Message) GetReplyToID() string {
	if m.ReplyToID.Valid {
		return m.ReplyToID.String
	}
	return ""
}

// MessageWithUser includes user info
type MessageWithUser struct {
	Message
	Username    string         `db:"username" json:"username"`
	DisplayName sql.NullString `db:"display_name" json:"display_name,omitempty"`
	AvatarURL   sql.NullString `db:"avatar_url" json:"avatar_url,omitempty"`
}

// GetUserDisplayName returns display_name or username
func (m *MessageWithUser) GetUserDisplayName() string {
	if m.DisplayName.Valid && m.DisplayName.String != "" {
		return m.DisplayName.String
	}
	return m.Username
}

// GetUserAvatarURL returns avatar_url or empty string
func (m *MessageWithUser) GetUserAvatarURL() string {
	if m.AvatarURL.Valid {
		return m.AvatarURL.String
	}
	return ""
}

// MessageAttachment represents a file attached to a message
type MessageAttachment struct {
	ID        string    `db:"id" json:"id"`
	MessageID string    `db:"message_id" json:"message_id"`
	FileName  string    `db:"file_name" json:"file_name"`
	FileURL   string    `db:"file_url" json:"file_url"`
	FileType  string    `db:"file_type" json:"file_type"`
	FileSize  int64     `db:"file_size" json:"file_size"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// MessageDetail includes attachments and reply info
type MessageDetail struct {
	MessageWithUser
	Attachments []*MessageAttachment `json:"attachments,omitempty"`
	ReplyTo     *MessageWithUser     `json:"reply_to,omitempty"`
}
