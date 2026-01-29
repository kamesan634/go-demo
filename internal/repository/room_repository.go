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
	ErrRoomNotFound      = errors.New("room not found")
	ErrRoomAlreadyExists = errors.New("room already exists")
	ErrNotRoomMember     = errors.New("not a room member")
	ErrAlreadyRoomMember = errors.New("already a room member")
	ErrRoomFull          = errors.New("room is full")
)

type RoomRepository struct {
	db *sqlx.DB
}

func NewRoomRepository(db *sqlx.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

// Create creates a new room
func (r *RoomRepository) Create(ctx context.Context, room *model.Room) error {
	query := `
		INSERT INTO rooms (name, description, type, owner_id, max_members)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		room.Name,
		room.Description,
		room.Type,
		room.OwnerID,
		room.MaxMembers,
	).Scan(&room.ID, &room.CreatedAt, &room.UpdatedAt)
}

// GetByID retrieves a room by ID
func (r *RoomRepository) GetByID(ctx context.Context, id string) (*model.Room, error) {
	var room model.Room
	query := `SELECT * FROM rooms WHERE id = $1`

	if err := r.db.GetContext(ctx, &room, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("failed to get room by id: %w", err)
	}

	return &room, nil
}

// GetByIDWithMemberCount retrieves a room by ID with member count
func (r *RoomRepository) GetByIDWithMemberCount(ctx context.Context, id string) (*model.RoomWithMemberCount, error) {
	var room model.RoomWithMemberCount
	query := `
		SELECT r.*, COUNT(rm.id) as member_count
		FROM rooms r
		LEFT JOIN room_members rm ON r.id = rm.room_id
		WHERE r.id = $1
		GROUP BY r.id`

	if err := r.db.GetContext(ctx, &room, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, fmt.Errorf("failed to get room with member count: %w", err)
	}

	return &room, nil
}

// Update updates a room
func (r *RoomRepository) Update(ctx context.Context, room *model.Room) error {
	query := `
		UPDATE rooms
		SET name = $2, description = $3, max_members = $4
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		room.ID,
		room.Name,
		room.Description,
		room.MaxMembers,
	)
	if err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrRoomNotFound
	}

	return nil
}

// Delete deletes a room
func (r *RoomRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM rooms WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete room: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrRoomNotFound
	}

	return nil
}

// ListPublic lists public rooms
func (r *RoomRepository) ListPublic(ctx context.Context, limit, offset int) ([]*model.RoomWithMemberCount, error) {
	query := `
		SELECT r.*, COUNT(rm.id) as member_count
		FROM rooms r
		LEFT JOIN room_members rm ON r.id = rm.room_id
		WHERE r.type = 'public'
		GROUP BY r.id
		ORDER BY r.created_at DESC
		LIMIT $1 OFFSET $2`

	var rooms []*model.RoomWithMemberCount
	if err := r.db.SelectContext(ctx, &rooms, query, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list public rooms: %w", err)
	}

	return rooms, nil
}

// ListByUserID lists rooms that user is a member of
func (r *RoomRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.RoomWithMemberCount, error) {
	query := `
		SELECT r.*, COUNT(rm2.id) as member_count
		FROM rooms r
		INNER JOIN room_members rm ON r.id = rm.room_id AND rm.user_id = $1
		LEFT JOIN room_members rm2 ON r.id = rm2.room_id
		GROUP BY r.id, rm.joined_at
		ORDER BY rm.joined_at DESC
		LIMIT $2 OFFSET $3`

	var rooms []*model.RoomWithMemberCount
	if err := r.db.SelectContext(ctx, &rooms, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list user rooms: %w", err)
	}

	return rooms, nil
}

// Search searches rooms by name
func (r *RoomRepository) Search(ctx context.Context, query string, limit, offset int) ([]*model.RoomWithMemberCount, error) {
	searchQuery := `
		SELECT r.*, COUNT(rm.id) as member_count
		FROM rooms r
		LEFT JOIN room_members rm ON r.id = rm.room_id
		WHERE r.type = 'public' AND r.name ILIKE $1
		GROUP BY r.id
		ORDER BY r.name
		LIMIT $2 OFFSET $3`

	var rooms []*model.RoomWithMemberCount
	pattern := "%" + query + "%"

	if err := r.db.SelectContext(ctx, &rooms, searchQuery, pattern, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to search rooms: %w", err)
	}

	return rooms, nil
}

