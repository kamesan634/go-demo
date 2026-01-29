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

func setupTestAuthService(t *testing.T) (*AuthService, *sqlx.DB) {
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
	return service, db
}

func cleanupAuthTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE users CASCADE")
}

func TestAuthService_Register(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	result, err := service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
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
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	// First registration
	service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test1@example.com",
		Password: "password123",
	})

	// Second registration with same username
	_, err := service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test2@example.com",
		Password: "password123",
	})

	if err == nil {
		t.Error("Expected error for duplicate username")
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	// First registration
	service.Register(ctx, &RegisterInput{
		Username: "testuser1",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Second registration with same email
	_, err := service.Register(ctx, &RegisterInput{
		Username: "testuser2",
		Email:    "test@example.com",
		Password: "password123",
	})

	if err == nil {
		t.Error("Expected error for duplicate email")
	}
}

func TestAuthService_Login(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	// Register first
	service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Login
	result, err := service.Login(ctx, &LoginInput{
		Username: "testuser",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	if result.User.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", result.User.Username)
	}

	if result.TokenPair.AccessToken == "" {
		t.Error("Expected access token to be set")
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	// Register first
	service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Login with wrong password
	_, err := service.Login(ctx, &LoginInput{
		Username: "testuser",
		Password: "wrongpassword",
	})

	if err == nil {
		t.Error("Expected error for wrong password")
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

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
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	// Register and get tokens
	result, _ := service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

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
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	_, err := service.RefreshToken(ctx, "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid refresh token")
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	// Register
	result, _ := service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Change password
	err := service.ChangePassword(ctx, &ChangePasswordInput{
		UserID:          result.User.ID,
		CurrentPassword: "password123",
		NewPassword:     "newpassword456",
	})

	if err != nil {
		t.Fatalf("Failed to change password: %v", err)
	}

	// Login with new password
	_, err = service.Login(ctx, &LoginInput{
		Username: "testuser",
		Password: "newpassword456",
	})

	if err != nil {
		t.Error("Expected login with new password to succeed")
	}

	// Login with old password should fail
	_, err = service.Login(ctx, &LoginInput{
		Username: "testuser",
		Password: "password123",
	})

	if err == nil {
		t.Error("Expected login with old password to fail")
	}
}

func TestAuthService_ChangePassword_WrongCurrent(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	result, _ := service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	err := service.ChangePassword(ctx, &ChangePasswordInput{
		UserID:          result.User.ID,
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword456",
	})

	if err == nil {
		t.Error("Expected error for wrong current password")
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	result, _ := service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	user, err := service.ValidateToken(ctx, result.TokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	_, err := service.ValidateToken(ctx, "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestAuthService_Logout(t *testing.T) {
	service, db := setupTestAuthService(t)
	defer db.Close()
	defer cleanupAuthTestDB(t, db)

	ctx := context.Background()

	result, _ := service.Register(ctx, &RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	err := service.Logout(ctx, result.User.ID)
	if err != nil {
		t.Fatalf("Failed to logout: %v", err)
	}

	// Verify user status is offline
	user, _ := service.GetUserByID(ctx, result.User.ID)
	if user.Status != model.UserStatusOffline {
		t.Errorf("Expected status offline, got %s", user.Status)
	}
}
