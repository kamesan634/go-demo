package repository

import (
	"context"
	"testing"

	"github.com/go-demo/chat/internal/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// 使用有效的 UUID 格式作為不存在的 ID
const dmNonExistentUUID = "00000000-0000-0000-0000-000000000000"

func setupDMTestDBIsolated(t *testing.T) (*sqlx.DB, string) {
	t.Helper()
	return SetupIsolatedTestDB(t)
}

func cleanupDMTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	CleanupTestDataByPrefix(t, db, prefix)
}

func createTestUserForDMIsolated(t *testing.T, db *sqlx.DB, prefix, username string) *model.User {
	t.Helper()
	return CreateIsolatedTestUser(t, db, prefix, username)
}

func TestDirectMessageRepository_Create(t *testing.T) {
	db, prefix := setupDMTestDBIsolated(t)
	defer db.Close()
	defer cleanupDMTestByPrefix(t, db, prefix)

	sender := createTestUserForDMIsolated(t, db, prefix, "dm_sender")
	receiver := createTestUserForDMIsolated(t, db, prefix, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	dm := &model.DirectMessage{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Hello!",
		Type:       model.MessageTypeText,
	}

	err := repo.Create(ctx, dm)
	if err != nil {
		t.Fatalf("Failed to create direct message: %v", err)
	}

	if dm.ID == "" {
		t.Error("Expected message ID to be set")
	}
	if dm.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}
}

func TestDirectMessageRepository_GetByID(t *testing.T) {
	db, prefix := setupDMTestDBIsolated(t)
	defer db.Close()
	defer cleanupDMTestByPrefix(t, db, prefix)

	sender := createTestUserForDMIsolated(t, db, prefix, "dm_sender")
	receiver := createTestUserForDMIsolated(t, db, prefix, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	dm := &model.DirectMessage{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Test message",
		Type:       model.MessageTypeText,
	}
	if err := repo.Create(ctx, dm); err != nil {
		t.Fatalf("Failed to create DM: %v", err)
	}

	found, err := repo.GetByID(ctx, dm.ID)
	if err != nil {
		t.Fatalf("Failed to get direct message: %v", err)
	}

	if found.Content != dm.Content {
		t.Errorf("Expected content '%s', got '%s'", dm.Content, found.Content)
	}

	// Test not found with valid UUID
	_, err = repo.GetByID(ctx, dmNonExistentUUID)
	if err != ErrDirectMessageNotFound {
		t.Errorf("Expected ErrDirectMessageNotFound, got %v", err)
	}
}

func TestDirectMessageRepository_ListConversation(t *testing.T) {
	db, prefix := setupDMTestDBIsolated(t)
	defer db.Close()
	defer cleanupDMTestByPrefix(t, db, prefix)

	user1 := createTestUserForDMIsolated(t, db, prefix, "dm_user1")
	user2 := createTestUserForDMIsolated(t, db, prefix, "dm_user2")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	// 建立雙向對話
	messages := []*model.DirectMessage{
		{SenderID: user1.ID, ReceiverID: user2.ID, Content: "Hi", Type: model.MessageTypeText},
		{SenderID: user2.ID, ReceiverID: user1.ID, Content: "Hello", Type: model.MessageTypeText},
		{SenderID: user1.ID, ReceiverID: user2.ID, Content: "How are you?", Type: model.MessageTypeText},
	}

	for _, dm := range messages {
		if err := repo.Create(ctx, dm); err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
	}

	// 取得對話
	conversation, err := repo.ListConversation(ctx, user1.ID, user2.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list conversation: %v", err)
	}

	if len(conversation) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(conversation))
	}
}

func TestDirectMessageRepository_ListConversations(t *testing.T) {
	db, prefix := setupDMTestDBIsolated(t)
	defer db.Close()
	defer cleanupDMTestByPrefix(t, db, prefix)

	user := createTestUserForDMIsolated(t, db, prefix, "dm_main_user")
	contact1 := createTestUserForDMIsolated(t, db, prefix, "dm_contact1")
	contact2 := createTestUserForDMIsolated(t, db, prefix, "dm_contact2")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	// 與 contact1 的對話
	if err := repo.Create(ctx, &model.DirectMessage{
		SenderID: contact1.ID, ReceiverID: user.ID, Content: "Hi from contact1", Type: model.MessageTypeText,
	}); err != nil {
		t.Fatalf("Failed to create DM: %v", err)
	}

	// 與 contact2 的對話
	if err := repo.Create(ctx, &model.DirectMessage{
		SenderID: contact2.ID, ReceiverID: user.ID, Content: "Hi from contact2", Type: model.MessageTypeText,
	}); err != nil {
		t.Fatalf("Failed to create DM: %v", err)
	}

	conversations, err := repo.ListConversations(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(conversations))
	}
}

