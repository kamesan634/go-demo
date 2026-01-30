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

func setupTestUserServiceIsolated(t *testing.T) (*UserService, *sqlx.DB, string) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	blockedRepo := repository.NewBlockedUserRepository(db)
	friendshipRepo := repository.NewFriendshipRepository(db)
	logger := zap.NewNop()

	service := NewUserService(userRepo, blockedRepo, friendshipRepo, logger)
	prefix := repository.GenerateUniquePrefix()
	return service, db, prefix
}

func cleanupUserServiceTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	repository.CleanupTestDataByPrefix(t, db, prefix)
}

func createUserForServiceTestIsolated(t *testing.T, db *sqlx.DB, prefix, username string) *model.User {
	t.Helper()
	return repository.CreateIsolatedTestUser(t, db, prefix, username)
}

func TestUserService_GetByID(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "testuser")
	ctx := context.Background()

	found, err := service.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if found.Username != user.Username {
		t.Errorf("Expected username '%s', got '%s'", user.Username, found.Username)
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	ctx := context.Background()

	_, err := service.GetByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestUserService_GetProfile(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "testuser")
	ctx := context.Background()

	profile, err := service.GetProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}

	if profile.Username != user.Username {
		t.Errorf("Expected username '%s', got '%s'", user.Username, profile.Username)
	}
}

func TestUserService_UpdateProfile(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "testuser")
	ctx := context.Background()

	displayName := "Test User"
	bio := "Hello World"

	updated, err := service.UpdateProfile(ctx, &UpdateProfileInput{
		UserID:      user.ID,
		DisplayName: &displayName,
		Bio:         &bio,
	})
	if err != nil {
		t.Fatalf("Failed to update profile: %v", err)
	}

	if updated.DisplayName.String != displayName {
		t.Errorf("Expected display name '%s', got '%s'", displayName, updated.DisplayName.String)
	}
	if updated.Bio.String != bio {
		t.Errorf("Expected bio '%s', got '%s'", bio, updated.Bio.String)
	}
}

func TestUserService_Search(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	alice := createUserForServiceTestIsolated(t, db, prefix, "alice")
	createUserForServiceTestIsolated(t, db, prefix, "bob")
	createUserForServiceTestIsolated(t, db, prefix, "charlie")
	ctx := context.Background()

	// Search using the full prefixed username to find alice
	results, err := service.Search(ctx, prefix+"_alice", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search users: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Username != alice.Username {
		t.Errorf("Expected '%s', got '%s'", alice.Username, results[0].Username)
	}
}

func TestUserService_UpdateStatus(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "testuser")
	ctx := context.Background()

	err := service.UpdateStatus(ctx, user.ID, model.UserStatusOnline)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	found, _ := service.GetByID(ctx, user.ID)
	if found.Status != model.UserStatusOnline {
		t.Errorf("Expected status 'online', got '%s'", found.Status)
	}
}

func TestUserService_BlockUser(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	blocker := createUserForServiceTestIsolated(t, db, prefix, "blocker")
	blocked := createUserForServiceTestIsolated(t, db, prefix, "blocked")
	ctx := context.Background()

	err := service.BlockUser(ctx, blocker.ID, blocked.ID)
	if err != nil {
		t.Fatalf("Failed to block user: %v", err)
	}

	isBlocked, _ := service.IsBlocked(ctx, blocker.ID, blocked.ID)
	if !isBlocked {
		t.Error("Expected user to be blocked")
	}
}

func TestUserService_BlockUser_Self(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	ctx := context.Background()

	err := service.BlockUser(ctx, user.ID, user.ID)
	if err == nil {
		t.Error("Expected error when blocking self")
	}
}

func TestUserService_UnblockUser(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	blocker := createUserForServiceTestIsolated(t, db, prefix, "blocker")
	blocked := createUserForServiceTestIsolated(t, db, prefix, "blocked")
	ctx := context.Background()

	service.BlockUser(ctx, blocker.ID, blocked.ID)

	err := service.UnblockUser(ctx, blocker.ID, blocked.ID)
	if err != nil {
		t.Fatalf("Failed to unblock user: %v", err)
	}

	isBlocked, _ := service.IsBlocked(ctx, blocker.ID, blocked.ID)
	if isBlocked {
		t.Error("Expected user to be unblocked")
	}
}

func TestUserService_IsBlockedEither(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user1 := createUserForServiceTestIsolated(t, db, prefix, "user1")
	user2 := createUserForServiceTestIsolated(t, db, prefix, "user2")
	ctx := context.Background()

	service.BlockUser(ctx, user1.ID, user2.ID)

	// Both directions should return true
	isBlocked, _ := service.IsBlockedEither(ctx, user1.ID, user2.ID)
	if !isBlocked {
		t.Error("Expected IsBlockedEither to return true")
	}

	isBlocked, _ = service.IsBlockedEither(ctx, user2.ID, user1.ID)
	if !isBlocked {
		t.Error("Expected IsBlockedEither to return true (reverse)")
	}
}

func TestUserService_ListBlockedUsers(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	blocker := createUserForServiceTestIsolated(t, db, prefix, "blocker")
	blocked1 := createUserForServiceTestIsolated(t, db, prefix, "blocked1")
	blocked2 := createUserForServiceTestIsolated(t, db, prefix, "blocked2")
	ctx := context.Background()

	service.BlockUser(ctx, blocker.ID, blocked1.ID)
	service.BlockUser(ctx, blocker.ID, blocked2.ID)

	blockedUsers, err := service.ListBlockedUsers(ctx, blocker.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list blocked users: %v", err)
	}

	if len(blockedUsers) != 2 {
		t.Errorf("Expected 2 blocked users, got %d", len(blockedUsers))
	}
}

