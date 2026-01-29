package model

import (
	"database/sql"
	"time"
)

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

type RoomMember struct {
	ID         string         `db:"id" json:"id"`
	RoomID     string         `db:"room_id" json:"room_id"`
	UserID     string         `db:"user_id" json:"user_id"`
	Role       MemberRole     `db:"role" json:"role"`
	Nickname   sql.NullString `db:"nickname" json:"nickname,omitempty"`
	JoinedAt   time.Time      `db:"joined_at" json:"joined_at"`
	LastReadAt time.Time      `db:"last_read_at" json:"last_read_at"`
	IsMuted    bool           `db:"is_muted" json:"is_muted"`
}

// GetNickname returns nickname or empty string
func (rm *RoomMember) GetNickname() string {
	if rm.Nickname.Valid {
		return rm.Nickname.String
	}
	return ""
}

// IsOwner checks if member is room owner
func (rm *RoomMember) IsOwner() bool {
	return rm.Role == MemberRoleOwner
}

// IsAdmin checks if member is room admin
func (rm *RoomMember) IsAdmin() bool {
	return rm.Role == MemberRoleAdmin
}

// CanModerate checks if member can moderate (owner or admin)
func (rm *RoomMember) CanModerate() bool {
	return rm.Role == MemberRoleOwner || rm.Role == MemberRoleAdmin
}

// RoomMemberWithUser includes user info
type RoomMemberWithUser struct {
	RoomMember
	Username    string         `db:"username" json:"username"`
	DisplayName sql.NullString `db:"display_name" json:"display_name,omitempty"`
	AvatarURL   sql.NullString `db:"avatar_url" json:"avatar_url,omitempty"`
	Status      UserStatus     `db:"status" json:"status"`
}

// GetUserDisplayName returns display_name, nickname, or username
func (rm *RoomMemberWithUser) GetUserDisplayName() string {
	// Priority: nickname > display_name > username
	if rm.Nickname.Valid && rm.Nickname.String != "" {
		return rm.Nickname.String
	}
	if rm.DisplayName.Valid && rm.DisplayName.String != "" {
		return rm.DisplayName.String
	}
	return rm.Username
}

// GetUserAvatarURL returns avatar_url or empty string
func (rm *RoomMemberWithUser) GetUserAvatarURL() string {
	if rm.AvatarURL.Valid {
		return rm.AvatarURL.String
	}
	return ""
}
