package service

import (
	"context"

	"github.com/go-demo/chat/internal/model"
	apperrors "github.com/go-demo/chat/internal/pkg/errors"
	"github.com/go-demo/chat/internal/repository"
	"go.uber.org/zap"
)

type DirectMessageService struct {
	dmRepo      *repository.DirectMessageRepository
	userRepo    *repository.UserRepository
	blockedRepo *repository.BlockedUserRepository
	logger      *zap.Logger
}

func NewDirectMessageService(
	dmRepo *repository.DirectMessageRepository,
	userRepo *repository.UserRepository,
	blockedRepo *repository.BlockedUserRepository,
	logger *zap.Logger,
) *DirectMessageService {
	return &DirectMessageService{
		dmRepo:      dmRepo,
		userRepo:    userRepo,
		blockedRepo: blockedRepo,
		logger:      logger,
	}
}

// SendMessageInput represents DM sending input
type SendDMInput struct {
	SenderID   string
	ReceiverID string
	Content    string
	Type       model.MessageType
}

// SendMessage sends a direct message
func (s *DirectMessageService) SendMessage(ctx context.Context, input *SendDMInput) (*model.DirectMessageWithUser, error) {
	// Cannot message self
	if input.SenderID == input.ReceiverID {
		return nil, apperrors.ErrCannotMessageSelf
	}

	// Check if receiver exists
	if _, err := s.userRepo.GetByID(ctx, input.ReceiverID); err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.ErrInternal
	}

	// Check if blocked
	blocked, err := s.blockedRepo.IsBlockedEither(ctx, input.SenderID, input.ReceiverID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	if blocked {
		return nil, apperrors.ErrUserBlocked
	}

	// Set default type
	if input.Type == "" {
		input.Type = model.MessageTypeText
	}

	msg := &model.DirectMessage{
		SenderID:   input.SenderID,
		ReceiverID: input.ReceiverID,
		Content:    input.Content,
		Type:       input.Type,
	}

	if err := s.dmRepo.Create(ctx, msg); err != nil {
		s.logger.Error("Failed to create direct message", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	// Get message with user info
	msgWithUser, err := s.dmRepo.GetByIDWithUser(ctx, msg.ID)
	if err != nil {
		s.logger.Error("Failed to get direct message with user", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return msgWithUser, nil
}

// GetConversation retrieves messages between two users
func (s *DirectMessageService) GetConversation(ctx context.Context, userID, otherUserID string, limit, offset int) ([]*model.DirectMessageWithUser, error) {
	// Check if other user exists
	if _, err := s.userRepo.GetByID(ctx, otherUserID); err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.ErrInternal
	}

	messages, err := s.dmRepo.ListConversation(ctx, userID, otherUserID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list conversation", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return messages, nil
}

// ListConversations lists all conversations for a user
func (s *DirectMessageService) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*model.Conversation, error) {
	conversations, err := s.dmRepo.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list conversations", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return conversations, nil
}

// MarkAsRead marks messages as read
func (s *DirectMessageService) MarkAsRead(ctx context.Context, userID, senderID string) error {
	if err := s.dmRepo.MarkAsRead(ctx, senderID, userID); err != nil {
		s.logger.Error("Failed to mark as read", zap.Error(err))
		return apperrors.ErrInternal
	}
	return nil
}

// DeleteMessage deletes a message for a user
func (s *DirectMessageService) DeleteMessage(ctx context.Context, messageID, userID string) error {
	if err := s.dmRepo.DeleteForUser(ctx, messageID, userID); err != nil {
		if err == repository.ErrDirectMessageNotFound {
			return apperrors.ErrNotFound
		}
		s.logger.Error("Failed to delete direct message", zap.Error(err))
		return apperrors.ErrInternal
	}
	return nil
}

// CountUnread counts unread messages for a user
func (s *DirectMessageService) CountUnread(ctx context.Context, userID string) (int, error) {
	count, err := s.dmRepo.CountUnread(ctx, userID)
	if err != nil {
		return 0, apperrors.ErrInternal
	}
	return count, nil
}

// CountUnreadFromUser counts unread messages from a specific user
func (s *DirectMessageService) CountUnreadFromUser(ctx context.Context, receiverID, senderID string) (int, error) {
	count, err := s.dmRepo.CountUnreadFromUser(ctx, receiverID, senderID)
	if err != nil {
		return 0, apperrors.ErrInternal
	}
	return count, nil
}

// GetByID retrieves a direct message by ID
func (s *DirectMessageService) GetByID(ctx context.Context, id, userID string) (*model.DirectMessageWithUser, error) {
	msg, err := s.dmRepo.GetByIDWithUser(ctx, id)
	if err != nil {
		if err == repository.ErrDirectMessageNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.ErrInternal
	}

	// Check if user is part of the conversation
	if msg.SenderID != userID && msg.ReceiverID != userID {
		return nil, apperrors.ErrPermissionDenied
	}

	return msg, nil
}
