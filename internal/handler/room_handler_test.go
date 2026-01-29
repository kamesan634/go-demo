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

func setupRoomHandlerTest(t *testing.T) (*gin.Engine, *service.RoomService, *utils.JWTManager, *sqlx.DB) {
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

	return router, roomService, jwtManager, db
}

func cleanupRoomHandlerTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE rooms, room_members, users, messages CASCADE")
}

func createUserForRoomHandlerTest(t *testing.T, db *sqlx.DB, username string) *model.User {
	t.Helper()
	userRepo := repository.NewUserRepository(db)
	user := &model.User{
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

func TestRoomHandler_Create(t *testing.T) {
	router, _, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	user := createUserForRoomHandlerTest(t, db, "alice")

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"name":        "Test Room",
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
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	user := createUserForRoomHandlerTest(t, db, "alice")

	// Create rooms
	roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Public Room 1",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})
	roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Public Room 2",
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
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(data))
	}
}

func TestRoomHandler_GetByID(t *testing.T) {
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	user := createUserForRoomHandlerTest(t, db, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Test Room",
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
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	user := createUserForRoomHandlerTest(t, db, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Original Name",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"name": "Updated Name",
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
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	user := createUserForRoomHandlerTest(t, db, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "To Delete",
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
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	owner := createUserForRoomHandlerTest(t, db, "alice")
	member := createUserForRoomHandlerTest(t, db, "bob")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Public Room",
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
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	owner := createUserForRoomHandlerTest(t, db, "alice")
	member := createUserForRoomHandlerTest(t, db, "bob")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Public Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	roomService.Join(context.Background(), room.ID, member.ID)

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
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	owner := createUserForRoomHandlerTest(t, db, "alice")
	member := createUserForRoomHandlerTest(t, db, "bob")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	roomService.Join(context.Background(), room.ID, member.ID)

	tokenPair, _ := jwtManager.GenerateTokenPair(owner.ID, owner.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/"+room.ID+"/members", nil)
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
		t.Errorf("Expected 2 members, got %d", len(data))
	}
}

func TestRoomHandler_Search(t *testing.T) {
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	user := createUserForRoomHandlerTest(t, db, "alice")

	roomService.Create(context.Background(), &service.CreateRoomInput{Name: "Tech Talk", Type: model.RoomTypePublic, OwnerID: user.ID})
	roomService.Create(context.Background(), &service.CreateRoomInput{Name: "General", Type: model.RoomTypePublic, OwnerID: user.ID})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/search?q=Tech", nil)
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
		t.Errorf("Expected 1 room, got %d", len(data))
	}
}

func TestRoomHandler_ListMyRooms(t *testing.T) {
	router, roomService, jwtManager, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	user := createUserForRoomHandlerTest(t, db, "alice")

	roomService.Create(context.Background(), &service.CreateRoomInput{Name: "Room 1", Type: model.RoomTypePublic, OwnerID: user.ID})
	roomService.Create(context.Background(), &service.CreateRoomInput{Name: "Room 2", Type: model.RoomTypePublic, OwnerID: user.ID})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/me", nil)
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
		t.Errorf("Expected 2 rooms, got %d", len(data))
	}
}

func TestRoomHandler_Unauthorized(t *testing.T) {
	router, _, _, db := setupRoomHandlerTest(t)
	defer db.Close()
	defer cleanupRoomHandlerTestDB(t, db)

	req := httptest.NewRequest("GET", "/api/v1/rooms", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}
