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

func setupMessageHandlerTest(t *testing.T) (*gin.Engine, *service.MessageService, *service.RoomService, *service.DirectMessageService, *utils.JWTManager, *sqlx.DB) {
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
	dmRepo := repository.NewDirectMessageRepository(db)
	blockedRepo := repository.NewBlockedUserRepository(db)
	logger := zap.NewNop()

	roomService := service.NewRoomService(roomRepo, userRepo, messageRepo, logger)
	messageService := service.NewMessageService(messageRepo, roomRepo, logger)
	dmService := service.NewDirectMessageService(dmRepo, userRepo, blockedRepo, logger)
	jwtManager := utils.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, "test")

	handler := NewMessageHandler(messageService, roomService, dmService)

	router := gin.New()
	rooms := router.Group("/api/v1/rooms")
	rooms.Use(middleware.Auth(jwtManager))
	{
		rooms.GET("/:room_id/messages", handler.GetMessages)
		rooms.POST("/:room_id/messages", handler.SendMessage)
		rooms.PUT("/:room_id/messages/:message_id", handler.UpdateMessage)
		rooms.DELETE("/:room_id/messages/:message_id", handler.DeleteMessage)
		rooms.GET("/:room_id/messages/search", handler.SearchMessages)
	}

	dm := router.Group("/api/v1/dm")
	dm.Use(middleware.Auth(jwtManager))
	{
		dm.GET("", handler.ListConversations)
		dm.GET("/unread", handler.GetUnreadCount)
		dm.GET("/:user_id", handler.GetConversation)
		dm.POST("/:user_id", handler.SendDirectMessage)
		dm.POST("/:user_id/read", handler.MarkDMAsRead)
	}

	return router, messageService, roomService, dmService, jwtManager, db
}

func cleanupMessageHandlerTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE messages, direct_messages, rooms, room_members, users CASCADE")
}

func createUserForMsgHandlerTest(t *testing.T, db *sqlx.DB, username string) *model.User {
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

func TestMessageHandler_SendMessage(t *testing.T) {
	router, _, roomService, _, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	user := createUserForMsgHandlerTest(t, db, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"content": "Hello, World!",
		"type":    "text",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/rooms/"+room.ID+"/messages", bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMessageHandler_GetMessages(t *testing.T) {
	router, messageService, roomService, _, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	user := createUserForMsgHandlerTest(t, db, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	// Send some messages
	messageService.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID: room.ID, UserID: user.ID, Content: "Message 1", Type: model.MessageTypeText,
	})
	messageService.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID: room.ID, UserID: user.ID, Content: "Message 2", Type: model.MessageTypeText,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/"+room.ID+"/messages", nil)
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
		t.Errorf("Expected 2 messages, got %d", len(data))
	}
}

func TestMessageHandler_UpdateMessage(t *testing.T) {
	router, messageService, roomService, _, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	user := createUserForMsgHandlerTest(t, db, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	msg, _ := messageService.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID: room.ID, UserID: user.ID, Content: "Original", Type: model.MessageTypeText,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	body := map[string]interface{}{
		"content": "Updated",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/v1/rooms/"+room.ID+"/messages/"+msg.ID, bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMessageHandler_DeleteMessage(t *testing.T) {
	router, messageService, roomService, _, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	user := createUserForMsgHandlerTest(t, db, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	msg, _ := messageService.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID: room.ID, UserID: user.ID, Content: "To delete", Type: model.MessageTypeText,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("DELETE", "/api/v1/rooms/"+room.ID+"/messages/"+msg.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Accept both 200 and 204 as valid responses for delete
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("Expected status 200 or 204, got %d", w.Code)
	}
}

func TestMessageHandler_SearchMessages(t *testing.T) {
	router, messageService, roomService, _, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	user := createUserForMsgHandlerTest(t, db, "alice")

	room, _ := roomService.Create(context.Background(), &service.CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	messageService.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID: room.ID, UserID: user.ID, Content: "Hello World", Type: model.MessageTypeText,
	})
	messageService.SendMessage(context.Background(), &service.SendMessageInput{
		RoomID: room.ID, UserID: user.ID, Content: "Golang is great", Type: model.MessageTypeText,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/rooms/"+room.ID+"/messages/search?q=Golang", nil)
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
		t.Errorf("Expected 1 message, got %d", len(data))
	}
}

func TestMessageHandler_SendDirectMessage(t *testing.T) {
	router, _, _, _, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	sender := createUserForMsgHandlerTest(t, db, "alice")
	receiver := createUserForMsgHandlerTest(t, db, "bob")

	tokenPair, _ := jwtManager.GenerateTokenPair(sender.ID, sender.Username)

	body := map[string]interface{}{
		"content": "Hello Bob!",
		"type":    "text",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/dm/"+receiver.ID, bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMessageHandler_GetConversation(t *testing.T) {
	router, _, _, dmService, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	user1 := createUserForMsgHandlerTest(t, db, "alice")
	user2 := createUserForMsgHandlerTest(t, db, "bob")

	dmService.SendMessage(context.Background(), &service.SendDMInput{
		SenderID: user1.ID, ReceiverID: user2.ID, Content: "Hi Bob", Type: model.MessageTypeText,
	})
	dmService.SendMessage(context.Background(), &service.SendDMInput{
		SenderID: user2.ID, ReceiverID: user1.ID, Content: "Hi Alice", Type: model.MessageTypeText,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user1.ID, user1.Username)

	req := httptest.NewRequest("GET", "/api/v1/dm/"+user2.ID, nil)
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
		t.Errorf("Expected 2 messages, got %d", len(data))
	}
}

func TestMessageHandler_ListConversations(t *testing.T) {
	router, _, _, dmService, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	user := createUserForMsgHandlerTest(t, db, "alice")
	contact1 := createUserForMsgHandlerTest(t, db, "bob")
	contact2 := createUserForMsgHandlerTest(t, db, "charlie")

	dmService.SendMessage(context.Background(), &service.SendDMInput{
		SenderID: contact1.ID, ReceiverID: user.ID, Content: "Hi", Type: model.MessageTypeText,
	})
	dmService.SendMessage(context.Background(), &service.SendDMInput{
		SenderID: contact2.ID, ReceiverID: user.ID, Content: "Hey", Type: model.MessageTypeText,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/dm", nil)
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
		t.Errorf("Expected 2 conversations, got %d", len(data))
	}
}

func TestMessageHandler_GetUnreadCount(t *testing.T) {
	router, _, _, dmService, jwtManager, db := setupMessageHandlerTest(t)
	defer db.Close()
	defer cleanupMessageHandlerTestDB(t, db)

	user := createUserForMsgHandlerTest(t, db, "alice")
	sender := createUserForMsgHandlerTest(t, db, "bob")

	// Send some unread messages
	dmService.SendMessage(context.Background(), &service.SendDMInput{
		SenderID: sender.ID, ReceiverID: user.ID, Content: "Msg 1", Type: model.MessageTypeText,
	})
	dmService.SendMessage(context.Background(), &service.SendDMInput{
		SenderID: sender.ID, ReceiverID: user.ID, Content: "Msg 2", Type: model.MessageTypeText,
	})

	tokenPair, _ := jwtManager.GenerateTokenPair(user.ID, user.Username)

	req := httptest.NewRequest("GET", "/api/v1/dm/unread", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %v", response)
	}

	unreadCount, ok := data["count"]
	if !ok {
		t.Fatalf("Missing count in response: %v", data)
	}

	count := int(unreadCount.(float64))
	if count < 0 {
		t.Errorf("Expected non-negative unread count, got %d", count)
	}
}
