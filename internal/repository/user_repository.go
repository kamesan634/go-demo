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
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, display_name, avatar_url, status, bio)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowxContext(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.AvatarURL,
		user.Status,
		user.Bio,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE id = $1`

	if err := r.db.GetContext(ctx, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE username = $1`

	if err := r.db.GetContext(ctx, &user, query, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE email = $1`

	if err := r.db.GetContext(ctx, &user, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET display_name = $2, avatar_url = $3, bio = $4, status = $5
		WHERE id = $1
		RETURNING updated_at`

	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.DisplayName,
		user.AvatarURL,
		user.Bio,
		user.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdatePassword updates user password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateStatus updates user online status
func (r *UserRepository) UpdateStatus(ctx context.Context, userID string, status model.UserStatus) error {
	query := `UPDATE users SET status = $2, last_seen_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, userID, status)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Search searches users by username or display_name
func (r *UserRepository) Search(ctx context.Context, query string, limit, offset int) ([]*model.User, error) {
	searchQuery := `
		SELECT * FROM users
		WHERE username ILIKE $1 OR display_name ILIKE $1
		ORDER BY username
		LIMIT $2 OFFSET $3`

	var users []*model.User
	pattern := "%" + query + "%"

	if err := r.db.SelectContext(ctx, &users, searchQuery, pattern, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	return users, nil
}

// ExistsByUsername checks if username exists
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`

	if err := r.db.GetContext(ctx, &exists, query, username); err != nil {
		return false, fmt.Errorf("failed to check username exists: %w", err)
	}

	return exists, nil
}

// ExistsByEmail checks if email exists
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	if err := r.db.GetContext(ctx, &exists, query, email); err != nil {
		return false, fmt.Errorf("failed to check email exists: %w", err)
	}

	return exists, nil
}

// GetOnlineUsers gets online users (for admin dashboard)
func (r *UserRepository) GetOnlineUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	query := `
		SELECT * FROM users
		WHERE status = 'online'
		ORDER BY last_seen_at DESC
		LIMIT $1 OFFSET $2`

	var users []*model.User
	if err := r.db.SelectContext(ctx, &users, query, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to get online users: %w", err)
	}

	return users, nil
}

// GetByIDs retrieves multiple users by IDs
func (r *UserRepository) GetByIDs(ctx context.Context, ids []string) ([]*model.User, error) {
	if len(ids) == 0 {
		return []*model.User{}, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM users WHERE id IN (?)`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	query = r.db.Rebind(query)
	var users []*model.User

	if err := r.db.SelectContext(ctx, &users, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get users by ids: %w", err)
	}

	return users, nil
}
