package service

import (
	"context"
	"testing"
	"time"

	"github.com/go-demo/chat/internal/model"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/go-demo/chat/internal/repository"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func setupTestAuthServiceIsolated(t *testing.T) (*AuthService, *sqlx.DB, string) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	jwtManager := utils.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, "test")
	logger := zap.NewNop()

	service := NewAuthService(userRepo, jwtManager, logger)
	prefix := repository.GenerateUniquePrefix()
	return service, db, prefix
}

func cleanupAuthTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	repository.CleanupTestDataByPrefix(t, db, prefix)
}

func TestAuthService_Register(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()

	result, err := service.Register(ctx, &RegisterInput{
		Username: prefix + "_testuser",
		Email:    prefix + "_test@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	if result.User.ID == "" {
		t.Error("Expected user ID to be set")
	}

	if result.TokenPair.AccessToken == "" {
		t.Error("Expected access token to be set")
	}

	if result.TokenPair.RefreshToken == "" {
		t.Error("Expected refresh token to be set")
	}
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()
	username := prefix + "_dupuser"

	// First registration
	_, err := service.Register(ctx, &RegisterInput{
		Username: username,
		Email:    prefix + "_test1@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register first user: %v", err)
	}

	// Second registration with same username
	_, err = service.Register(ctx, &RegisterInput{
		Username: username,
		Email:    prefix + "_test2@example.com",
		Password: "password123",
	})

	if err == nil {
		t.Error("Expected error for duplicate username")
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()
	email := prefix + "_test@example.com"

	// First registration
	_, err := service.Register(ctx, &RegisterInput{
		Username: prefix + "_testuser1",
		Email:    email,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register first user: %v", err)
	}

	// Second registration with same email
	_, err = service.Register(ctx, &RegisterInput{
		Username: prefix + "_testuser2",
		Email:    email,
		Password: "password123",
	})

	if err == nil {
		t.Error("Expected error for duplicate email")
	}
}

func TestAuthService_Login(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()
	username := prefix + "_testuser"

	// Register first
	_, err := service.Register(ctx, &RegisterInput{
		Username: username,
		Email:    prefix + "_test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Login
	result, err := service.Login(ctx, &LoginInput{
		Username: username,
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	if result.User.Username != username {
		t.Errorf("Expected username '%s', got '%s'", username, result.User.Username)
	}

	if result.TokenPair.AccessToken == "" {
		t.Error("Expected access token to be set")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()
	username := prefix + "_testuser"

	// Register first
	_, err := service.Register(ctx, &RegisterInput{
		Username: username,
		Email:    prefix + "_test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Login with wrong password
	_, err = service.Login(ctx, &LoginInput{
		Username: username,
		Password: "wrongpassword",
	})

	if err == nil {
		t.Error("Expected error for wrong password")
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()

	_, err := service.Login(ctx, &LoginInput{
		Username: "nonexistent",
		Password: "password123",
	})

	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()

	// Register and get tokens
	result, err := service.Register(ctx, &RegisterInput{
		Username: prefix + "_testuser",
		Email:    prefix + "_test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Refresh token
	newTokenPair, err := service.RefreshToken(ctx, result.TokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if newTokenPair.AccessToken == "" {
		t.Error("Expected new access token")
	}

	if newTokenPair.RefreshToken == "" {
		t.Error("Expected new refresh token")
	}
}

func TestAuthService_RefreshToken_Invalid(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()

	_, err := service.RefreshToken(ctx, "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid refresh token")
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()
	username := prefix + "_changepwd"

	// Register
	result, err := service.Register(ctx, &RegisterInput{
		Username: username,
		Email:    prefix + "_changepwd@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Change password
	err = service.ChangePassword(ctx, &ChangePasswordInput{
		UserID:          result.User.ID,
		CurrentPassword: "password123",
		NewPassword:     "newpassword456",
	})

	if err != nil {
		t.Fatalf("Failed to change password: %v", err)
	}

	// Login with new password
	_, err = service.Login(ctx, &LoginInput{
		Username: username,
		Password: "newpassword456",
	})

	if err != nil {
		t.Error("Expected login with new password to succeed")
	}

	// Login with old password should fail
	_, err = service.Login(ctx, &LoginInput{
		Username: username,
		Password: "password123",
	})

	if err == nil {
		t.Error("Expected login with old password to fail")
	}
}

func TestAuthService_ChangePassword_WrongCurrent(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()
	username := prefix + "_changepwd_wrong"

	result, err := service.Register(ctx, &RegisterInput{
		Username: username,
		Email:    prefix + "_changepwd_wrong@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	err = service.ChangePassword(ctx, &ChangePasswordInput{
		UserID:          result.User.ID,
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword456",
	})

	if err == nil {
		t.Error("Expected error for wrong current password")
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()
	username := prefix + "_testuser"

	result, err := service.Register(ctx, &RegisterInput{
		Username: username,
		Email:    prefix + "_test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	user, err := service.ValidateToken(ctx, result.TokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if user.Username != username {
		t.Errorf("Expected username '%s', got '%s'", username, user.Username)
	}
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()

	_, err := service.ValidateToken(ctx, "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestAuthService_Logout(t *testing.T) {
	service, db, prefix := setupTestAuthServiceIsolated(t)
	defer db.Close()
	defer cleanupAuthTestByPrefix(t, db, prefix)

	ctx := context.Background()

	result, err := service.Register(ctx, &RegisterInput{
		Username: prefix + "_testuser",
		Email:    prefix + "_test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	err = service.Logout(ctx, result.User.ID)
	if err != nil {
		t.Fatalf("Failed to logout: %v", err)
	}

	// Verify user status is offline
	user, _ := service.GetUserByID(ctx, result.User.ID)
	if user.Status != model.UserStatusOffline {
		t.Errorf("Expected status offline, got %s", user.Status)
	}
}
