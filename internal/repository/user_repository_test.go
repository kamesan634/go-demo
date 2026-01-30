package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-demo/chat/internal/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// 使用有效的 UUID 格式作為不存在的 ID
const nonExistentUUID = "00000000-0000-0000-0000-000000000000"

// setupUserTestDBIsolated creates an isolated test database connection
func setupUserTestDBIsolated(t *testing.T) (*sqlx.DB, string) {
	t.Helper()
	return SetupIsolatedTestDB(t)
}

func cleanupUserTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	CleanupTestDataByPrefix(t, db, prefix)
}

func TestUserRepository_Create(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     prefix + "_testuser_create",
		Email:        prefix + "_testcreate@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == "" {
		t.Error("Expected user ID to be set")
	}
	if user.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &model.User{
		Username:     prefix + "_testuser_getbyid",
		Email:        prefix + "_testgetbyid@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test GetByID
	found, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if found.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, found.Username)
	}

	// Test not found with valid UUID format
	_, err = repo.GetByID(ctx, nonExistentUUID)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	username := prefix + "_testuser_getbyusername"
	user := &model.User{
		Username:     username,
		Email:        prefix + "_testgetbyusername@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	found, err := repo.GetByUsername(ctx, username)
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, found.ID)
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	email := prefix + "_testgetbyemail@example.com"
	user := &model.User{
		Username:     prefix + "_testuser_getbyemail",
		Email:        email,
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	found, err := repo.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, found.ID)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     prefix + "_testuser_update",
		Email:        prefix + "_testupdate@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update user
	user.DisplayName = sql.NullString{String: "Test User", Valid: true}
	user.Bio = sql.NullString{String: "Hello world", Valid: true}

	err := repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify update
	found, _ := repo.GetByID(ctx, user.ID)
	if found.DisplayName.String != "Test User" {
		t.Errorf("Expected display name 'Test User', got '%s'", found.DisplayName.String)
	}
}

func TestUserRepository_UpdateStatus(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     prefix + "_testuser_status",
		Email:        prefix + "_teststatus@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	err := repo.UpdateStatus(ctx, user.ID, model.UserStatusOnline)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	found, _ := repo.GetByID(ctx, user.ID)
	if found.Status != model.UserStatusOnline {
		t.Errorf("Expected status online, got %s", found.Status)
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     prefix + "_testuser_delete",
		Email:        prefix + "_testdelete@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	err := repo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	_, err = repo.GetByID(ctx, user.ID)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound after delete, got %v", err)
	}
}

func TestUserRepository_ExistsByUsername(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	username := prefix + "_testuser_exists"
	user := &model.User{
		Username:     username,
		Email:        prefix + "_testexists@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	exists, err := repo.ExistsByUsername(ctx, username)
	if err != nil {
		t.Fatalf("Failed to check username exists: %v", err)
	}
	if !exists {
		t.Error("Expected username to exist")
	}

	exists, err = repo.ExistsByUsername(ctx, "nonexistent_user_xyz_12345")
	if err != nil {
		t.Fatalf("Failed to check username exists: %v", err)
	}
	if exists {
		t.Error("Expected username to not exist")
	}
}

func TestUserRepository_Search(t *testing.T) {
	db, prefix := setupUserTestDBIsolated(t)
	defer db.Close()
	defer cleanupUserTestByPrefix(t, db, prefix)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test users with unique names
	users := []*model.User{
		{Username: prefix + "_search_alice", Email: prefix + "_search_alice@example.com", PasswordHash: "hash", Status: model.UserStatusOffline},
		{Username: prefix + "_search_bob", Email: prefix + "_search_bob@example.com", PasswordHash: "hash", Status: model.UserStatusOffline},
		{Username: prefix + "_search_charlie", Email: prefix + "_search_charlie@example.com", PasswordHash: "hash", Status: model.UserStatusOffline},
	}

	for _, u := range users {
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	results, err := repo.Search(ctx, prefix+"_search_ali", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search users: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Username != prefix+"_search_alice" {
		t.Errorf("Expected %s_search_alice, got %s", prefix, results[0].Username)
	}
}
