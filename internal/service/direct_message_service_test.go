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

func setupTestDMServiceIsolated(t *testing.T) (*DirectMessageService, *sqlx.DB, string) {
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
	prefix := repository.GenerateUniquePrefix()
	return service, db, prefix
}

func cleanupDMServiceTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	repository.CleanupTestDataByPrefix(t, db, prefix)
}

func createUserForDMServiceTestIsolated(t *testing.T, db *sqlx.DB, prefix, username string) *model.User {
	t.Helper()
	return repository.CreateIsolatedTestUser(t, db, prefix, username)
}

func TestDirectMessageService_SendMessage(t *testing.T) {
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	user := createUserForDMServiceTestIsolated(t, db, prefix, "user")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	ctx := context.Background()

	_, err := service.SendMessage(ctx, &SendDMInput{
		SenderID:   sender.ID,
		ReceiverID: "00000000-0000-0000-0000-000000000000",
		Content:    "Hello",
		Type:       model.MessageTypeText,
	})

	if err == nil {
		t.Error("Expected error when sending to non-existent user")
	}
}

func TestDirectMessageService_SendMessage_Blocked(t *testing.T) {
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	user1 := createUserForDMServiceTestIsolated(t, db, prefix, "user1")
	user2 := createUserForDMServiceTestIsolated(t, db, prefix, "user2")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	user := createUserForDMServiceTestIsolated(t, db, prefix, "user")
	ctx := context.Background()

	_, err := service.GetConversation(ctx, user.ID, "00000000-0000-0000-0000-000000000000", 10, 0)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestDirectMessageService_ListConversations(t *testing.T) {
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	user := createUserForDMServiceTestIsolated(t, db, prefix, "user")
	contact1 := createUserForDMServiceTestIsolated(t, db, prefix, "contact1")
	contact2 := createUserForDMServiceTestIsolated(t, db, prefix, "contact2")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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

	if count < 3 {
		t.Errorf("Expected at least 3 unread messages, got %d", count)
	}
}

func TestDirectMessageService_CountUnreadFromUser(t *testing.T) {
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender1 := createUserForDMServiceTestIsolated(t, db, prefix, "sender1")
	sender2 := createUserForDMServiceTestIsolated(t, db, prefix, "sender2")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
	otherUser := createUserForDMServiceTestIsolated(t, db, prefix, "other")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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
	service, db, prefix := setupTestDMServiceIsolated(t)
	defer db.Close()
	defer cleanupDMServiceTestByPrefix(t, db, prefix)

	sender := createUserForDMServiceTestIsolated(t, db, prefix, "sender")
	receiver := createUserForDMServiceTestIsolated(t, db, prefix, "receiver")
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
