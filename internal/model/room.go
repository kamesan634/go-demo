package model

import (
	"database/sql"
	"time"
)

type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
	RoomTypeDirect  RoomType = "direct"
)

type Room struct {
	ID          string         `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Description sql.NullString `db:"description" json:"description,omitempty"`
	Type        RoomType       `db:"type" json:"type"`
	OwnerID     string         `db:"owner_id" json:"owner_id"`
	MaxMembers  int            `db:"max_members" json:"max_members"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
}

// GetDescription returns description or empty string
func (r *Room) GetDescription() string {
	if r.Description.Valid {
		return r.Description.String
	}
	return ""
}

// IsPublic checks if room is public
func (r *Room) IsPublic() bool {
	return r.Type == RoomTypePublic
}

// IsPrivate checks if room is private
func (r *Room) IsPrivate() bool {
	return r.Type == RoomTypePrivate
}

// IsDirect checks if room is for direct messages
func (r *Room) IsDirect() bool {
	return r.Type == RoomTypeDirect
}

// RoomWithMemberCount includes member count
type RoomWithMemberCount struct {
	Room
	MemberCount int `db:"member_count" json:"member_count"`
}

// RoomDetail includes owner info and member count
type RoomDetail struct {
	Room
	MemberCount int          `db:"member_count" json:"member_count"`
	Owner       *UserProfile `json:"owner,omitempty"`
}
