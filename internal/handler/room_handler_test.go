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

func setupRoomHandlerTestIsolated(t *testing.T) (*gin.Engine, *service.RoomService, *utils.JWTManager, *sqlx.DB, string) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	gin.SetMode(gin.TestMode)

	roomRepo := repository.NewRoomRepository(db)
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	logger := zap.NewNop()

	roomService := service.NewRoomService(roomRepo, userRepo, messageRepo, logger)
	jwtManager := utils.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, "test")

	handler := NewRoomHandler(roomService)

	router := gin.New()
	rooms := router.Group("/api/v1/rooms")
	rooms.Use(middleware.Auth(jwtManager))
	{
		rooms.GET("", handler.ListPublic)
		rooms.POST("", handler.Create)
		rooms.GET("/me", handler.ListMyRooms)
		rooms.GET("/search", handler.Search)
		rooms.GET("/:id", handler.GetByID)
		rooms.PUT("/:id", handler.Update)
		rooms.DELETE("/:id", handler.Delete)
		rooms.POST("/:id/join", handler.Join)
		rooms.POST("/:id/leave", handler.Leave)
		rooms.POST("/:id/invite", handler.InviteMember)
		rooms.GET("/:id/members", handler.ListMembers)
	}

	prefix := repository.GenerateUniquePrefix()
	return router, roomService, jwtManager, db, prefix
}

func cleanupRoomHandlerTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	repository.CleanupTestDataByPrefix(t, db, prefix)
}

func createUserForRoomHandlerTestIsolated(t *testing.T, db *sqlx.DB, prefix, username string) *model.User {
	t.Helper()
	return repository.CreateIsolatedTestUser(t, db, prefix, username)
}

func TestRoomHandler_Create(t *testing.T) {
	router, _, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	user := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"name":        prefix + "_Test Room",
		"description": "A test room",
		"type":        "public",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/rooms", bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRoomHandler_ListPublic(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	user := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")

	// Create rooms
	_, _ = roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    prefix + "_Public Room 1",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})
	_, _ = roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    prefix + "_Public Room 2",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	// Count rooms with our prefix
	count := 0
	for _, r := range data {
		room := r.(map[string]interface{})
		name := room["name"].(string)
		if len(name) > len(prefix) && name[:len(prefix)] == prefix {
			count++
		}
	}
	if count != 2 {
		t.Errorf("Expected 2 rooms with prefix, got %d", count)
	}
}

func TestRoomHandler_GetByID(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	user := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    prefix + "_Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/"+room.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRoomHandler_Update(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	user := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    prefix + "_Original Name",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"name": prefix + "_Updated Name",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/rooms/"+room.ID, bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRoomHandler_Delete(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	user := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    prefix + "_To Delete",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("DELETE", "/api/v1/rooms/"+room.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 204 No Content 是刪除成功的標準 RESTful 回應碼
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}
}

func TestRoomHandler_Join(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	owner := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")
	member := createUserForRoomHandlerTestIsolated(t, db, prefix, "bob")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    prefix + "_Public Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(member.ID, member.Username)

	req := httptest.NewRequest("POST", "/api/v1/rooms/"+room.ID+"/join", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRoomHandler_Leave(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	owner := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")
	member := createUserForRoomHandlerTestIsolated(t, db, prefix, "bob")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    prefix + "_Public Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	_ = roomService.Join(context.Background(), room.ID, member.ID)

	tokenPair, _ := jwtManager.GenerateTokenPair(member.ID, member.Username)

	req := httptest.NewRequest("POST", "/api/v1/rooms/"+room.ID+"/leave", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRoomHandler_ListMembers(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	owner := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")
	member := createUserForRoomHandlerTestIsolated(t, db, prefix, "bob")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    prefix + "_Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	_ = roomService.Join(context.Background(), room.ID, member.ID)

	tokenPair, _ := jwtManager.GenerateTokenPair(owner.ID, owner.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/"+room.ID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("Expected 2 members, got %d", len(data))
	}
}

func TestRoomHandler_Search(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	user := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")

	_, _ = roomService.Create(context.Background(), &service.CreateRoomInput{Name: prefix + "_Tech Talk", Type: model.RoomTypePublic, OwnerID: user.ID})
	_, _ = roomService.Create(context.Background(), &service.CreateRoomInput{Name: prefix + "_General", Type: model.RoomTypePublic, OwnerID: user.ID})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/search?q="+prefix+"_Tech", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 1 {
		t.Errorf("Expected 1 room, got %d", len(data))
	}
}

func TestRoomHandler_ListMyRooms(t *testing.T) {
	router, roomService, jwtManager, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	user := createUserForRoomHandlerTestIsolated(t, db, prefix, "alice")

	_, _ = roomService.Create(context.Background(), &service.CreateRoomInput{Name: prefix + "_Room 1", Type: model.RoomTypePublic, OwnerID: user.ID})
	_, _ = roomService.Create(context.Background(), &service.CreateRoomInput{Name: prefix + "_Room 2", Type: model.RoomTypePublic, OwnerID: user.ID})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(data))
	}
}

func TestRoomHandler_Unauthorized(t *testing.T) {
	router, _, _, db, prefix := setupRoomHandlerTestIsolated(t)
	defer db.Close()
	defer cleanupRoomHandlerTestByPrefix(t, db, prefix)

	req := httptest.NewRequest("GET", "/api/v1/rooms", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
