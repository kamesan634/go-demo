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
	ErrDirectMessageNotFound = errors.New("direct message not found")
)

type DirectMessageRepository struct {
	db *sqlx.DB
}

func NewDirectMessageRepository(db *sqlx.DB) *DirectMessageRepository {
	return &DirectMessageRepository{db: db}
}

// Create creates a new direct message
func (r *DirectMessageRepository) Create(ctx context.Context, msg *model.DirectMessage) error {
	query := `
		INSERT INTO direct_messages (sender_id, receiver_id, content, type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		msg.SenderID,
		msg.ReceiverID,
		msg.Content,
		msg.Type,
	).Scan(&msg.ID, &msg.CreatedAt, &msg.UpdatedAt)
}

// GetByID retrieves a direct message by ID
func (r *DirectMessageRepository) GetByID(ctx context.Context, id string) (*model.DirectMessage, error) {
	var msg model.DirectMessage
	query := `SELECT * FROM direct_messages WHERE id = $1`

	if err := r.db.GetContext(ctx, &msg, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDirectMessageNotFound
		}
		return nil, fmt.Errorf("failed to get direct message: %w", err)
	}

	return &msg, nil
}

// GetByIDWithUser retrieves a direct message with sender info
func (r *DirectMessageRepository) GetByIDWithUser(ctx context.Context, id string) (*model.DirectMessageWithUser, error) {
	var msg model.DirectMessageWithUser
	query := `
		SELECT dm.*, u.username as sender_username, u.display_name as sender_display_name, u.avatar_url as sender_avatar_url
		FROM direct_messages dm
		INNER JOIN users u ON dm.sender_id = u.id
		WHERE dm.id = $1`

	if err := r.db.GetContext(ctx, &msg, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDirectMessageNotFound
		}
		return nil, fmt.Errorf("failed to get direct message with user: %w", err)
	}

	return &msg, nil
}

// ListConversation retrieves messages between two users
func (r *DirectMessageRepository) ListConversation(ctx context.Context, userID1, userID2 string, limit, offset int) ([]*model.DirectMessageWithUser, error) {
	query := `
		SELECT dm.*, u.username as sender_username, u.display_name as sender_display_name, u.avatar_url as sender_avatar_url
		FROM direct_messages dm
		INNER JOIN users u ON dm.sender_id = u.id
		WHERE (
			(dm.sender_id = $1 AND dm.receiver_id = $2 AND dm.is_deleted_by_sender = false)
			OR
			(dm.sender_id = $2 AND dm.receiver_id = $1 AND dm.is_deleted_by_receiver = false)
		)
		ORDER BY dm.created_at DESC
		LIMIT $3 OFFSET $4`

	var messages []*model.DirectMessageWithUser
	if err := r.db.SelectContext(ctx, &messages, query, userID1, userID2, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list conversation: %w", err)
	}

	// Reverse for chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// ListConversations lists all conversations for a user
func (r *DirectMessageRepository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*model.Conversation, error) {
	query := `
		WITH latest_messages AS (
			SELECT DISTINCT ON (
				LEAST(sender_id, receiver_id),
				GREATEST(sender_id, receiver_id)
			)
				CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END as other_user_id,
				content as last_message,
				created_at as last_message_at
			FROM direct_messages
			WHERE (sender_id = $1 AND is_deleted_by_sender = false)
				OR (receiver_id = $1 AND is_deleted_by_receiver = false)
			ORDER BY
				LEAST(sender_id, receiver_id),
				GREATEST(sender_id, receiver_id),
				created_at DESC
		),
		unread_counts AS (
			SELECT sender_id, COUNT(*) as unread_count
			FROM direct_messages
			WHERE receiver_id = $1 AND is_read = false AND is_deleted_by_receiver = false
			GROUP BY sender_id
		)
		SELECT
			u.id as user_id,
			u.username,
			COALESCE(u.display_name, u.username) as display_name,
			COALESCE(u.avatar_url, '') as avatar_url,
			u.status,
			lm.last_message,
			lm.last_message_at,
			COALESCE(uc.unread_count, 0) as unread_count
		FROM latest_messages lm
		INNER JOIN users u ON lm.other_user_id = u.id
		LEFT JOIN unread_counts uc ON u.id = uc.sender_id
		ORDER BY lm.last_message_at DESC
		LIMIT $2 OFFSET $3`

	var conversations []*model.Conversation
	if err := r.db.SelectContext(ctx, &conversations, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	return conversations, nil
}

// MarkAsRead marks messages as read
func (r *DirectMessageRepository) MarkAsRead(ctx context.Context, senderID, receiverID string) error {
	query := `
		UPDATE direct_messages
		SET is_read = true
		WHERE sender_id = $1 AND receiver_id = $2 AND is_read = false`

	_, err := r.db.ExecContext(ctx, query, senderID, receiverID)
	if err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	return nil
}

// DeleteForUser soft deletes messages for a specific user
func (r *DirectMessageRepository) DeleteForUser(ctx context.Context, messageID, userID string) error {
	// First get the message to check sender/receiver
	msg, err := r.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	var query string
	if msg.SenderID == userID {
		query = `UPDATE direct_messages SET is_deleted_by_sender = true WHERE id = $1`
	} else if msg.ReceiverID == userID {
		query = `UPDATE direct_messages SET is_deleted_by_receiver = true WHERE id = $1`
	} else {
		return fmt.Errorf("user is not part of this conversation")
	}

	_, err = r.db.ExecContext(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message for user: %w", err)
	}

	return nil
}

// CountUnread counts unread messages for a user
func (r *DirectMessageRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM direct_messages
		WHERE receiver_id = $1 AND is_read = false AND is_deleted_by_receiver = false`

	if err := r.db.GetContext(ctx, &count, query, userID); err != nil {
		return 0, fmt.Errorf("failed to count unread: %w", err)
	}

	return count, nil
}

// CountUnreadFromUser counts unread messages from a specific user
func (r *DirectMessageRepository) CountUnreadFromUser(ctx context.Context, receiverID, senderID string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM direct_messages
		WHERE receiver_id = $1 AND sender_id = $2 AND is_read = false AND is_deleted_by_receiver = false`

	if err := r.db.GetContext(ctx, &count, query, receiverID, senderID); err != nil {
		return 0, fmt.Errorf("failed to count unread from user: %w", err)
	}

	return count, nil
}
