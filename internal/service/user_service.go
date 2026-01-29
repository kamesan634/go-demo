package service

import (
	"context"
	"database/sql"

	"github.com/go-demo/chat/internal/model"
	apperrors "github.com/go-demo/chat/internal/pkg/errors"
	"github.com/go-demo/chat/internal/repository"
	"go.uber.org/zap"
)

type UserService struct {
	userRepo       *repository.UserRepository
	blockedRepo    *repository.BlockedUserRepository
	friendshipRepo *repository.FriendshipRepository
	logger         *zap.Logger
}

func NewUserService(
	userRepo *repository.UserRepository,
	blockedRepo *repository.BlockedUserRepository,
	friendshipRepo *repository.FriendshipRepository,
	logger *zap.Logger,
) *UserService {
	return &UserService{
		userRepo:       userRepo,
		blockedRepo:    blockedRepo,
		friendshipRepo: friendshipRepo,
		logger:         logger,
	}
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.ErrUserNotFound
		}
		s.logger.Error("Failed to get user", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return user, nil
}

// GetProfile retrieves a user's public profile
func (s *UserService) GetProfile(ctx context.Context, id string) (*model.UserProfile, error) {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user.ToProfile(), nil
}

// UpdateProfileInput represents profile update input
type UpdateProfileInput struct {
	UserID      string
	DisplayName *string
	AvatarURL   *string
	Bio         *string
}

// UpdateProfile updates a user's profile
func (s *UserService) UpdateProfile(ctx context.Context, input *UpdateProfileInput) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.ErrUserNotFound
		}
		s.logger.Error("Failed to get user", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	// Update fields if provided
	if input.DisplayName != nil {
		user.DisplayName = sql.NullString{String: *input.DisplayName, Valid: *input.DisplayName != ""}
	}
	if input.AvatarURL != nil {
		user.AvatarURL = sql.NullString{String: *input.AvatarURL, Valid: *input.AvatarURL != ""}
	}
	if input.Bio != nil {
		user.Bio = sql.NullString{String: *input.Bio, Valid: *input.Bio != ""}
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return user, nil
}

// Search searches users by query
func (s *UserService) Search(ctx context.Context, query string, limit, offset int) ([]*model.UserProfile, error) {
	users, err := s.userRepo.Search(ctx, query, limit, offset)
	if err != nil {
		s.logger.Error("Failed to search users", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	profiles := make([]*model.UserProfile, len(users))
	for i, user := range users {
		profiles[i] = user.ToProfile()
	}

	return profiles, nil
}

// UpdateStatus updates user online status
func (s *UserService) UpdateStatus(ctx context.Context, userID string, status model.UserStatus) error {
	if err := s.userRepo.UpdateStatus(ctx, userID, status); err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.ErrUserNotFound
		}
		s.logger.Error("Failed to update status", zap.Error(err))
		return apperrors.ErrInternal
	}
	return nil
}

// BlockUser blocks a user
func (s *UserService) BlockUser(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return apperrors.ErrCannotBlockSelf
	}

	// Check if blocked user exists
	if _, err := s.userRepo.GetByID(ctx, blockedID); err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.ErrUserNotFound
		}
		return apperrors.ErrInternal
	}

	if err := s.blockedRepo.Block(ctx, blockerID, blockedID); err != nil {
		if err == repository.ErrAlreadyBlocked {
			return apperrors.ErrAlreadyBlocked
		}
		s.logger.Error("Failed to block user", zap.Error(err))
		return apperrors.ErrInternal
	}

	// Remove friendship if exists
	_ = s.friendshipRepo.Remove(ctx, blockerID, blockedID)

	s.logger.Info("User blocked",
		zap.String("blocker_id", blockerID),
		zap.String("blocked_id", blockedID),
	)

	return nil
}

// UnblockUser unblocks a user
func (s *UserService) UnblockUser(ctx context.Context, blockerID, blockedID string) error {
	if err := s.blockedRepo.Unblock(ctx, blockerID, blockedID); err != nil {
		if err == repository.ErrBlockNotFound {
			return apperrors.ErrNotFound
		}
		s.logger.Error("Failed to unblock user", zap.Error(err))
		return apperrors.ErrInternal
	}

	s.logger.Info("User unblocked",
		zap.String("blocker_id", blockerID),
		zap.String("blocked_id", blockedID),
	)

	return nil
}

// IsBlocked checks if a user is blocked by another
func (s *UserService) IsBlocked(ctx context.Context, blockerID, blockedID string) (bool, error) {
	return s.blockedRepo.IsBlocked(ctx, blockerID, blockedID)
}

