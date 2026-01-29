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
	ErrBlockNotFound      = errors.New("block not found")
	ErrAlreadyBlocked     = errors.New("user already blocked")
	ErrCannotBlockSelf    = errors.New("cannot block yourself")
	ErrFriendshipNotFound = errors.New("friendship not found")
)

type BlockedUserRepository struct {
	db *sqlx.DB
}

func NewBlockedUserRepository(db *sqlx.DB) *BlockedUserRepository {
	return &BlockedUserRepository{db: db}
}

// Block blocks a user
func (r *BlockedUserRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrCannotBlockSelf
	}

	query := `
		INSERT INTO blocked_users (blocker_id, blocked_id)
		VALUES ($1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING
		RETURNING id`

	var id string
	err := r.db.QueryRowxContext(ctx, query, blockerID, blockedID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAlreadyBlocked
		}
		return fmt.Errorf("failed to block user: %w", err)
	}

	return nil
}

// Unblock unblocks a user
func (r *BlockedUserRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	query := `DELETE FROM blocked_users WHERE blocker_id = $1 AND blocked_id = $2`

	result, err := r.db.ExecContext(ctx, query, blockerID, blockedID)
	if err != nil {
		return fmt.Errorf("failed to unblock user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrBlockNotFound
	}

	return nil
}

// IsBlocked checks if a user is blocked by another
func (r *BlockedUserRepository) IsBlocked(ctx context.Context, blockerID, blockedID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM blocked_users WHERE blocker_id = $1 AND blocked_id = $2)`

	if err := r.db.GetContext(ctx, &exists, query, blockerID, blockedID); err != nil {
		return false, fmt.Errorf("failed to check if blocked: %w", err)
	}

	return exists, nil
}

// IsBlockedEither checks if either user has blocked the other
func (r *BlockedUserRepository) IsBlockedEither(ctx context.Context, userID1, userID2 string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM blocked_users
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)`

	if err := r.db.GetContext(ctx, &exists, query, userID1, userID2); err != nil {
		return false, fmt.Errorf("failed to check if blocked either: %w", err)
	}

	return exists, nil
}

// ListBlocked lists users blocked by a user
func (r *BlockedUserRepository) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]*model.User, error) {
	query := `
		SELECT u.*
		FROM users u
		INNER JOIN blocked_users b ON u.id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3`

	var users []*model.User
	if err := r.db.SelectContext(ctx, &users, query, blockerID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list blocked users: %w", err)
	}

	return users, nil
}

// FriendshipRepository handles friendship operations
type FriendshipRepository struct {
	db *sqlx.DB
}

func NewFriendshipRepository(db *sqlx.DB) *FriendshipRepository {
	return &FriendshipRepository{db: db}
}

// Create creates a friend request
func (r *FriendshipRepository) Create(ctx context.Context, userID, friendID string) error {
	query := `
		INSERT INTO friendships (user_id, friend_id, status)
		VALUES ($1, $2, 'pending')
		ON CONFLICT (user_id, friend_id) DO NOTHING
		RETURNING id`

	var id string
	err := r.db.QueryRowxContext(ctx, query, userID, friendID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("friend request already exists")
		}
		return fmt.Errorf("failed to create friend request: %w", err)
	}

	return nil
}

