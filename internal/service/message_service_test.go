package service

import (
	"context"
	"testing"

	"github.com/go-demo/chat/internal/model"
	"github.com/go-demo/chat/internal/repository"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func setupTestMessageService(t *testing.T) (*MessageService, *RoomService, *sqlx.DB) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	messageRepo := repository.NewMessageRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	userRepo := repository.NewUserRepository(db)
	logger := zap.NewNop()

	messageService := NewMessageService(messageRepo, roomRepo, logger)
	roomService := NewRoomService(roomRepo, userRepo, messageRepo, logger)

	return messageService, roomService, db
}

func cleanupMessageServiceTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE messages, rooms, room_members, users CASCADE")
}

func createUserForMessageServiceTest(t *testing.T, db *sqlx.DB, username string) *model.User {
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

func TestMessageService_SendMessage(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	msg, err := msgService.SendMessage(ctx, &SendMessageInput{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Hello, World!",
		Type:    model.MessageTypeText,
	})

	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if msg.ID == "" {
		t.Error("Expected message ID to be set")
	}
	if msg.Content != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got '%s'", msg.Content)
	}
}

func TestMessageService_SendMessage_NotMember(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	owner := createUserForMessageServiceTest(t, db, "owner")
	nonMember := createUserForMessageServiceTest(t, db, "nonmember")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	_, err := msgService.SendMessage(ctx, &SendMessageInput{
		RoomID:  room.ID,
		UserID:  nonMember.ID,
		Content: "Hello",
		Type:    model.MessageTypeText,
	})

	if err == nil {
		t.Error("Expected permission denied for non-member")
	}
}

func TestMessageService_GetByID(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	sent, _ := msgService.SendMessage(ctx, &SendMessageInput{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Test message",
		Type:    model.MessageTypeText,
	})

	found, err := msgService.GetByID(ctx, sent.ID)
	if err != nil {
		t.Fatalf("Failed to get message: %v", err)
	}

	if found.Content != sent.Content {
		t.Errorf("Expected content '%s', got '%s'", sent.Content, found.Content)
	}
}

func TestMessageService_UpdateMessage(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	sent, _ := msgService.SendMessage(ctx, &SendMessageInput{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Original message",
		Type:    model.MessageTypeText,
	})

	updated, err := msgService.UpdateMessage(ctx, sent.ID, user.ID, "Updated message")
	if err != nil {
		t.Fatalf("Failed to update message: %v", err)
	}

	if updated.Content != "Updated message" {
		t.Errorf("Expected content 'Updated message', got '%s'", updated.Content)
	}
	if !updated.IsEdited {
		t.Error("Expected is_edited to be true")
	}
}

func TestMessageService_UpdateMessage_NotOwner(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	owner := createUserForMessageServiceTest(t, db, "owner")
	otherUser := createUserForMessageServiceTest(t, db, "other")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	roomService.Join(ctx, room.ID, otherUser.ID)

	sent, _ := msgService.SendMessage(ctx, &SendMessageInput{
		RoomID:  room.ID,
		UserID:  owner.ID,
		Content: "Original message",
		Type:    model.MessageTypeText,
	})

	_, err := msgService.UpdateMessage(ctx, sent.ID, otherUser.ID, "Trying to update")
	if err == nil {
		t.Error("Expected permission denied for non-owner")
	}
}

func TestMessageService_DeleteMessage(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	sent, _ := msgService.SendMessage(ctx, &SendMessageInput{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "To be deleted",
		Type:    model.MessageTypeText,
	})

	err := msgService.DeleteMessage(ctx, sent.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete message: %v", err)
	}

	found, _ := msgService.GetByID(ctx, sent.ID)
	if !found.IsDeleted {
		t.Error("Expected message to be deleted")
	}
}

func TestMessageService_ListByRoomID(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	// Send 5 messages
	for i := 0; i < 5; i++ {
		msgService.SendMessage(ctx, &SendMessageInput{
			RoomID:  room.ID,
			UserID:  user.ID,
			Content: "Message",
			Type:    model.MessageTypeText,
		})
	}

	messages, err := msgService.ListByRoomID(ctx, room.ID, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	if len(messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(messages))
	}
}

func TestMessageService_ListByRoomID_Pagination(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	// Send 10 messages
	for i := 0; i < 10; i++ {
		msgService.SendMessage(ctx, &SendMessageInput{
			RoomID:  room.ID,
			UserID:  user.ID,
			Content: "Message",
			Type:    model.MessageTypeText,
		})
	}

	page1, _ := msgService.ListByRoomID(ctx, room.ID, user.ID, 5, 0)
	page2, _ := msgService.ListByRoomID(ctx, room.ID, user.ID, 5, 5)

	if len(page1) != 5 {
		t.Errorf("Expected 5 messages on page 1, got %d", len(page1))
	}
	if len(page2) != 5 {
		t.Errorf("Expected 5 messages on page 2, got %d", len(page2))
	}
}

func TestMessageService_Search(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	msgService.SendMessage(ctx, &SendMessageInput{RoomID: room.ID, UserID: user.ID, Content: "Hello World", Type: model.MessageTypeText})
	msgService.SendMessage(ctx, &SendMessageInput{RoomID: room.ID, UserID: user.ID, Content: "Golang is great", Type: model.MessageTypeText})
	msgService.SendMessage(ctx, &SendMessageInput{RoomID: room.ID, UserID: user.ID, Content: "Testing", Type: model.MessageTypeText})

	results, err := msgService.Search(ctx, room.ID, user.ID, "Golang", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search messages: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestMessageService_SendMessage_WithReply(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	original, _ := msgService.SendMessage(ctx, &SendMessageInput{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Original message",
		Type:    model.MessageTypeText,
	})

	reply, err := msgService.SendMessage(ctx, &SendMessageInput{
		RoomID:    room.ID,
		UserID:    user.ID,
		Content:   "This is a reply",
		Type:      model.MessageTypeText,
		ReplyToID: original.ID,
	})

	if err != nil {
		t.Fatalf("Failed to send reply: %v", err)
	}

	if reply.GetReplyToID() != original.ID {
		t.Errorf("Expected reply_to_id '%s', got '%s'", original.ID, reply.GetReplyToID())
	}
}

func TestMessageService_SendMessage_MessageTypes(t *testing.T) {
	msgService, roomService, db := setupTestMessageService(t)
	defer db.Close()
	defer cleanupMessageServiceTestDB(t, db)

	user := createUserForMessageServiceTest(t, db, "sender")
	ctx := context.Background()

	room, _ := roomService.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: user.ID,
	})

	types := []model.MessageType{
		model.MessageTypeText,
		model.MessageTypeImage,
		model.MessageTypeFile,
	}

	for _, msgType := range types {
		msg, err := msgService.SendMessage(ctx, &SendMessageInput{
			RoomID:  room.ID,
			UserID:  user.ID,
			Content: "Message of type " + string(msgType),
			Type:    msgType,
		})

		if err != nil {
			t.Errorf("Failed to send message of type %s: %v", msgType, err)
		}

		if msg.Type != msgType {
			t.Errorf("Expected type '%s', got '%s'", msgType, msg.Type)
		}
	}
}
