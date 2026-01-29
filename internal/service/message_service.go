package service

import (
	"context"
	"database/sql"

	"github.com/go-demo/chat/internal/model"
	apperrors "github.com/go-demo/chat/internal/pkg/errors"
	"github.com/go-demo/chat/internal/repository"
	"go.uber.org/zap"
)

type MessageService struct {
	messageRepo *repository.MessageRepository
	roomRepo    *repository.RoomRepository
	logger      *zap.Logger
}

func NewMessageService(
	messageRepo *repository.MessageRepository,
	roomRepo *repository.RoomRepository,
	logger *zap.Logger,
) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
		roomRepo:    roomRepo,
		logger:      logger,
	}
}

// SendMessageInput represents message sending input
type SendMessageInput struct {
	RoomID    string
	UserID    string
	Content   string
	Type      model.MessageType
	ReplyToID string
}

// SendMessage sends a message to a room
func (s *MessageService) SendMessage(ctx context.Context, input *SendMessageInput) (*model.MessageWithUser, error) {
	// Check if user is a member of the room
	isMember, err := s.roomRepo.IsMember(ctx, input.RoomID, input.UserID)
	if err != nil {
		s.logger.Error("Failed to check membership", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	if !isMember {
		return nil, apperrors.ErrPermissionDenied
	}

	// Set default type
	if input.Type == "" {
		input.Type = model.MessageTypeText
	}

	msg := &model.Message{
		RoomID:  input.RoomID,
		UserID:  input.UserID,
		Content: input.Content,
		Type:    input.Type,
	}

	if input.ReplyToID != "" {
		msg.ReplyToID = sql.NullString{String: input.ReplyToID, Valid: true}
	}

	if err := s.messageRepo.Create(ctx, msg); err != nil {
		s.logger.Error("Failed to create message", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	// Get message with user info
	msgWithUser, err := s.messageRepo.GetByIDWithUser(ctx, msg.ID)
	if err != nil {
		s.logger.Error("Failed to get message with user", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return msgWithUser, nil
}

// GetByID retrieves a message by ID
func (s *MessageService) GetByID(ctx context.Context, id string) (*model.MessageWithUser, error) {
	msg, err := s.messageRepo.GetByIDWithUser(ctx, id)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			return nil, apperrors.ErrNotFound
		}
		s.logger.Error("Failed to get message", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	return msg, nil
}

// UpdateMessage updates a message content
func (s *MessageService) UpdateMessage(ctx context.Context, messageID, userID, content string) (*model.MessageWithUser, error) {
	// Get the original message
	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.ErrInternal
	}

	// Check ownership
	if msg.UserID != userID {
		return nil, apperrors.ErrPermissionDenied
	}

	// Check if deleted
	if msg.IsDeleted {
		return nil, apperrors.New(400, "無法編輯已刪除的訊息")
	}

	if err := s.messageRepo.Update(ctx, messageID, content); err != nil {
		s.logger.Error("Failed to update message", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return s.messageRepo.GetByIDWithUser(ctx, messageID)
}

// DeleteMessage soft deletes a message
func (s *MessageService) DeleteMessage(ctx context.Context, messageID, userID string) error {
	// Get the message
	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		if err == repository.ErrMessageNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.ErrInternal
	}

	// Check ownership or moderation permission
	if msg.UserID != userID {
		// Check if user can moderate
		member, err := s.roomRepo.GetMember(ctx, msg.RoomID, userID)
		if err != nil || !member.CanModerate() {
			return apperrors.ErrPermissionDenied
		}
	}

	if err := s.messageRepo.SoftDelete(ctx, messageID); err != nil {
		s.logger.Error("Failed to delete message", zap.Error(err))
		return apperrors.ErrInternal
	}

	s.logger.Info("Message deleted",
		zap.String("message_id", messageID),
		zap.String("deleted_by", userID),
	)

	return nil
}

// ListByRoomID retrieves messages for a room
func (s *MessageService) ListByRoomID(ctx context.Context, roomID, userID string, limit, offset int) ([]*model.MessageWithUser, error) {
	// Check if user is a member
	isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	// For public rooms, allow non-members to view
	if !isMember {
		room, err := s.roomRepo.GetByID(ctx, roomID)
		if err != nil {
			if err == repository.ErrRoomNotFound {
				return nil, apperrors.ErrRoomNotFound
			}
			return nil, apperrors.ErrInternal
		}
		if !room.IsPublic() {
			return nil, apperrors.ErrPermissionDenied
		}
	}

	messages, err := s.messageRepo.ListByRoomID(ctx, roomID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list messages", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return messages, nil
}

// ListSince retrieves messages since a specific message ID
func (s *MessageService) ListSince(ctx context.Context, roomID, userID, sinceID string, limit int) ([]*model.MessageWithUser, error) {
	// Check if user is a member
	isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	if !isMember {
		return nil, apperrors.ErrPermissionDenied
	}

	messages, err := s.messageRepo.ListByRoomIDSince(ctx, roomID, sinceID, limit)
	if err != nil {
		s.logger.Error("Failed to list messages since", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return messages, nil
}

// Search searches messages in a room
func (s *MessageService) Search(ctx context.Context, roomID, userID, query string, limit, offset int) ([]*model.MessageWithUser, error) {
	// Check if user is a member
	isMember, err := s.roomRepo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	if !isMember {
		return nil, apperrors.ErrPermissionDenied
	}

	messages, err := s.messageRepo.Search(ctx, roomID, query, limit, offset)
	if err != nil {
		s.logger.Error("Failed to search messages", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return messages, nil
}

// CountUnread counts unread messages for a user in a room
func (s *MessageService) CountUnread(ctx context.Context, roomID, userID string) (int, error) {
	count, err := s.messageRepo.CountUnreadByRoomID(ctx, roomID, userID)
	if err != nil {
		return 0, apperrors.ErrInternal
	}
	return count, nil
}

// CreateAttachment creates a message attachment
func (s *MessageService) CreateAttachment(ctx context.Context, att *model.MessageAttachment) error {
	return s.messageRepo.CreateAttachment(ctx, att)
}

// GetAttachments retrieves attachments for a message
func (s *MessageService) GetAttachments(ctx context.Context, messageID string) ([]*model.MessageAttachment, error) {
	return s.messageRepo.GetAttachmentsByMessageID(ctx, messageID)
}