// Accept accepts a friend request
func (r *FriendshipRepository) Accept(ctx context.Context, userID, friendID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update the existing request to accepted
	query := `
		UPDATE friendships
		SET status = 'accepted'
		WHERE user_id = $1 AND friend_id = $2 AND status = 'pending'`

	result, err := tx.ExecContext(ctx, query, friendID, userID)
	if err != nil {
		return fmt.Errorf("failed to accept friend request: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrFriendshipNotFound
	}

	// Create reverse friendship
	reverseQuery := `
		INSERT INTO friendships (user_id, friend_id, status)
		VALUES ($1, $2, 'accepted')
		ON CONFLICT (user_id, friend_id) DO UPDATE SET status = 'accepted'`

	_, err = tx.ExecContext(ctx, reverseQuery, userID, friendID)
	if err != nil {
		return fmt.Errorf("failed to create reverse friendship: %w", err)
	}

	return tx.Commit()
}

// Reject rejects a friend request
func (r *FriendshipRepository) Reject(ctx context.Context, userID, friendID string) error {
	query := `
		UPDATE friendships
		SET status = 'rejected'
		WHERE user_id = $1 AND friend_id = $2 AND status = 'pending'`

	result, err := r.db.ExecContext(ctx, query, friendID, userID)
	if err != nil {
		return fmt.Errorf("failed to reject friend request: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrFriendshipNotFound
	}

	return nil
}

// Remove removes a friendship
func (r *FriendshipRepository) Remove(ctx context.Context, userID, friendID string) error {
	query := `
		DELETE FROM friendships
		WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`

	result, err := r.db.ExecContext(ctx, query, userID, friendID)
	if err != nil {
		return fmt.Errorf("failed to remove friendship: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrFriendshipNotFound
	}

	return nil
}

// ListFriends lists accepted friends
func (r *FriendshipRepository) ListFriends(ctx context.Context, userID string, limit, offset int) ([]*model.FriendshipWithUser, error) {
	query := `
		SELECT f.*, u.username as friend_username, u.display_name as friend_display_name,
			   u.avatar_url as friend_avatar_url, u.status as friend_status
		FROM friendships f
		INNER JOIN users u ON f.friend_id = u.id
		WHERE f.user_id = $1 AND f.status = 'accepted'
		ORDER BY u.username
		LIMIT $2 OFFSET $3`

	var friendships []*model.FriendshipWithUser
	if err := r.db.SelectContext(ctx, &friendships, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list friends: %w", err)
	}

	return friendships, nil
}

// ListPendingRequests lists pending friend requests (received)
func (r *FriendshipRepository) ListPendingRequests(ctx context.Context, userID string, limit, offset int) ([]*model.FriendshipWithUser, error) {
	query := `
		SELECT f.*, u.username as friend_username, u.display_name as friend_display_name,
			   u.avatar_url as friend_avatar_url, u.status as friend_status
		FROM friendships f
		INNER JOIN users u ON f.user_id = u.id
		WHERE f.friend_id = $1 AND f.status = 'pending'
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`

	var friendships []*model.FriendshipWithUser
	if err := r.db.SelectContext(ctx, &friendships, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list pending requests: %w", err)
	}

	return friendships, nil
}

// ListSentRequests lists pending friend requests (sent)
func (r *FriendshipRepository) ListSentRequests(ctx context.Context, userID string, limit, offset int) ([]*model.FriendshipWithUser, error) {
	query := `
		SELECT f.*, u.username as friend_username, u.display_name as friend_display_name,
			   u.avatar_url as friend_avatar_url, u.status as friend_status
		FROM friendships f
		INNER JOIN users u ON f.friend_id = u.id
		WHERE f.user_id = $1 AND f.status = 'pending'
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`

	var friendships []*model.FriendshipWithUser
	if err := r.db.SelectContext(ctx, &friendships, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list sent requests: %w", err)
	}

	return friendships, nil
}

// AreFriends checks if two users are friends
func (r *FriendshipRepository) AreFriends(ctx context.Context, userID, friendID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM friendships
			WHERE user_id = $1 AND friend_id = $2 AND status = 'accepted'
		)`

	if err := r.db.GetContext(ctx, &exists, query, userID, friendID); err != nil {
		return false, fmt.Errorf("failed to check friendship: %w", err)
	}

	return exists, nil
}

// GetFriendship gets the friendship status between two users
func (r *FriendshipRepository) GetFriendship(ctx context.Context, userID, friendID string) (*model.Friendship, error) {
	var friendship model.Friendship
	query := `SELECT * FROM friendships WHERE user_id = $1 AND friend_id = $2`

	if err := r.db.GetContext(ctx, &friendship, query, userID, friendID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFriendshipNotFound
		}
		return nil, fmt.Errorf("failed to get friendship: %w", err)
	}

	return &friendship, nil
}
