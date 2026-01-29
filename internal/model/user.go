package model

import (
	"database/sql"
	"time"
)

type UserStatus string

const (
	UserStatusOnline  UserStatus = "online"
	UserStatusOffline UserStatus = "offline"
	UserStatusAway    UserStatus = "away"
	UserStatusBusy    UserStatus = "busy"
)

type User struct {
	ID           string         `db:"id" json:"id"`
	Username     string         `db:"username" json:"username"`
	Email        string         `db:"email" json:"email"`
	PasswordHash string         `db:"password_hash" json:"-"`
	DisplayName  sql.NullString `db:"display_name" json:"display_name,omitempty"`
	AvatarURL    sql.NullString `db:"avatar_url" json:"avatar_url,omitempty"`
	Status       UserStatus     `db:"status" json:"status"`
	Bio          sql.NullString `db:"bio" json:"bio,omitempty"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
	LastSeenAt   sql.NullTime   `db:"last_seen_at" json:"last_seen_at,omitempty"`
}

// GetDisplayName returns display_name or username as fallback
func (u *User) GetDisplayName() string {
	if u.DisplayName.Valid && u.DisplayName.String != "" {
		return u.DisplayName.String
	}
	return u.Username
}

// GetAvatarURL returns avatar_url or empty string
func (u *User) GetAvatarURL() string {
	if u.AvatarURL.Valid {
		return u.AvatarURL.String
	}
	return ""
}

// GetBio returns bio or empty string
func (u *User) GetBio() string {
	if u.Bio.Valid {
		return u.Bio.String
	}
	return ""
}

// IsOnline checks if user is online
func (u *User) IsOnline() bool {
	return u.Status == UserStatusOnline
}

// UserProfile is a public-facing user profile
type UserProfile struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	AvatarURL   string     `json:"avatar_url"`
	Status      UserStatus `json:"status"`
	Bio         string     `json:"bio"`
}

// ToProfile converts User to UserProfile
func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.GetDisplayName(),
		AvatarURL:   u.GetAvatarURL(),
		Status:      u.Status,
		Bio:         u.GetBio(),
	}
}
