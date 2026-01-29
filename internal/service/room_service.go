package service

import (
	"context"
	"database/sql"

	"github.com/go-demo/chat/internal/model"
	apperrors "github.com/go-demo/chat/internal/pkg/errors"
	"github.com/go-demo/chat/internal/repository"
	"go.uber.org/zap"
)

type RoomService struct {
	roomRepo    *repository.RoomRepository
	userRepo    *repository.UserRepository
	messageRepo *repository.MessageRepository
	logger      *zap.Logger
}

func NewRoomService(
	roomRepo *repository.RoomRepository,
	userRepo *repository.UserRepository,
	messageRepo *repository.MessageRepository,
	logger *zap.Logger,
) *RoomService {
	return &RoomService{
		roomRepo:    roomRepo,
		userRepo:    userRepo,
		messageRepo: messageRepo,
		logger:      logger,
	}
}

// CreateRoomInput represents room creation input
type CreateRoomInput struct {
	Name        string
	Description string
	Type        model.RoomType
	OwnerID     string
	MaxMembers  int
}

// Create creates a new room
func (s *RoomService) Create(ctx context.Context, input *CreateRoomInput) (*model.Room, error) {
	// Set default max members
	if input.MaxMembers <= 0 {
		input.MaxMembers = 100
	}

	room := &model.Room{
		Name:       input.Name,
		Type:       input.Type,
		OwnerID:    input.OwnerID,
		MaxMembers: input.MaxMembers,
	}

	if input.Description != "" {
		room.Description = sql.NullString{String: input.Description, Valid: true}
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		s.logger.Error("Failed to create room", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	// Add owner as member with owner role
	member := &model.RoomMember{
		RoomID: room.ID,
		UserID: input.OwnerID,
		Role:   model.MemberRoleOwner,
	}

	if err := s.roomRepo.AddMember(ctx, member); err != nil {
		s.logger.Error("Failed to add owner as member", zap.Error(err))
		// Delete the room if we can't add the owner
		_ = s.roomRepo.Delete(ctx, room.ID)
		return nil, apperrors.ErrInternal
	}

	s.logger.Info("Room created",
		zap.String("room_id", room.ID),
		zap.String("name", room.Name),
		zap.String("owner_id", input.OwnerID),
	)

	return room, nil
}

// GetByID retrieves a room by ID
func (s *RoomService) GetByID(ctx context.Context, id string) (*model.Room, error) {
	room, err := s.roomRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			return nil, apperrors.ErrRoomNotFound
		}
		s.logger.Error("Failed to get room", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return room, nil
}

// GetByIDWithDetails retrieves a room with member count and owner info
func (s *RoomService) GetByIDWithDetails(ctx context.Context, id string) (*model.RoomDetail, error) {
	room, err := s.roomRepo.GetByIDWithMemberCount(ctx, id)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			return nil, apperrors.ErrRoomNotFound
		}
		s.logger.Error("Failed to get room", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	owner, err := s.userRepo.GetByID(ctx, room.OwnerID)
	if err != nil {
		s.logger.Warn("Failed to get room owner", zap.Error(err))
	}

	detail := &model.RoomDetail{
		Room:        room.Room,
		MemberCount: room.MemberCount,
	}

	if owner != nil {
		detail.Owner = owner.ToProfile()
	}

	return detail, nil
}

// UpdateRoomInput represents room update input
type UpdateRoomInput struct {
	RoomID      string
	UserID      string
	Name        *string
	Description *string
	MaxMembers  *int
}

// Update updates a room
func (s *RoomService) Update(ctx context.Context, input *UpdateRoomInput) (*model.Room, error) {
	room, err := s.roomRepo.GetByID(ctx, input.RoomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			return nil, apperrors.ErrRoomNotFound
		}
		return nil, apperrors.ErrInternal
	}

	// Check permission (only owner can update)
	if room.OwnerID != input.UserID {
		// Check if user is admin
		member, err := s.roomRepo.GetMember(ctx, input.RoomID, input.UserID)
		if err != nil || !member.CanModerate() {
			return nil, apperrors.ErrPermissionDenied
		}
	}

	// Update fields
	if input.Name != nil {
		room.Name = *input.Name
	}
	if input.Description != nil {
		room.Description = sql.NullString{String: *input.Description, Valid: *input.Description != ""}
	}
	if input.MaxMembers != nil && *input.MaxMembers > 0 {
		room.MaxMembers = *input.MaxMembers
	}

	if err := s.roomRepo.Update(ctx, room); err != nil {
		s.logger.Error("Failed to update room", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return room, nil
}

// Delete deletes a room
func (s *RoomService) Delete(ctx context.Context, roomID, userID string) error {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			return apperrors.ErrRoomNotFound
		}
		return apperrors.ErrInternal
	}

	// Only owner can delete
	if room.OwnerID != userID {
		return apperrors.ErrPermissionDenied
	}

	if err := s.roomRepo.Delete(ctx, roomID); err != nil {
		s.logger.Error("Failed to delete room", zap.Error(err))
		return apperrors.ErrInternal
	}

	s.logger.Info("Room deleted",
		zap.String("room_id", roomID),
		zap.String("deleted_by", userID),
	)

	return nil
}

// ListPublic lists public rooms
func (s *RoomService) ListPublic(ctx context.Context, limit, offset int) ([]*model.RoomWithMemberCount, error) {
	rooms, err := s.roomRepo.ListPublic(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list public rooms", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return rooms, nil
}

// ListByUserID lists rooms that user is a member of
func (s *RoomService) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.RoomWithMemberCount, error) {
	rooms, err := s.roomRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list user rooms", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return rooms, nil
}

// Search searches rooms by name
func (s *RoomService) Search(ctx context.Context, query string, limit, offset int) ([]*model.RoomWithMemberCount, error) {
	rooms, err := s.roomRepo.Search(ctx, query, limit, offset)
	if err != nil {
		s.logger.Error("Failed to search rooms", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return rooms, nil
}

// Join joins a room
func (s *RoomService) Join(ctx context.Context, roomID, userID string) error {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			return apperrors.ErrRoomNotFound
		}
		return apperrors.ErrInternal
	}

	// Check if room is private (need invite)
	if room.IsPrivate() {
		return apperrors.ErrPermissionDenied
	}

	member := &model.RoomMember{
		RoomID: roomID,
		UserID: userID,
		Role:   model.MemberRoleMember,
	}

	if err := s.roomRepo.AddMember(ctx, member); err != nil {
		if err == repository.ErrAlreadyRoomMember {
			return apperrors.ErrAlreadyRoomMember
		}
		if err == repository.ErrRoomFull {
			return apperrors.ErrRoomFull
		}
		s.logger.Error("Failed to join room", zap.Error(err))
		return apperrors.ErrInternal
	}

	s.logger.Info("User joined room",
		zap.String("room_id", roomID),
		zap.String("user_id", userID),
	)

	return nil
}

// Leave leaves a room
func (s *RoomService) Leave(ctx context.Context, roomID, userID string) error {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			return apperrors.ErrRoomNotFound
		}
		return apperrors.ErrInternal
	}

	// Owner cannot leave (must transfer ownership or delete room)
	if room.OwnerID == userID {
		return apperrors.New(400, "房主無法離開聊天室，請先轉移所有權或刪除聊天室")
	}

	if err := s.roomRepo.RemoveMember(ctx, roomID, userID); err != nil {
		if err == repository.ErrNotRoomMember {
			return apperrors.ErrNotFound
		}
		s.logger.Error("Failed to leave room", zap.Error(err))
		return apperrors.ErrInternal
	}

	s.logger.Info("User left room",
		zap.String("room_id", roomID),
		zap.String("user_id", userID),
	)

	return nil
}

