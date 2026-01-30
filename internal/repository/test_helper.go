package repository

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-demo/chat/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// 全域計數器確保唯一性
var testCounter int64

// GenerateUniquePrefix 生成唯一的測試前綴
// 使用 UUID 確保並行測試不會衝突
func GenerateUniquePrefix() string {
	count := atomic.AddInt64(&testCounter, 1)
	return uuid.New().String()[:8] + "_" + time.Now().Format("150405") + "_" + string(rune(count%26+'a'))
}

// SetupIsolatedTestDB 建立隔離的測試資料庫連線
// 每個測試使用唯一前綴，避免並行測試衝突
func SetupIsolatedTestDB(t *testing.T) (*sqlx.DB, string) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	// 生成此測試的唯一前綴
	prefix := GenerateUniquePrefix()

	return db, prefix
}

// CleanupTestDataByPrefix 清理特定前綴的測試資料
// 只清理本測試建立的資料，不影響其他測試
func CleanupTestDataByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()

	ctx := context.Background()

	// 按照外鍵依賴順序刪除
	_, _ = db.ExecContext(ctx, "DELETE FROM message_attachments WHERE message_id IN (SELECT id FROM messages WHERE content LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM notifications WHERE user_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM direct_messages WHERE sender_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM direct_messages WHERE receiver_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM messages WHERE user_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM room_members WHERE user_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM blocked_users WHERE blocker_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM blocked_users WHERE blocked_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM friendships WHERE user_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM friendships WHERE friend_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM rooms WHERE owner_id IN (SELECT id FROM users WHERE username LIKE $1)", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM rooms WHERE name LIKE $1", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE username LIKE $1", prefix+"%")
	_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE email LIKE $1", prefix+"%")
}

// CreateIsolatedTestUser 建立隔離的測試用戶
func CreateIsolatedTestUser(t *testing.T, db *sqlx.DB, prefix, name string) *model.User {
	t.Helper()

	userRepo := NewUserRepository(db)
	username := prefix + "_" + name
	user := &model.User{
		Username:     username,
		Email:        username + "@test.example.com",
		PasswordHash: "hashedpassword",
		Status:       model.UserStatusOffline,
	}

	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

// CreateIsolatedTestRoom 建立隔離的測試聊天室
func CreateIsolatedTestRoom(t *testing.T, db *sqlx.DB, prefix string, owner *model.User) *model.Room {
	t.Helper()

	roomRepo := NewRoomRepository(db)
	room := &model.Room{
		Name:       prefix + "_room",
		Type:       model.RoomTypePublic,
		OwnerID:    owner.ID,
		MaxMembers: 100,
	}

	if err := roomRepo.Create(context.Background(), room); err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}

	return room
}
