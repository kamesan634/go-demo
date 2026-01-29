package utils

import (
	"testing"
	"time"
)

func createTestManager() *JWTManager {
	return NewJWTManager(
		"test-secret-key-for-testing",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)
}

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	manager := createTestManager()

	tokenPair, err := manager.GenerateTokenPair("user-123", "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("Expected access token to be set")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Expected refresh token to be set")
	}

	if tokenPair.ExpiresAt.Before(time.Now()) {
		t.Error("Expected expiry to be in the future")
	}
}

func TestJWTManager_ValidateAccessToken(t *testing.T) {
	manager := createTestManager()

	tokenPair, _ := manager.GenerateTokenPair("user-123", "testuser")

	claims, err := manager.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
	}

	if claims.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", claims.Username)
	}

	if claims.Type != AccessToken {
		t.Errorf("Expected token type 'access', got '%s'", claims.Type)
	}
}

func TestJWTManager_ValidateRefreshToken(t *testing.T) {
	manager := createTestManager()

	tokenPair, _ := manager.GenerateTokenPair("user-123", "testuser")

	claims, err := manager.ValidateRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to validate refresh token: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("Expected user_id 'user-123', got '%s'", claims.UserID)
	}

	if claims.Type != RefreshToken {
		t.Errorf("Expected token type 'refresh', got '%s'", claims.Type)
	}
}

func TestJWTManager_ValidateAccessToken_WithRefreshToken(t *testing.T) {
	manager := createTestManager()

	tokenPair, _ := manager.GenerateTokenPair("user-123", "testuser")

	// Try to validate refresh token as access token
	_, err := manager.ValidateAccessToken(tokenPair.RefreshToken)
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken when validating refresh token as access token")
	}
}

func TestJWTManager_ValidateRefreshToken_WithAccessToken(t *testing.T) {
	manager := createTestManager()

	tokenPair, _ := manager.GenerateTokenPair("user-123", "testuser")

	// Try to validate access token as refresh token
	_, err := manager.ValidateRefreshToken(tokenPair.AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken when validating access token as refresh token")
	}
}

func TestJWTManager_ValidateToken_Invalid(t *testing.T) {
	manager := createTestManager()

	_, err := manager.ValidateToken("invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTManager_ValidateToken_Expired(t *testing.T) {
	// Create manager with very short TTL
	manager := NewJWTManager(
		"test-secret-key",
		1*time.Millisecond, // Very short TTL
		1*time.Millisecond,
		"test-issuer",
	)

	tokenPair, _ := manager.GenerateTokenPair("user-123", "testuser")

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err := manager.ValidateToken(tokenPair.AccessToken)
	if err != ErrExpiredToken {
		t.Errorf("Expected ErrExpiredToken, got %v", err)
	}
}

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	manager := createTestManager()

	token, expiresAt, err := manager.GenerateAccessToken("user-123", "testuser")
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	if token == "" {
		t.Error("Expected token to be set")
	}

	if expiresAt.Before(time.Now()) {
		t.Error("Expected expiry to be in the future")
	}
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	manager := createTestManager()

	token, expiresAt, err := manager.GenerateRefreshToken("user-123", "testuser")
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	if token == "" {
		t.Error("Expected token to be set")
	}

	if expiresAt.Before(time.Now()) {
		t.Error("Expected expiry to be in the future")
	}
}

func TestJWTManager_GetTokenID(t *testing.T) {
	manager := createTestManager()

	tokenPair, _ := manager.GenerateTokenPair("user-123", "testuser")

	tokenID, err := manager.GetTokenID(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to get token ID: %v", err)
	}

	if tokenID == "" {
		t.Error("Expected token ID to be set")
	}
}

func TestJWTManager_DifferentSecrets(t *testing.T) {
	manager1 := NewJWTManager("secret-1", 15*time.Minute, 7*24*time.Hour, "issuer")
	manager2 := NewJWTManager("secret-2", 15*time.Minute, 7*24*time.Hour, "issuer")

	tokenPair, _ := manager1.GenerateTokenPair("user-123", "testuser")

	// Token from manager1 should not be valid with manager2
	_, err := manager2.ValidateToken(tokenPair.AccessToken)
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken for token validated with different secret")
	}
}

func TestJWTManager_ClaimsContent(t *testing.T) {
	manager := createTestManager()

	tokenPair, _ := manager.GenerateTokenPair("user-123", "testuser")

	claims, _ := manager.ValidateAccessToken(tokenPair.AccessToken)

	if claims.Issuer != "test-issuer" {
		t.Errorf("Expected issuer 'test-issuer', got '%s'", claims.Issuer)
	}

	if claims.Subject != "user-123" {
		t.Errorf("Expected subject 'user-123', got '%s'", claims.Subject)
	}

	if claims.ID == "" {
		t.Error("Expected token ID (jti) to be set")
	}

	if claims.IssuedAt == nil {
		t.Error("Expected issued_at to be set")
	}

	if claims.ExpiresAt == nil {
		t.Error("Expected expires_at to be set")
	}
}
