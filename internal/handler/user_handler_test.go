package handler

import (
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

func setupUserHandlerTestIsolated(t *testing.T) (*gin.Engine, *service.UserService, *utils.JWTManager, *sqlx.DB, string) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	gin.SetMode(gin.TestMode)

	userRepo := repository.NewUserRepository(db)
	blockedRepo := repository.NewBlockedUserRepository(db)
	friendshipRepo := repository.NewFriendshipRepository(db)
	logger := zap.NewNop()

	userService := service.NewUserService(userRepo, blockedRepo, friendshipRepo, logger)
	jwtManager := utils.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, "test")

	handler := NewUserHandler(userService)

	router := gin.New()
	users := router.Group("/api/v1/users")
	users.Use(middleware.Auth(jwtManager))
	{
		users.GET("/search", handler.Search)
		users.GET("/online", handler.GetOnlineUsers)
		users.GET("/blocked", handler.ListBlockedUsers)
		users.GET("/friends", handler.ListFriends)
		users.GET("/friend-requests/pending", handler.ListPendingRequests)
		users.GET("/friend-requests/sent", handler.ListSentRequests)
		users.GET("/:id", handler.GetProfile)
		users.POST("/:id/block", handler.BlockUser)
		users.POST("/:id/unblock", handler.UnblockUser)
		users.POST("/:id/friend-request", handler.SendFriendRequest)
		users.POST("/:id/friend-request/accept", handler.AcceptFriendRequest)
		users.POST("/:id/friend-request/reject", handler.RejectFriendRequest)
		users.DELETE("/:id/friend", handler.RemoveFriend)
	}

	prefix := repository.GenerateUniquePrefix()
	return router, userService, jwtManager, db, prefix
}

func cleanupUserHandlerTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	repository.CleanupTestDataByPrefix(t, db, prefix)
}

func createUserForHandlerTestIsolated(t *testing.T, db *sqlx.DB, prefix, username string) *model.User {
	t.Helper()
	return repository.CreateIsolatedTestUser(t, db, prefix, username)
}

func TestUserHandler_Search(t *testing.T) {
	router, _, jwtManager, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	user := createUserForHandlerTestIsolated(t, db, prefix, "alice")
	bob := createUserForHandlerTestIsolated(t, db, prefix, "bob")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	// Search for bob using the full prefixed name
	req := httptest.NewRequest("GET", "/api/v1/users/search?q="+prefix+"_bob", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 1 {
		t.Errorf("Expected 1 result, got %d", len(data))
	}

	_ = bob // Use bob variable
}

func TestUserHandler_GetProfile(t *testing.T) {
	router, _, jwtManager, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	user := createUserForHandlerTestIsolated(t, db, prefix, "alice")
	target := createUserForHandlerTestIsolated(t, db, prefix, "bob")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/users/"+target.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUserHandler_BlockUser(t *testing.T) {
	router, _, jwtManager, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	user := createUserForHandlerTestIsolated(t, db, prefix, "alice")
	target := createUserForHandlerTestIsolated(t, db, prefix, "bob")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("POST", "/api/v1/users/"+target.ID+"/block", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUserHandler_UnblockUser(t *testing.T) {
	router, userService, jwtManager, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	user := createUserForHandlerTestIsolated(t, db, prefix, "alice")
	target := createUserForHandlerTestIsolated(t, db, prefix, "bob")

	// Block first
	userService.BlockUser(context.Background(), user.ID, target.ID)

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("POST", "/api/v1/users/"+target.ID+"/unblock", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUserHandler_ListBlockedUsers(t *testing.T) {
	router, userService, jwtManager, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	user := createUserForHandlerTestIsolated(t, db, prefix, "alice")
	target1 := createUserForHandlerTestIsolated(t, db, prefix, "bob")
	target2 := createUserForHandlerTestIsolated(t, db, prefix, "charlie")

	userService.BlockUser(context.Background(), user.ID, target1.ID)
	userService.BlockUser(context.Background(), user.ID, target2.ID)

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/users/blocked", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("Expected 2 blocked users, got %d", len(data))
	}
}

func TestUserHandler_SendFriendRequest(t *testing.T) {
	router, _, jwtManager, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	user := createUserForHandlerTestIsolated(t, db, prefix, "alice")
	friend := createUserForHandlerTestIsolated(t, db, prefix, "bob")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("POST", "/api/v1/users/"+friend.ID+"/friend-request", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUserHandler_AcceptFriendRequest(t *testing.T) {
	router, userService, jwtManager, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	user := createUserForHandlerTestIsolated(t, db, prefix, "alice")
	friend := createUserForHandlerTestIsolated(t, db, prefix, "bob")

	userService.SendFriendRequest(context.Background(), friend.ID, user.ID)

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("POST", "/api/v1/users/"+friend.ID+"/friend-request/accept", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUserHandler_ListFriends(t *testing.T) {
	router, userService, jwtManager, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	user := createUserForHandlerTestIsolated(t, db, prefix, "alice")
	friend := createUserForHandlerTestIsolated(t, db, prefix, "bob")

	userService.SendFriendRequest(context.Background(), user.ID, friend.ID)
	userService.AcceptFriendRequest(context.Background(), friend.ID, user.ID)

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/users/friends", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUserHandler_Unauthorized(t *testing.T) {
	router, _, _, db, prefix := setupUserHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupUserHandlerTestByPrefix(t, db, prefix)

	req := httptest.NewRequest("GET", "/api/v1/users/search?q=test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
