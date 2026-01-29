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

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	return db
}

func cleanupTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE users CASCADE")
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db) // 測試前先清理
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser_create",
		Email:        "testcreate@example.com",
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
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &model.User{
		Username:     "testuser_getbyid",
		Email:        "testgetbyid@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	repo.Create(ctx, user)

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
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser_getbyusername",
		Email:        "testgetbyusername@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	repo.Create(ctx, user)

	found, err := repo.GetByUsername(ctx, "testuser_getbyusername")
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, found.ID)
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser_getbyemail",
		Email:        "testgetbyemail@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	repo.Create(ctx, user)

	found, err := repo.GetByEmail(ctx, "testgetbyemail@example.com")
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, found.ID)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser_update",
		Email:        "testupdate@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	repo.Create(ctx, user)

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
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser_status",
		Email:        "teststatus@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	repo.Create(ctx, user)

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
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser_delete",
		Email:        "testdelete@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	repo.Create(ctx, user)

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
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser_exists",
		Email:        "testexists@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	repo.Create(ctx, user)

	exists, err := repo.ExistsByUsername(ctx, "testuser_exists")
	if err != nil {
		t.Fatalf("Failed to check username exists: %v", err)
	}
	if !exists {
		t.Error("Expected username to exist")
	}

	exists, err = repo.ExistsByUsername(ctx, "nonexistent_user_xyz")
	if err != nil {
		t.Fatalf("Failed to check username exists: %v", err)
	}
	if exists {
		t.Error("Expected username to not exist")
	}
}

func TestUserRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test users with unique names
	users := []*model.User{
		{Username: "search_alice", Email: "search_alice@example.com", PasswordHash: "hash", Status: model.UserStatusOffline},
		{Username: "search_bob", Email: "search_bob@example.com", PasswordHash: "hash", Status: model.UserStatusOffline},
		{Username: "search_charlie", Email: "search_charlie@example.com", PasswordHash: "hash", Status: model.UserStatusOffline},
	}

	for _, u := range users {
		repo.Create(ctx, u)
	}

	results, err := repo.Search(ctx, "search_ali", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search users: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Username != "search_alice" {
		t.Errorf("Expected search_alice, got %s", results[0].Username)
	}
}
