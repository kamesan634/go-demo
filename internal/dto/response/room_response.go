package response

import (
	"time"

	"github.com/go-demo/chat/internal/model"
)

// RoomResponse represents a room response
type RoomResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	OwnerID     string `json:"owner_id"`
	MaxMembers  int    `json:"max_members"`
	MemberCount int    `json:"member_count"`
	CreatedAt   string `json:"created_at"`
}

// NewRoomResponse creates a room response from model
func NewRoomResponse(room *model.RoomWithMemberCount) *RoomResponse {
	description := ""
	if room.Description.Valid {
		description = room.Description.String
	}

	return &RoomResponse{
		ID:          room.ID,
		Name:        room.Name,
		Description: description,
		Type:        string(room.Type),
		OwnerID:     room.OwnerID,
		MaxMembers:  room.MaxMembers,
		MemberCount: room.MemberCount,
		CreatedAt:   room.CreatedAt.Format(time.RFC3339),
	}
}

// RoomDetailResponse represents a detailed room response
type RoomDetailResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Type        string           `json:"type"`
	Owner       *ProfileResponse `json:"owner"`
	MaxMembers  int              `json:"max_members"`
	MemberCount int              `json:"member_count"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

// NewRoomDetailResponse creates a detailed room response from model
func NewRoomDetailResponse(room *model.RoomDetail) *RoomDetailResponse {
	description := ""
	if room.Description.Valid {
		description = room.Description.String
	}

	resp := &RoomDetailResponse{
		ID:          room.ID,
		Name:        room.Name,
		Description: description,
		Type:        string(room.Type),
		MaxMembers:  room.MaxMembers,
		MemberCount: room.MemberCount,
		CreatedAt:   room.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   room.UpdatedAt.Format(time.RFC3339),
	}

	if room.Owner != nil {
		resp.Owner = NewProfileResponse(room.Owner)
	}

	return resp
}

// RoomMemberResponse represents a room member response
type RoomMemberResponse struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Role        string `json:"role"`
	Nickname    string `json:"nickname,omitempty"`
	Status      string `json:"status"`
	JoinedAt    string `json:"joined_at"`
}

// NewRoomMemberResponse creates a room member response from model
func NewRoomMemberResponse(m *model.RoomMemberWithUser) *RoomMemberResponse {
	displayName := m.Username
	if m.DisplayName.Valid && m.DisplayName.String != "" {
		displayName = m.DisplayName.String
	}

	avatarURL := ""
	if m.AvatarURL.Valid {
		avatarURL = m.AvatarURL.String
	}

	nickname := ""
	if m.Nickname.Valid {
		nickname = m.Nickname.String
	}

	return &RoomMemberResponse{
		ID:          m.ID,
		UserID:      m.UserID,
		Username:    m.Username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		Role:        string(m.Role),
		Nickname:    nickname,
		Status:      string(m.Status),
		JoinedAt:    m.JoinedAt.Format(time.RFC3339),
	}
}

// RoomListResponse represents a list of rooms
type RoomListResponse struct {
	Rooms      []*RoomResponse `json:"rooms"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}

// NewRoomListResponse creates a room list response
func NewRoomListResponse(rooms []*model.RoomWithMemberCount, total, page, limit int) *RoomListResponse {
	roomResponses := make([]*RoomResponse, len(rooms))
	for i, room := range rooms {
		roomResponses[i] = NewRoomResponse(room)
	}

	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return &RoomListResponse{
		Rooms:      roomResponses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}
