package response

import (
	"time"

	"github.com/go-demo/chat/internal/model"
)

// TokenResponse represents token response
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// UserResponse represents a user response
type UserResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Status      string `json:"status"`
	Bio         string `json:"bio"`
	CreatedAt   string `json:"created_at"`
}

// NewUserResponse creates a user response from model
func NewUserResponse(user *model.User, includeEmail bool) *UserResponse {
	resp := &UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.GetDisplayName(),
		AvatarURL:   user.GetAvatarURL(),
		Status:      string(user.Status),
		Bio:         user.GetBio(),
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	}
	if includeEmail {
		resp.Email = user.Email
	}
	return resp
}

// AuthResponse represents authentication response
type AuthResponse struct {
	User  *UserResponse  `json:"user"`
	Token *TokenResponse `json:"token"`
}

// ProfileResponse represents user profile response
type ProfileResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Status      string `json:"status"`
	Bio         string `json:"bio"`
}

// NewProfileResponse creates a profile response from model
func NewProfileResponse(profile *model.UserProfile) *ProfileResponse {
	return &ProfileResponse{
		ID:          profile.ID,
		Username:    profile.Username,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Status:      string(profile.Status),
		Bio:         profile.Bio,
	}
}

// FriendResponse represents a friend response
type FriendResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Status      string `json:"status"`
	FriendSince string `json:"friend_since"`
}

// NewFriendResponse creates a friend response from model
func NewFriendResponse(f *model.FriendshipWithUser) *FriendResponse {
	displayName := f.FriendUsername
	if f.FriendDisplayName.Valid && f.FriendDisplayName.String != "" {
		displayName = f.FriendDisplayName.String
	}

	avatarURL := ""
	if f.FriendAvatarURL.Valid {
		avatarURL = f.FriendAvatarURL.String
	}

	return &FriendResponse{
		ID:          f.FriendID,
		Username:    f.FriendUsername,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		Status:      string(f.FriendStatus),
		FriendSince: f.CreatedAt.Format(time.RFC3339),
	}
}

// FriendRequestResponse represents a friend request response
type FriendRequestResponse struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Status      string `json:"status"`
	RequestedAt string `json:"requested_at"`
}

// NewFriendRequestResponse creates a friend request response
func NewFriendRequestResponse(f *model.FriendshipWithUser) *FriendRequestResponse {
	displayName := f.FriendUsername
	if f.FriendDisplayName.Valid && f.FriendDisplayName.String != "" {
		displayName = f.FriendDisplayName.String
	}

	avatarURL := ""
	if f.FriendAvatarURL.Valid {
		avatarURL = f.FriendAvatarURL.String
	}

	return &FriendRequestResponse{
		ID:          f.ID,
		UserID:      f.UserID,
		Username:    f.FriendUsername,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		Status:      string(f.Status),
		RequestedAt: f.CreatedAt.Format(time.RFC3339),
	}
}
