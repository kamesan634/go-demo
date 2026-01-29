package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/middleware"
	"github.com/go-demo/chat/internal/model"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/go-demo/chat/internal/repository"
	"github.com/go-demo/chat/internal/service"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func setupAuthHandlerTest(t *testing.T) (*gin.Engine, *service.AuthService, *utils.JWTManager, *sqlx.DB) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	gin.SetMode(gin.TestMode)

	userRepo := repository.NewUserRepository(db)
	logger := zap.NewNop()
	jwtManager := utils.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, "test")

	authService := service.NewAuthService(userRepo, jwtManager, logger)
	handler := NewAuthHandler(authService)

	router := gin.New()

	// Public routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.RefreshToken)
	}

	// Protected routes
	authProtected := router.Group("/api/v1/auth")
	authProtected.Use(middleware.Auth(jwtManager))
	{
		authProtected.POST("/logout", handler.Logout)
		authProtected.GET("/me", handler.GetMe)
		authProtected.PUT("/password", handler.ChangePassword)
		authProtected.PUT("/profile", handler.UpdateProfile)
	}

	return router, authService, jwtManager, db
}

func cleanupAuthHandlerTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE users CASCADE")
}

func createUserForAuthHandlerTest(t *testing.T, db *sqlx.DB, username, password string) *model.User {
	t.Helper()
	userRepo := repository.NewUserRepository(db)

	hashedPassword, _ := utils.HashPassword(password)
	user := &model.User{
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: hashedPassword,
		Status:       model.UserStatusOffline,
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

func TestAuthHandler_Register(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	body := map[string]interface{}{
		"username": "newuser",
		"email":    "newuser@example.com",
		"password": "Password123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	if data["user"] == nil {
		t.Error("Expected user in response")
	}
	if data["token"] == nil {
		t.Error("Expected token in response")
	}
}

func TestAuthHandler_Register_DuplicateUsername(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	// Create existing user
	createUserForAuthHandlerTest(t, db, "existinguser", "password123")

	body := map[string]interface{}{
		"username": "existinguser",
		"email":    "new@example.com",
		"password": "Password123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate username, got %d", w.Code)
	}
}

func TestAuthHandler_Register_InvalidUsername(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	body := map[string]interface{}{
		"username": "ab",  // Too short
		"email":    "test@example.com",
		"password": "Password123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid username, got %d", w.Code)
	}
}

func TestAuthHandler_Register_InvalidEmail(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	body := map[string]interface{}{
		"username": "testuser",
		"email":    "invalid-email",
		"password": "Password123!",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid email, got %d", w.Code)
	}
}

func TestAuthHandler_Register_WeakPassword(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	body := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "weak",  // Too weak
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for weak password, got %d", w.Code)
	}
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestAuthHandler_Login(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	createUserForAuthHandlerTest(t, db, "loginuser", "password123")

	body := map[string]interface{}{
		"username": "loginuser",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	if data["token"] == nil {
		t.Error("Expected token in response")
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	createUserForAuthHandlerTest(t, db, "loginuser", "password123")

	body := map[string]interface{}{
		"username": "loginuser",
		"password": "wrongpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for wrong password, got %d", w.Code)
	}
}

func TestAuthHandler_Login_UserNotFound(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	body := map[string]interface{}{
		"username": "nonexistent",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for non-existent user, got %d", w.Code)
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestAuthHandler_GetMe(t *testing.T) {
	router, _, jwtManager, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	user := createUserForAuthHandlerTest(t, db, "alice", "password123")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	if data["username"] != "alice" {
		t.Errorf("Expected username 'alice', got '%v'", data["username"])
	}
}

func TestAuthHandler_GetMe_Unauthorized(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	router, _, jwtManager, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	user := createUserForAuthHandlerTest(t, db, "alice", "password123")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthHandler_Logout_Unauthorized(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	router, _, jwtManager, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	user := createUserForAuthHandlerTest(t, db, "alice", "password123")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"refresh_token": tokenPair.RefreshToken,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	if data["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
}

func TestAuthHandler_RefreshToken_InvalidToken(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	body := map[string]interface{}{
		"refresh_token": "invalid-token",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid token, got %d", w.Code)
	}
}

func TestAuthHandler_RefreshToken_InvalidJSON(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	router, _, jwtManager, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	user := createUserForAuthHandlerTest(t, db, "alice", "oldpassword123")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"current_password": "oldpassword123",
		"new_password":     "newpassword456",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_ChangePassword_WrongCurrentPassword(t *testing.T) {
	router, _, jwtManager, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	user := createUserForAuthHandlerTest(t, db, "alice", "oldpassword123")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"current_password": "wrongpassword",
		"new_password":     "newpassword456",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for wrong current password, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePassword_Unauthorized(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	body := map[string]interface{}{
		"current_password": "old",
		"new_password":     "new",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/auth/password", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_UpdateProfile(t *testing.T) {
	router, _, jwtManager, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	user := createUserForAuthHandlerTest(t, db, "alice", "password123")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	displayName := "Alice Wonderland"
	body := map[string]interface{}{
		"display_name": displayName,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_UpdateProfile_Unauthorized(t *testing.T) {
	router, _, _, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	body := map[string]interface{}{
		"display_name": "New Name",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthHandler_UpdateProfile_InvalidJSON(t *testing.T) {
	router, _, jwtManager, db := setupAuthHandlerTest(t)
	defer db.Close()
	defer cleanupAuthHandlerTestDB(t, db)

	user := createUserForAuthHandlerTest(t, db, "alice", "password123")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("PUT", "/api/v1/auth/profile", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}
