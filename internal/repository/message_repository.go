package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-demo/chat/internal/model"
	"github.com/jmoiron/sqlx"
)

var (
	ErrMessageNotFound = errors.New("message not found")
)

type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create creates a new message
func (r *MessageRepository) Create(ctx context.Context, msg *model.Message) error {
	query := `
		INSERT INTO messages (room_id, user_id, content, type, reply_to_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		msg.RoomID,
		msg.UserID,
		msg.Content,
		msg.Type,
		msg.ReplyToID,
	).Scan(&msg.ID, &msg.CreatedAt, &msg.UpdatedAt)
}

// GetByID retrieves a message by ID
func (r *MessageRepository) GetByID(ctx context.Context, id string) (*model.Message, error) {
	var msg model.Message
	query := `SELECT * FROM messages WHERE id = $1`

	if err := r.db.GetContext(ctx, &msg, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to get message by id: %w", err)
	}

	return &msg, nil
}

// GetByIDWithUser retrieves a message by ID with user info
func (r *MessageRepository) GetByIDWithUser(ctx context.Context, id string) (*model.MessageWithUser, error) {
	var msg model.MessageWithUser
	query := `
		SELECT m.*, u.username, u.display_name, u.avatar_url
		FROM messages m
		INNER JOIN users u ON m.user_id = u.id
		WHERE m.id = $1`

	if err := r.db.GetContext(ctx, &msg, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to get message with user: %w", err)
	}

	return &msg, nil
}

// Update updates a message content
func (r *MessageRepository) Update(ctx context.Context, id, content string) error {
	query := `UPDATE messages SET content = $2, is_edited = true WHERE id = $1 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, id, content)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrMessageNotFound
	}

	return nil
}

// SoftDelete marks a message as deleted
func (r *MessageRepository) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE messages SET is_deleted = true, content = '[訊息已刪除]' WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrMessageNotFound
	}

	return nil
}

// ListByRoomID retrieves messages for a room (paginated)
func (r *MessageRepository) ListByRoomID(ctx context.Context, roomID string, limit, offset int) ([]*model.MessageWithUser, error) {
	query := `
		SELECT m.*, u.username, u.display_name, u.avatar_url
		FROM messages m
		INNER JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3`

	var messages []*model.MessageWithUser
	if err := r.db.SelectContext(ctx, &messages, query, roomID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// ListByRoomIDSince retrieves messages after a specific time (for real-time sync)
func (r *MessageRepository) ListByRoomIDSince(ctx context.Context, roomID string, sinceID string, limit int) ([]*model.MessageWithUser, error) {
	query := `
		SELECT m.*, u.username, u.display_name, u.avatar_url
		FROM messages m
		INNER JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1 AND m.created_at > (
			SELECT created_at FROM messages WHERE id = $2
		)
		ORDER BY m.created_at ASC
		LIMIT $3`

	var messages []*model.MessageWithUser
	if err := r.db.SelectContext(ctx, &messages, query, roomID, sinceID, limit); err != nil {
		return nil, fmt.Errorf("failed to list messages since: %w", err)
	}

	return messages, nil
}

// CountByRoomID counts messages in a room
func (r *MessageRepository) CountByRoomID(ctx context.Context, roomID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM messages WHERE room_id = $1`

	if err := r.db.GetContext(ctx, &count, query, roomID); err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}

// CountUnreadByRoomID counts unread messages for a user in a room
func (r *MessageRepository) CountUnreadByRoomID(ctx context.Context, roomID, userID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM messages m
		INNER JOIN room_members rm ON m.room_id = rm.room_id AND rm.user_id = $2
		WHERE m.room_id = $1 AND m.created_at > rm.last_read_at AND m.user_id != $2`

	if err := r.db.GetContext(ctx, &count, query, roomID, userID); err != nil {
		return 0, fmt.Errorf("failed to count unread messages: %w", err)
	}

	return count, nil
}

// Search searches messages in a room
func (r *MessageRepository) Search(ctx context.Context, roomID, query string, limit, offset int) ([]*model.MessageWithUser, error) {
	searchQuery := `
		SELECT m.*, u.username, u.display_name, u.avatar_url
		FROM messages m
		INNER JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1 AND m.content ILIKE $2 AND m.is_deleted = false
		ORDER BY m.created_at DESC
		LIMIT $3 OFFSET $4`

	var messages []*model.MessageWithUser
	pattern := "%" + query + "%"

	if err := r.db.SelectContext(ctx, &messages, searchQuery, roomID, pattern, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	return messages, nil
}

// CreateAttachment creates a message attachment
func (r *MessageRepository) CreateAttachment(ctx context.Context, att *model.MessageAttachment) error {
	query := `
		INSERT INTO message_attachments (message_id, file_name, file_url, file_type, file_size)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	return r.db.QueryRowxContext(ctx, query,
		att.MessageID,
		att.FileName,
		att.FileURL,
		att.FileType,
		att.FileSize,
	).Scan(&att.ID, &att.CreatedAt)
}

// GetAttachmentsByMessageID retrieves attachments for a message
func (r *MessageRepository) GetAttachmentsByMessageID(ctx context.Context, messageID string) ([]*model.MessageAttachment, error) {
	query := `SELECT * FROM message_attachments WHERE message_id = $1 ORDER BY created_at`

	var attachments []*model.MessageAttachment
	if err := r.db.SelectContext(ctx, &attachments, query, messageID); err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}

	return attachments, nil
}

// GetLatestByRoomID retrieves the latest message in a room
func (r *MessageRepository) GetLatestByRoomID(ctx context.Context, roomID string) (*model.MessageWithUser, error) {
	var msg model.MessageWithUser
	query := `
		SELECT m.*, u.username, u.display_name, u.avatar_url
		FROM messages m
		INNER JOIN users u ON m.user_id = u.id
		WHERE m.room_id = $1
		ORDER BY m.created_at DESC
		LIMIT 1`

	if err := r.db.GetContext(ctx, &msg, query, roomID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No messages yet
		}
		return nil, fmt.Errorf("failed to get latest message: %w", err)
	}

	return &msg, nil
}