// InviteMember invites a user to a private room
func (s *RoomService) InviteMember(ctx context.Context, roomID, inviterID, inviteeID string) error {
	// Check if inviter can moderate
	member, err := s.roomRepo.GetMember(ctx, roomID, inviterID)
	if err != nil {
		if err == repository.ErrNotRoomMember {
			return apperrors.ErrPermissionDenied
		}
		return apperrors.ErrInternal
	}

	if !member.CanModerate() {
		return apperrors.ErrPermissionDenied
	}

	// Check if invitee exists
	if _, err := s.userRepo.GetByID(ctx, inviteeID); err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.ErrUserNotFound
		}
		return apperrors.ErrInternal
	}

	newMember := &model.RoomMember{
		RoomID: roomID,
		UserID: inviteeID,
		Role:   model.MemberRoleMember,
	}

	if err := s.roomRepo.AddMember(ctx, newMember); err != nil {
		if err == repository.ErrAlreadyRoomMember {
			return apperrors.ErrAlreadyRoomMember
		}
		if err == repository.ErrRoomFull {
			return apperrors.ErrRoomFull
		}
		return apperrors.ErrInternal
	}

	return nil
}

// KickMember removes a member from a room
func (s *RoomService) KickMember(ctx context.Context, roomID, kickerID, targetID string) error {
	// Check if kicker can moderate
	kicker, err := s.roomRepo.GetMember(ctx, roomID, kickerID)
	if err != nil {
		if err == repository.ErrNotRoomMember {
			return apperrors.ErrPermissionDenied
		}
		return apperrors.ErrInternal
	}

	if !kicker.CanModerate() {
		return apperrors.ErrPermissionDenied
	}

	// Get target member
	target, err := s.roomRepo.GetMember(ctx, roomID, targetID)
	if err != nil {
		if err == repository.ErrNotRoomMember {
			return apperrors.ErrNotFound
		}
		return apperrors.ErrInternal
	}

	// Cannot kick owner or same/higher role
	if target.IsOwner() {
		return apperrors.ErrPermissionDenied
	}
	if kicker.Role == model.MemberRoleAdmin && target.Role == model.MemberRoleAdmin {
		return apperrors.ErrPermissionDenied
	}

	if err := s.roomRepo.RemoveMember(ctx, roomID, targetID); err != nil {
		return apperrors.ErrInternal
	}

	s.logger.Info("User kicked from room",
		zap.String("room_id", roomID),
		zap.String("kicked_by", kickerID),
		zap.String("target", targetID),
	)

	return nil
}

