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

func setupTestDMService(t *testing.T) (*DirectMessageService, *sqlx.DB) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	dmRepo := repository.NewDirectMessageRepository(db)
	userRepo := repository.NewUserRepository(db)
	blockedRepo := repository.NewBlockedUserRepository(db)
	logger := zap.NewNop()

	service := NewDirectMessageService(dmRepo, userRepo, blockedRepo, logger)
	return service, db
}

func cleanupDMServiceTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE direct_messages, users, blocked_users CASCADE")
}

func createUserForDMServiceTest(t *testing.T, db *sqlx.DB, username string) *model.User {
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

func TestDirectMessageService_SendMessage(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	ctx := context.Background()

	msg, err := service.SendMessage(ctx, &SendDMInput{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Hello!",
		Type:       model.MessageTypeText,
	})

	if err != nil {
		t.Fatalf("Failed to send direct message: %v", err)
	}

	if msg.ID == "" {
		t.Error("Expected message ID to be set")
	}
	if msg.Content != "Hello!" {
		t.Errorf("Expected content 'Hello!', got '%s'", msg.Content)
	}
}

func TestDirectMessageService_SendMessage_ToSelf(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	user := createUserForDMServiceTest(t, db, "user")
	ctx := context.Background()

	_, err := service.SendMessage(ctx, &SendDMInput{
		SenderID:   user.ID,
		ReceiverID: user.ID,
		Content:    "Hello self",
		Type:       model.MessageTypeText,
	})

	if err == nil {
		t.Error("Expected error when sending message to self")
	}
}

func TestDirectMessageService_SendMessage_ToNonExistent(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	ctx := context.Background()

	_, err := service.SendMessage(ctx, &SendDMInput{
		SenderID:   sender.ID,
		ReceiverID: "non-existent-id",
		Content:    "Hello",
		Type:       model.MessageTypeText,
	})

	if err == nil {
		t.Error("Expected error when sending to non-existent user")
	}
}

func TestDirectMessageService_SendMessage_Blocked(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	blockedRepo := repository.NewBlockedUserRepository(db)
	ctx := context.Background()

	// Receiver blocks sender
	blockedRepo.Block(ctx, receiver.ID, sender.ID)

	_, err := service.SendMessage(ctx, &SendDMInput{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Hello",
		Type:       model.MessageTypeText,
	})

	if err == nil {
		t.Error("Expected error when sending to user who blocked you")
	}
}

func TestDirectMessageService_GetConversation(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	user1 := createUserForDMServiceTest(t, db, "user1")
	user2 := createUserForDMServiceTest(t, db, "user2")
	ctx := context.Background()

	// Send messages back and forth
	service.SendMessage(ctx, &SendDMInput{SenderID: user1.ID, ReceiverID: user2.ID, Content: "Hi", Type: model.MessageTypeText})
	service.SendMessage(ctx, &SendDMInput{SenderID: user2.ID, ReceiverID: user1.ID, Content: "Hello", Type: model.MessageTypeText})
	service.SendMessage(ctx, &SendDMInput{SenderID: user1.ID, ReceiverID: user2.ID, Content: "How are you?", Type: model.MessageTypeText})

	conversation, err := service.GetConversation(ctx, user1.ID, user2.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	if len(conversation) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(conversation))
	}
}

func TestDirectMessageService_GetConversation_NonExistentUser(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	user := createUserForDMServiceTest(t, db, "user")
	ctx := context.Background()

	_, err := service.GetConversation(ctx, user.ID, "non-existent-id", 10, 0)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestDirectMessageService_ListConversations(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	user := createUserForDMServiceTest(t, db, "user")
	contact1 := createUserForDMServiceTest(t, db, "contact1")
	contact2 := createUserForDMServiceTest(t, db, "contact2")
	ctx := context.Background()

	service.SendMessage(ctx, &SendDMInput{SenderID: contact1.ID, ReceiverID: user.ID, Content: "Hi from contact1", Type: model.MessageTypeText})
	service.SendMessage(ctx, &SendDMInput{SenderID: contact2.ID, ReceiverID: user.ID, Content: "Hi from contact2", Type: model.MessageTypeText})

	conversations, err := service.ListConversations(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(conversations))
	}
}

func TestDirectMessageService_MarkAsRead(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	ctx := context.Background()

	// Send unread messages
	service.SendMessage(ctx, &SendDMInput{SenderID: sender.ID, ReceiverID: receiver.ID, Content: "Msg 1", Type: model.MessageTypeText})
	service.SendMessage(ctx, &SendDMInput{SenderID: sender.ID, ReceiverID: receiver.ID, Content: "Msg 2", Type: model.MessageTypeText})

	err := service.MarkAsRead(ctx, receiver.ID, sender.ID)
	if err != nil {
		t.Fatalf("Failed to mark as read: %v", err)
	}

	// Verify unread count is 0
	count, _ := service.CountUnreadFromUser(ctx, receiver.ID, sender.ID)
	if count != 0 {
		t.Errorf("Expected 0 unread messages, got %d", count)
	}
}

