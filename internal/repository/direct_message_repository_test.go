package repository

import (
	"context"
	"testing"
	"time"

	"github.com/go-demo/chat/internal/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// 使用有效的 UUID 格式作為不存在的 ID
const dmNonExistentUUID = "00000000-0000-0000-0000-000000000000"

func setupDMTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	return db
}

func cleanupDMTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE direct_messages, users CASCADE")
}

func createTestUserForDM(t *testing.T, db *sqlx.DB, username string) *model.User {
	t.Helper()
	userRepo := NewUserRepository(db)
	// 添加時間戳確保唯一性
	uniqueUsername := username + "_" + time.Now().Format("150405.000000")
	user := &model.User{
		Username:     uniqueUsername,
		Email:        uniqueUsername + "@example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

func TestDirectMessageRepository_Create(t *testing.T) {
	db := setupDMTestDB(t)
	defer db.Close()
	cleanupDMTestDB(t, db) // 測試前清理
	defer cleanupDMTestDB(t, db)

	sender := createTestUserForDM(t, db, "dm_sender")
	receiver := createTestUserForDM(t, db, "dm_receiver")
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
	db := setupDMTestDB(t)
	defer db.Close()
	cleanupDMTestDB(t, db)
	defer cleanupDMTestDB(t, db)

	sender := createTestUserForDM(t, db, "dm_sender")
	receiver := createTestUserForDM(t, db, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	dm := &model.DirectMessage{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Test message",
		Type:       model.MessageTypeText,
	}
	repo.Create(ctx, dm)

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
	db := setupDMTestDB(t)
	defer db.Close()
	cleanupDMTestDB(t, db)
	defer cleanupDMTestDB(t, db)

	user1 := createTestUserForDM(t, db, "dm_user1")
	user2 := createTestUserForDM(t, db, "dm_user2")
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
	db := setupDMTestDB(t)
	defer db.Close()
	cleanupDMTestDB(t, db)
	defer cleanupDMTestDB(t, db)

	user := createTestUserForDM(t, db, "dm_main_user")
	contact1 := createTestUserForDM(t, db, "dm_contact1")
	contact2 := createTestUserForDM(t, db, "dm_contact2")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	// 與 contact1 的對話
	repo.Create(ctx, &model.DirectMessage{
		SenderID: contact1.ID, ReceiverID: user.ID, Content: "Hi from contact1", Type: model.MessageTypeText,
	})

	// 與 contact2 的對話
	repo.Create(ctx, &model.DirectMessage{
		SenderID: contact2.ID, ReceiverID: user.ID, Content: "Hi from contact2", Type: model.MessageTypeText,
	})

	conversations, err := repo.ListConversations(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list conversations: %v", err)
	}

	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(conversations))
	}
}

func TestDirectMessageRepository_MarkAsRead(t *testing.T) {
	db := setupDMTestDB(t)
	defer db.Close()
	cleanupDMTestDB(t, db)
	defer cleanupDMTestDB(t, db)

	sender := createTestUserForDM(t, db, "dm_sender")
	receiver := createTestUserForDM(t, db, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	dm := &model.DirectMessage{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "Unread message",
		Type:       model.MessageTypeText,
	}
	repo.Create(ctx, dm)

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
	found, _ := repo.GetByID(ctx, dm.ID)
	if !found.IsRead {
		t.Error("Expected message to be read after marking")
	}
}

func TestDirectMessageRepository_CountUnread(t *testing.T) {
	db := setupDMTestDB(t)
	defer db.Close()
	cleanupDMTestDB(t, db)
	defer cleanupDMTestDB(t, db)

	sender := createTestUserForDM(t, db, "dm_sender")
	receiver := createTestUserForDM(t, db, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	// 建立 3 則未讀訊息
	for i := 0; i < 3; i++ {
		repo.Create(ctx, &model.DirectMessage{
			SenderID:   sender.ID,
			ReceiverID: receiver.ID,
			Content:    "Unread",
			Type:       model.MessageTypeText,
		})
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
	db := setupDMTestDB(t)
	defer db.Close()
	cleanupDMTestDB(t, db)
	defer cleanupDMTestDB(t, db)

	sender := createTestUserForDM(t, db, "dm_sender")
	receiver := createTestUserForDM(t, db, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	dm := &model.DirectMessage{
		SenderID:   sender.ID,
		ReceiverID: receiver.ID,
		Content:    "To delete",
		Type:       model.MessageTypeText,
	}
	repo.Create(ctx, dm)

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
	db := setupDMTestDB(t)
	defer db.Close()
	cleanupDMTestDB(t, db)
	defer cleanupDMTestDB(t, db)

	sender := createTestUserForDM(t, db, "dm_sender")
	receiver := createTestUserForDM(t, db, "dm_receiver")
	repo := NewDirectMessageRepository(db)
	ctx := context.Background()

	// 建立 2 則未讀訊息
	for i := 0; i < 2; i++ {
		repo.Create(ctx, &model.DirectMessage{
			SenderID:   sender.ID,
			ReceiverID: receiver.ID,
			Content:    "Unread",
			Type:       model.MessageTypeText,
		})
	}

	count, err := repo.CountUnreadFromUser(ctx, receiver.ID, sender.ID)
	if err != nil {
		t.Fatalf("Failed to count unread from user: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 unread messages from sender, got %d", count)
	}
}