// PromoteMember promotes a member to admin
func (s *RoomService) PromoteMember(ctx context.Context, roomID, promoterID, targetID string) error {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return apperrors.ErrRoomNotFound
	}

	// Only owner can promote
	if room.OwnerID != promoterID {
		return apperrors.ErrPermissionDenied
	}

	if err := s.roomRepo.UpdateMemberRole(ctx, roomID, targetID, model.MemberRoleAdmin); err != nil {
		if err == repository.ErrNotRoomMember {
			return apperrors.ErrNotFound
		}
		return apperrors.ErrInternal
	}

	return nil
}

// DemoteMember demotes an admin to member
func (s *RoomService) DemoteMember(ctx context.Context, roomID, demoterID, targetID string) error {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return apperrors.ErrRoomNotFound
	}

	// Only owner can demote
	if room.OwnerID != demoterID {
		return apperrors.ErrPermissionDenied
	}

	if err := s.roomRepo.UpdateMemberRole(ctx, roomID, targetID, model.MemberRoleMember); err != nil {
		if err == repository.ErrNotRoomMember {
			return apperrors.ErrNotFound
		}
		return apperrors.ErrInternal
	}

	return nil
}

// ListMembers lists all members of a room
func (s *RoomService) ListMembers(ctx context.Context, roomID, userID string) ([]*model.RoomMemberWithUser, error) {
	// Check if user is a member (for private rooms)
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		if err == repository.ErrRoomNotFound {
			return nil, apperrors.ErrRoomNotFound
		}
		return nil, apperrors.ErrInternal
	}

	if room.IsPrivate() {
		isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
		if err != nil {
			return nil, apperrors.ErrInternal
		}
		if !isMember {
			return nil, apperrors.ErrPermissionDenied
		}
	}

	members, err := s.roomRepo.ListMembers(ctx, roomID)
	if err != nil {
		s.logger.Error("Failed to list members", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return members, nil
}

// IsMember checks if user is a member of a room
func (s *RoomService) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	return s.roomRepo.IsMember(ctx, roomID, userID)
}

// GetMember gets a room member
func (s *RoomService) GetMember(ctx context.Context, roomID, userID string) (*model.RoomMember, error) {
	member, err := s.roomRepo.GetMember(ctx, roomID, userID)
	if err != nil {
		if err == repository.ErrNotRoomMember {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.ErrInternal
	}
	return member, nil
}

// UpdateLastRead updates the last read timestamp for a member
func (s *RoomService) UpdateLastRead(ctx context.Context, roomID, userID string) error {
	return s.roomRepo.UpdateLastReadAt(ctx, roomID, userID)
}