func TestUserService_SendFriendRequest(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	friend := createUserForServiceTestIsolated(t, db, prefix, "friend")
	ctx := context.Background()

	err := service.SendFriendRequest(ctx, user.ID, friend.ID)
	if err != nil {
		t.Fatalf("Failed to send friend request: %v", err)
	}
}

func TestUserService_SendFriendRequest_ToSelf(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	ctx := context.Background()

	err := service.SendFriendRequest(ctx, user.ID, user.ID)
	if err == nil {
		t.Error("Expected error when sending friend request to self")
	}
}

func TestUserService_SendFriendRequest_ToBlocked(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	friend := createUserForServiceTestIsolated(t, db, prefix, "friend")
	ctx := context.Background()

	service.BlockUser(ctx, friend.ID, user.ID)

	err := service.SendFriendRequest(ctx, user.ID, friend.ID)
	if err == nil {
		t.Error("Expected error when sending friend request to user who blocked you")
	}
}

func TestUserService_AcceptFriendRequest(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	friend := createUserForServiceTestIsolated(t, db, prefix, "friend")
	ctx := context.Background()

	service.SendFriendRequest(ctx, user.ID, friend.ID)

	err := service.AcceptFriendRequest(ctx, friend.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to accept friend request: %v", err)
	}

	areFriends, _ := service.AreFriends(ctx, user.ID, friend.ID)
	if !areFriends {
		t.Error("Expected users to be friends")
	}
}

func TestUserService_RejectFriendRequest(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	friend := createUserForServiceTestIsolated(t, db, prefix, "friend")
	ctx := context.Background()

	service.SendFriendRequest(ctx, user.ID, friend.ID)

	err := service.RejectFriendRequest(ctx, friend.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to reject friend request: %v", err)
	}

	areFriends, _ := service.AreFriends(ctx, user.ID, friend.ID)
	if areFriends {
		t.Error("Expected users not to be friends")
	}
}

func TestUserService_RemoveFriend(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	friend := createUserForServiceTestIsolated(t, db, prefix, "friend")
	ctx := context.Background()

	service.SendFriendRequest(ctx, user.ID, friend.ID)
	service.AcceptFriendRequest(ctx, friend.ID, user.ID)

	err := service.RemoveFriend(ctx, user.ID, friend.ID)
	if err != nil {
		t.Fatalf("Failed to remove friend: %v", err)
	}

	areFriends, _ := service.AreFriends(ctx, user.ID, friend.ID)
	if areFriends {
		t.Error("Expected users not to be friends")
	}
}

func TestUserService_ListFriends(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	friend1 := createUserForServiceTestIsolated(t, db, prefix, "friend1")
	friend2 := createUserForServiceTestIsolated(t, db, prefix, "friend2")
	ctx := context.Background()

	service.SendFriendRequest(ctx, user.ID, friend1.ID)
	service.AcceptFriendRequest(ctx, friend1.ID, user.ID)
	service.SendFriendRequest(ctx, user.ID, friend2.ID)
	service.AcceptFriendRequest(ctx, friend2.ID, user.ID)

	friends, err := service.ListFriends(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list friends: %v", err)
	}

	if len(friends) != 2 {
		t.Errorf("Expected 2 friends, got %d", len(friends))
	}
}

func TestUserService_ListPendingRequests(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	requester1 := createUserForServiceTestIsolated(t, db, prefix, "requester1")
	requester2 := createUserForServiceTestIsolated(t, db, prefix, "requester2")
	ctx := context.Background()

	service.SendFriendRequest(ctx, requester1.ID, user.ID)
	service.SendFriendRequest(ctx, requester2.ID, user.ID)

	pending, err := service.ListPendingRequests(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list pending requests: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending requests, got %d", len(pending))
	}
}

func TestUserService_ListSentRequests(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user := createUserForServiceTestIsolated(t, db, prefix, "user")
	target1 := createUserForServiceTestIsolated(t, db, prefix, "target1")
	target2 := createUserForServiceTestIsolated(t, db, prefix, "target2")
	ctx := context.Background()

	service.SendFriendRequest(ctx, user.ID, target1.ID)
	service.SendFriendRequest(ctx, user.ID, target2.ID)

	sent, err := service.ListSentRequests(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list sent requests: %v", err)
	}

	if len(sent) != 2 {
		t.Errorf("Expected 2 sent requests, got %d", len(sent))
	}
}

func TestUserService_GetOnlineUsers(t *testing.T) {
	service, db, prefix := setupTestUserServiceIsolated(t)
	defer db.Close()
	defer cleanupUserServiceTestByPrefix(t, db, prefix)

	user1 := createUserForServiceTestIsolated(t, db, prefix, "user1")
	createUserForServiceTestIsolated(t, db, prefix, "user2")
	ctx := context.Background()

	service.UpdateStatus(ctx, user1.ID, model.UserStatusOnline)

	onlineUsers, err := service.GetOnlineUsers(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get online users: %v", err)
	}

	// At least 1 online user (the one we just set)
	found := false
	for _, u := range onlineUsers {
		if u.ID == user1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find user1 in online users")
	}
}
