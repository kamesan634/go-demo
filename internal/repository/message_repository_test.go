package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-demo/chat/internal/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// 使用有效的 UUID 格式作為不存在的 ID
const msgNonExistentUUID = "00000000-0000-0000-0000-000000000000"

func setupMessageTestDBIsolated(t *testing.T) (*sqlx.DB, string) {
	t.Helper()
	return SetupIsolatedTestDB(t)
}

func cleanupMessageTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	CleanupTestDataByPrefix(t, db, prefix)
}

func createTestUserForMessageIsolated(t *testing.T, db *sqlx.DB, prefix, username string) *model.User {
	t.Helper()
	return CreateIsolatedTestUser(t, db, prefix, username)
}

func createTestRoomIsolated(t *testing.T, db *sqlx.DB, prefix string, owner *model.User) *model.Room {
	t.Helper()
	return CreateIsolatedTestRoom(t, db, prefix, owner)
}

func TestMessageRepository_Create(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	msg := &model.Message{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Hello, World!",
		Type:    model.MessageTypeText,
	}

	err := repo.Create(ctx, msg)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	if msg.ID == "" {
		t.Error("Expected message ID to be set")
	}
	if msg.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}
}

func TestMessageRepository_GetByID(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	msg := &model.Message{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Test message",
		Type:    model.MessageTypeText,
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	found, err := repo.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Failed to get message: %v", err)
	}

	if found.Content != msg.Content {
		t.Errorf("Expected content '%s', got '%s'", msg.Content, found.Content)
	}

	// Test not found
	_, err = repo.GetByID(ctx, msgNonExistentUUID)
	if err != ErrMessageNotFound {
		t.Errorf("Expected ErrMessageNotFound, got %v", err)
	}
}

func TestMessageRepository_GetByIDWithUser(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	msg := &model.Message{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Test message",
		Type:    model.MessageTypeText,
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	found, err := repo.GetByIDWithUser(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Failed to get message with user: %v", err)
	}

	if found.Username != user.Username {
		t.Errorf("Expected username '%s', got '%s'", user.Username, found.Username)
	}
}

func TestMessageRepository_Update(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	msg := &model.Message{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Original message",
		Type:    model.MessageTypeText,
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	err := repo.Update(ctx, msg.ID, "Updated message")
	if err != nil {
		t.Fatalf("Failed to update message: %v", err)
	}

	found, _ := repo.GetByID(ctx, msg.ID)
	if found.Content != "Updated message" {
		t.Errorf("Expected content 'Updated message', got '%s'", found.Content)
	}
	if !found.IsEdited {
		t.Error("Expected is_edited to be true")
	}
}

func TestMessageRepository_SoftDelete(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	msg := &model.Message{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "To be deleted",
		Type:    model.MessageTypeText,
	}
	if err := repo.Create(ctx, msg); err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	err := repo.SoftDelete(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Failed to soft delete message: %v", err)
	}

	found, _ := repo.GetByID(ctx, msg.ID)
	if !found.IsDeleted {
		t.Error("Expected is_deleted to be true")
	}
}

func TestMessageRepository_ListByRoomID(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	// Create multiple messages
	for i := 0; i < 5; i++ {
		msg := &model.Message{
			RoomID:  room.ID,
			UserID:  user.ID,
			Content: "Message " + string(rune('A'+i)),
			Type:    model.MessageTypeText,
		}
		if err := repo.Create(ctx, msg); err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
	}

	messages, err := repo.ListByRoomID(ctx, room.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	if len(messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(messages))
	}
}

func TestMessageRepository_ListByRoomID_Pagination(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	// Create 10 messages
	for i := 0; i < 10; i++ {
		msg := &model.Message{
			RoomID:  room.ID,
			UserID:  user.ID,
			Content: "Message",
			Type:    model.MessageTypeText,
		}
		if err := repo.Create(ctx, msg); err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
	}

	// Get first page (5 messages)
	page1, _ := repo.ListByRoomID(ctx, room.ID, 5, 0)
	if len(page1) != 5 {
		t.Errorf("Expected 5 messages on page 1, got %d", len(page1))
	}

	// Get second page (5 messages)
	page2, _ := repo.ListByRoomID(ctx, room.ID, 5, 5)
	if len(page2) != 5 {
		t.Errorf("Expected 5 messages on page 2, got %d", len(page2))
	}
}

func TestMessageRepository_Search(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	messages := []string{"Hello World", "Golang is great", "Testing is fun"}
	for _, content := range messages {
		msg := &model.Message{
			RoomID:  room.ID,
			UserID:  user.ID,
			Content: content,
			Type:    model.MessageTypeText,
		}
		if err := repo.Create(ctx, msg); err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
	}

	results, err := repo.Search(ctx, room.ID, "Golang", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search messages: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Content != "Golang is great" {
		t.Errorf("Expected 'Golang is great', got '%s'", results[0].Content)
	}
}

func TestMessageRepository_CountByRoomID(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	// Create 3 messages
	for i := 0; i < 3; i++ {
		msg := &model.Message{
			RoomID:  room.ID,
			UserID:  user.ID,
			Content: "Message",
			Type:    model.MessageTypeText,
		}
		if err := repo.Create(ctx, msg); err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
	}

	count, err := repo.CountByRoomID(ctx, room.ID)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 messages, got %d", count)
	}
}

func TestMessageRepository_MessageWithReply(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	// Create original message
	original := &model.Message{
		RoomID:  room.ID,
		UserID:  user.ID,
		Content: "Original message",
		Type:    model.MessageTypeText,
	}
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Create reply
	reply := &model.Message{
		RoomID:    room.ID,
		UserID:    user.ID,
		Content:   "This is a reply",
		Type:      model.MessageTypeText,
		ReplyToID: sql.NullString{String: original.ID, Valid: true},
	}
	if err := repo.Create(ctx, reply); err != nil {
		t.Fatalf("Failed to create reply: %v", err)
	}

	found, _ := repo.GetByID(ctx, reply.ID)
	if found.GetReplyToID() != original.ID {
		t.Errorf("Expected reply_to_id '%s', got '%s'", original.ID, found.GetReplyToID())
	}
}

func TestMessageRepository_MessageTypes(t *testing.T) {
	db, prefix := setupMessageTestDBIsolated(t)
	defer db.Close()
	defer cleanupMessageTestByPrefix(t, db, prefix)

	user := createTestUserForMessageIsolated(t, db, prefix, "sender")
	room := createTestRoomIsolated(t, db, prefix, user)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	types := []model.MessageType{
		model.MessageTypeText,
		model.MessageTypeImage,
		model.MessageTypeFile,
		model.MessageTypeSystem,
	}

	for _, msgType := range types {
		msg := &model.Message{
			RoomID:  room.ID,
			UserID:  user.ID,
			Content: "Message of type " + string(msgType),
			Type:    msgType,
		}
		err := repo.Create(ctx, msg)
		if err != nil {
			t.Errorf("Failed to create message of type %s: %v", msgType, err)
		}

		found, _ := repo.GetByID(ctx, msg.ID)
		if found.Type != msgType {
			t.Errorf("Expected type %s, got %s", msgType, found.Type)
		}
	}
}