// IsBlockedEither checks if either user has blocked the other
func (s *UserService) IsBlockedEither(ctx context.Context, userID1, userID2 string) (bool, error) {
	return s.blockedRepo.IsBlockedEither(ctx, userID1, userID2)
}

// ListBlockedUsers lists blocked users
func (s *UserService) ListBlockedUsers(ctx context.Context, blockerID string, limit, offset int) ([]*model.UserProfile, error) {
	users, err := s.blockedRepo.ListBlocked(ctx, blockerID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list blocked users", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	profiles := make([]*model.UserProfile, len(users))
	for i, user := range users {
		profiles[i] = user.ToProfile()
	}

	return profiles, nil
}

// SendFriendRequest sends a friend request
func (s *UserService) SendFriendRequest(ctx context.Context, userID, friendID string) error {
	if userID == friendID {
		return apperrors.New(400, "無法加自己為好友")
	}

	// Check if user is blocked
	blocked, err := s.blockedRepo.IsBlockedEither(ctx, userID, friendID)
	if err != nil {
		return apperrors.ErrInternal
	}
	if blocked {
		return apperrors.ErrUserBlocked
	}

	// Check if friend exists
	if _, err := s.userRepo.GetByID(ctx, friendID); err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.ErrUserNotFound
		}
		return apperrors.ErrInternal
	}

	// Check if already friends
	areFriends, err := s.friendshipRepo.AreFriends(ctx, userID, friendID)
	if err != nil {
		return apperrors.ErrInternal
	}
	if areFriends {
		return apperrors.ErrAlreadyFriend
	}

	if err := s.friendshipRepo.Create(ctx, userID, friendID); err != nil {
		s.logger.Error("Failed to create friend request", zap.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}

// AcceptFriendRequest accepts a friend request
func (s *UserService) AcceptFriendRequest(ctx context.Context, userID, friendID string) error {
	if err := s.friendshipRepo.Accept(ctx, userID, friendID); err != nil {
		if err == repository.ErrFriendshipNotFound {
			return apperrors.ErrNotFound
		}
		s.logger.Error("Failed to accept friend request", zap.Error(err))
		return apperrors.ErrInternal
	}
	return nil
}

// RejectFriendRequest rejects a friend request
func (s *UserService) RejectFriendRequest(ctx context.Context, userID, friendID string) error {
	if err := s.friendshipRepo.Reject(ctx, userID, friendID); err != nil {
		if err == repository.ErrFriendshipNotFound {
			return apperrors.ErrNotFound
		}
		s.logger.Error("Failed to reject friend request", zap.Error(err))
		return apperrors.ErrInternal
	}
	return nil
}

// RemoveFriend removes a friend
func (s *UserService) RemoveFriend(ctx context.Context, userID, friendID string) error {
	if err := s.friendshipRepo.Remove(ctx, userID, friendID); err != nil {
		if err == repository.ErrFriendshipNotFound {
			return apperrors.ErrNotFound
		}
		s.logger.Error("Failed to remove friend", zap.Error(err))
		return apperrors.ErrInternal
	}
	return nil
}

// ListFriends lists user's friends
func (s *UserService) ListFriends(ctx context.Context, userID string, limit, offset int) ([]*model.FriendshipWithUser, error) {
	friends, err := s.friendshipRepo.ListFriends(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list friends", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return friends, nil
}

// ListPendingRequests lists pending friend requests
func (s *UserService) ListPendingRequests(ctx context.Context, userID string, limit, offset int) ([]*model.FriendshipWithUser, error) {
	requests, err := s.friendshipRepo.ListPendingRequests(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list pending requests", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return requests, nil
}

// ListSentRequests lists sent friend requests
func (s *UserService) ListSentRequests(ctx context.Context, userID string, limit, offset int) ([]*model.FriendshipWithUser, error) {
	requests, err := s.friendshipRepo.ListSentRequests(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list sent requests", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return requests, nil
}

// AreFriends checks if two users are friends
func (s *UserService) AreFriends(ctx context.Context, userID, friendID string) (bool, error) {
	return s.friendshipRepo.AreFriends(ctx, userID, friendID)
}

// GetOnlineUsers gets online users
func (s *UserService) GetOnlineUsers(ctx context.Context, limit, offset int) ([]*model.UserProfile, error) {
	users, err := s.userRepo.GetOnlineUsers(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get online users", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	profiles := make([]*model.UserProfile, len(users))
	for i, user := range users {
		profiles[i] = user.ToProfile()
	}

	return profiles, nil
}