// AddMember adds a user to a room
func (r *RoomRepository) AddMember(ctx context.Context, member *model.RoomMember) error {
	// Check room exists and not full
	var room struct {
		MaxMembers  int `db:"max_members"`
		MemberCount int `db:"member_count"`
	}

	checkQuery := `
		SELECT r.max_members, COUNT(rm.id) as member_count
		FROM rooms r
		LEFT JOIN room_members rm ON r.id = rm.room_id
		WHERE r.id = $1
		GROUP BY r.id`

	if err := r.db.GetContext(ctx, &room, checkQuery, member.RoomID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoomNotFound
		}
		return fmt.Errorf("failed to check room: %w", err)
	}

	if room.MemberCount >= room.MaxMembers {
		return ErrRoomFull
	}

	query := `
		INSERT INTO room_members (room_id, user_id, role, nickname)
		VALUES ($1, $2, $3, $4)
		RETURNING id, joined_at, last_read_at`

	err := r.db.QueryRowxContext(ctx, query,
		member.RoomID,
		member.UserID,
		member.Role,
		member.Nickname,
	).Scan(&member.ID, &member.JoinedAt, &member.LastReadAt)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == `pq: duplicate key value violates unique constraint "room_members_room_id_user_id_key"` {
			return ErrAlreadyRoomMember
		}
		return fmt.Errorf("failed to add member: %w", err)
	}

	return nil
}

// RemoveMember removes a user from a room
func (r *RoomRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	query := `DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotRoomMember
	}

	return nil
}

// GetMember retrieves a room member
func (r *RoomRepository) GetMember(ctx context.Context, roomID, userID string) (*model.RoomMember, error) {
	var member model.RoomMember
	query := `SELECT * FROM room_members WHERE room_id = $1 AND user_id = $2`

	if err := r.db.GetContext(ctx, &member, query, roomID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotRoomMember
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	return &member, nil
}

// ListMembers lists all members of a room with user info
func (r *RoomRepository) ListMembers(ctx context.Context, roomID string) ([]*model.RoomMemberWithUser, error) {
	query := `
		SELECT rm.*, u.username, u.display_name, u.avatar_url, u.status
		FROM room_members rm
		INNER JOIN users u ON rm.user_id = u.id
		WHERE rm.room_id = $1
		ORDER BY rm.role, rm.joined_at`

	var members []*model.RoomMemberWithUser
	if err := r.db.SelectContext(ctx, &members, query, roomID); err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	return members, nil
}

// UpdateMemberRole updates a member's role
func (r *RoomRepository) UpdateMemberRole(ctx context.Context, roomID, userID string, role model.MemberRole) error {
	query := `UPDATE room_members SET role = $3 WHERE room_id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, roomID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotRoomMember
	}

	return nil
}

// UpdateLastReadAt updates member's last read timestamp
func (r *RoomRepository) UpdateLastReadAt(ctx context.Context, roomID, userID string) error {
	query := `UPDATE room_members SET last_read_at = NOW() WHERE room_id = $1 AND user_id = $2`

	_, err := r.db.ExecContext(ctx, query, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to update last read at: %w", err)
	}

	return nil
}

// IsMember checks if user is a member of the room
func (r *RoomRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM room_members WHERE room_id = $1 AND user_id = $2)`

	if err := r.db.GetContext(ctx, &exists, query, roomID, userID); err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return exists, nil
}

// CountMembers counts room members
func (r *RoomRepository) CountMembers(ctx context.Context, roomID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM room_members WHERE room_id = $1`

	if err := r.db.GetContext(ctx, &count, query, roomID); err != nil {
		return 0, fmt.Errorf("failed to count members: %w", err)
	}

	return count, nil
}