func TestDirectMessageService_DeleteMessage(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	ctx := context.Background()

	msg, _ := service.SendMessage(ctx, &SendDMInput{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "To delete",
		Type:       model.MessageTypeText,
	})

	err := service.DeleteMessage(ctx, msg.ID, sender.ID)
	if err != nil {
		t.Fatalf("Failed to delete message: %v", err)
	}
}

func TestDirectMessageService_CountUnread(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	ctx := context.Background()

	// Send 3 unread messages
	for i := 0; i < 3; i++ {
		service.SendMessage(ctx, &SendDMInput{
			SenderID:   sender.ID,
			ReceiverID: receiver.ID,
			Content:    "Unread",
			Type:       model.MessageTypeText,
		})
	}

	count, err := service.CountUnread(ctx, receiver.ID)
	if err != nil {
		t.Fatalf("Failed to count unread: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 unread messages, got %d", count)
	}
}

func TestDirectMessageService_CountUnreadFromUser(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender1 := createUserForDMServiceTest(t, db, "sender1")
	sender2 := createUserForDMServiceTest(t, db, "sender2")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	ctx := context.Background()

	// Send messages from sender1
	for i := 0; i < 3; i++ {
		service.SendMessage(ctx, &SendDMInput{SenderID: sender1.ID, ReceiverID: receiver.ID, Content: "From s1", Type: model.MessageTypeText})
	}

	// Send messages from sender2
	for i := 0; i < 2; i++ {
		service.SendMessage(ctx, &SendDMInput{SenderID: sender2.ID, ReceiverID: receiver.ID, Content: "From s2", Type: model.MessageTypeText})
	}

	count1, _ := service.CountUnreadFromUser(ctx, receiver.ID, sender1.ID)
	if count1 != 3 {
		t.Errorf("Expected 3 unread from sender1, got %d", count1)
	}

	count2, _ := service.CountUnreadFromUser(ctx, receiver.ID, sender2.ID)
	if count2 != 2 {
		t.Errorf("Expected 2 unread from sender2, got %d", count2)
	}
}

func TestDirectMessageService_GetByID(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	ctx := context.Background()

	sent, _ := service.SendMessage(ctx, &SendDMInput{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Test message",
		Type:       model.MessageTypeText,
	})

	// Sender can get the message
	found, err := service.GetByID(ctx, sent.ID, sender.ID)
	if err != nil {
		t.Fatalf("Failed to get message as sender: %v", err)
	}
	if found.Content != sent.Content {
		t.Errorf("Expected content '%s', got '%s'", sent.Content, found.Content)
	}

	// Receiver can get the message
	found, err = service.GetByID(ctx, sent.ID, receiver.ID)
	if err != nil {
		t.Fatalf("Failed to get message as receiver: %v", err)
	}
}

func TestDirectMessageService_GetByID_NotParticipant(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	otherUser := createUserForDMServiceTest(t, db, "other")
	ctx := context.Background()

	sent, _ := service.SendMessage(ctx, &SendDMInput{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Private message",
		Type:       model.MessageTypeText,
	})

	_, err := service.GetByID(ctx, sent.ID, otherUser.ID)
	if err == nil {
		t.Error("Expected permission denied for non-participant")
	}
}

func TestDirectMessageService_DefaultMessageType(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	ctx := context.Background()

	msg, _ := service.SendMessage(ctx, &SendDMInput{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Test message",
		// Type not specified - should default to text
	})

	if msg.Type != model.MessageTypeText {
		t.Errorf("Expected default type 'text', got '%s'", msg.Type)
	}
}

func TestDirectMessageService_MessageTypes(t *testing.T) {
	service, db := setupTestDMService(t)
	defer db.Close()
	defer cleanupDMServiceTestDB(t, db)

	sender := createUserForDMServiceTest(t, db, "sender")
	receiver := createUserForDMServiceTest(t, db, "receiver")
	ctx := context.Background()

	types := []model.MessageType{
		model.MessageTypeText,
		model.MessageTypeImage,
		model.MessageTypeFile,
	}

	for _, msgType := range types {
		msg, err := service.SendMessage(ctx, &SendDMInput{
			SenderID:   sender.ID,
			ReceiverID: receiver.ID,
			Content:    "Content",
			Type:       msgType,
		})

		if err != nil {
			t.Errorf("Failed to send message of type %s: %v", msgType, err)
		}

		if msg.Type != msgType {
			t.Errorf("Expected type %s, got %s", msgType, msg.Type)
		}
	}
}
