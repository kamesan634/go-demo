package model

import (
	"database/sql"
	"time"
)

type DirectMessage struct {
	ID                  string      `db:"id" json:"id"`
	SenderID            string      `db:"sender_id" json:"sender_id"`
	ReceiverID          string      `db:"receiver_id" json:"receiver_id"`
	Content             string      `db:"content" json:"content"`
	Type                MessageType `db:"type" json:"type"`
	IsRead              bool        `db:"is_read" json:"is_read"`
	IsDeletedBySender   bool        `db:"is_deleted_by_sender" json:"-"`
	IsDeletedByReceiver bool        `db:"is_deleted_by_receiver" json:"-"`
	CreatedAt           time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time   `db:"updated_at" json:"updated_at"`
}

// DirectMessageWithUser includes sender info
type DirectMessageWithUser struct {
	DirectMessage
	SenderUsername    string         `db:"sender_username" json:"sender_username"`
	SenderDisplayName sql.NullString `db:"sender_display_name" json:"sender_display_name,omitempty"`
	SenderAvatarURL   sql.NullString `db:"sender_avatar_url" json:"sender_avatar_url,omitempty"`
}

// GetSenderDisplayName returns sender display_name or username
func (dm *DirectMessageWithUser) GetSenderDisplayName() string {
	if dm.SenderDisplayName.Valid && dm.SenderDisplayName.String != "" {
		return dm.SenderDisplayName.String
	}
	return dm.SenderUsername
}

// GetSenderAvatarURL returns sender avatar_url or empty string
func (dm *DirectMessageWithUser) GetSenderAvatarURL() string {
	if dm.SenderAvatarURL.Valid {
		return dm.SenderAvatarURL.String
	}
	return ""
}

// Conversation represents a direct message conversation with another user
type Conversation struct {
	UserID        string    `db:"user_id" json:"user_id"`
	Username      string    `db:"username" json:"username"`
	DisplayName   string    `db:"display_name" json:"display_name"`
	AvatarURL     string    `db:"avatar_url" json:"avatar_url"`
	Status        string    `db:"status" json:"status"`
	LastMessage   string    `db:"last_message" json:"last_message"`
	LastMessageAt time.Time `db:"last_message_at" json:"last_message_at"`
	UnreadCount   int       `db:"unread_count" json:"unread_count"`
}

// BlockedUser represents a blocked user relationship
type BlockedUser struct {
	ID        string    `db:"id" json:"id"`
	BlockerID string    `db:"blocker_id" json:"blocker_id"`
	BlockedID string    `db:"blocked_id" json:"blocked_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Friendship represents a friend relationship
type Friendship struct {
	ID        string           `db:"id" json:"id"`
	UserID    string           `db:"user_id" json:"user_id"`
	FriendID  string           `db:"friend_id" json:"friend_id"`
	Status    FriendshipStatus `db:"status" json:"status"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
}

type FriendshipStatus string

const (
	FriendshipStatusPending  FriendshipStatus = "pending"
	FriendshipStatusAccepted FriendshipStatus = "accepted"
	FriendshipStatusRejected FriendshipStatus = "rejected"
)

// FriendshipWithUser includes friend info
type FriendshipWithUser struct {
	Friendship
	FriendUsername    string         `db:"friend_username" json:"friend_username"`
	FriendDisplayName sql.NullString `db:"friend_display_name" json:"friend_display_name,omitempty"`
	FriendAvatarURL   sql.NullString `db:"friend_avatar_url" json:"friend_avatar_url,omitempty"`
	FriendStatus      UserStatus     `db:"friend_status" json:"friend_status"`
}

// GetFriendDisplayName returns friend display_name or username
func (f *FriendshipWithUser) GetFriendDisplayName() string {
	if f.FriendDisplayName.Valid && f.FriendDisplayName.String != "" {
		return f.FriendDisplayName.String
	}
	return f.FriendUsername
}

// Notification represents a user notification
type Notification struct {
	ID            string         `db:"id" json:"id"`
	UserID        string         `db:"user_id" json:"user_id"`
	Type          string         `db:"type" json:"type"`
	Title         string         `db:"title" json:"title"`
	Content       sql.NullString `db:"content" json:"content,omitempty"`
	ReferenceID   sql.NullString `db:"reference_id" json:"reference_id,omitempty"`
	ReferenceType sql.NullString `db:"reference_type" json:"reference_type,omitempty"`
	IsRead        bool           `db:"is_read" json:"is_read"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
}