func TestDirectMessageRepository_MarkAsRead(t *testing.T) {
	db, prefix := setupDMTestDBIsolated(t)
	defer db.Close()
	defer cleanupDMTestByPrefix(t, db, prefix)

	sender := createTestUserForDMIsolated(t, db, prefix, "dm_sender")
	receiver := createTestUserForDMIsolated(t, db, prefix, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	dm := &model.DirectMessage{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Unread message",
		Type:       model.MessageTypeText,
	}
	if err := repo.Create(ctx, dm); err != nil {
		t.Fatalf("Failed to create DM: %v", err)
	}

	// 確認初始為未讀
	if dm.IsRead {
		t.Error("Expected message to be unread initially")
	}

	// 標記為已讀 (MarkAsRead 參數是 senderID, receiverID)
	err := repo.MarkAsRead(ctx, sender.ID, receiver.ID)
	if err != nil {
		t.Fatalf("Failed to mark as read: %v", err)
	}

	// 驗證已讀
	found, err := repo.GetByID(ctx, dm.ID)
	if err != nil {
		t.Fatalf("Failed to get DM: %v", err)
	}
	if !found.IsRead {
		t.Error("Expected message to be read after marking")
	}
}

func TestDirectMessageRepository_CountUnread(t *testing.T) {
	db, prefix := setupDMTestDBIsolated(t)
	defer db.Close()
	defer cleanupDMTestByPrefix(t, db, prefix)

	sender := createTestUserForDMIsolated(t, db, prefix, "dm_sender")
	receiver := createTestUserForDMIsolated(t, db, prefix, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	// 建立 3 則未讀訊息
	for i := 0; i < 3; i++ {
		if err := repo.Create(ctx, &model.DirectMessage{
			SenderID:   sender.ID,
			ReceiverID: receiver.ID,
			Content:    "Unread",
			Type:       model.MessageTypeText,
		}); err != nil {
			t.Fatalf("Failed to create DM: %v", err)
		}
	}

	count, err := repo.CountUnread(ctx, receiver.ID)
	if err != nil {
		t.Fatalf("Failed to count unread: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 unread messages, got %d", count)
	}
}

func TestDirectMessageRepository_DeleteForUser(t *testing.T) {
	db, prefix := setupDMTestDBIsolated(t)
	defer db.Close()
	defer cleanupDMTestByPrefix(t, db, prefix)

	sender := createTestUserForDMIsolated(t, db, prefix, "dm_sender")
	receiver := createTestUserForDMIsolated(t, db, prefix, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	dm := &model.DirectMessage{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "To delete",
		Type:       model.MessageTypeText,
	}
	if err := repo.Create(ctx, dm); err != nil {
		t.Fatalf("Failed to create DM: %v", err)
	}

	// 接收者刪除訊息
	err := repo.DeleteForUser(ctx, dm.ID, receiver.ID)
	if err != nil {
		t.Fatalf("Failed to delete for user: %v", err)
	}

	// 訊息本身仍存在（只是對該用戶隱藏）
	found, err := repo.GetByID(ctx, dm.ID)
	if err != nil {
		t.Fatalf("Message should still exist: %v", err)
	}
	if found.ID != dm.ID {
		t.Error("Expected message to still exist after delete for user")
	}
}

func TestDirectMessageRepository_CountUnreadFromUser(t *testing.T) {
	db, prefix := setupDMTestDBIsolated(t)
	defer db.Close()
	defer cleanupDMTestByPrefix(t, db, prefix)

	sender := createTestUserForDMIsolated(t, db, prefix, "dm_sender")
	receiver := createTestUserForDMIsolated(t, db, prefix, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	// 建立 2 則未讀訊息
	for i := 0; i < 2; i++ {
		if err := repo.Create(ctx, &model.DirectMessage{
			SenderID:   sender.ID,
			ReceiverID: receiver.ID,
			Content:    "Unread",
			Type:       model.MessageTypeText,
		}); err != nil {
			t.Fatalf("Failed to create DM: %v", err)
		}
	}

	count, err := repo.CountUnreadFromUser(ctx, receiver.ID, sender.ID)
	if err != nil {
		t.Fatalf("Failed to count unread from user: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 unread messages from sender, got %d", count)
	}
}
