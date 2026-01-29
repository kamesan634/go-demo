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

func setupTestUserService(t *testing.T) (*UserService, *sqlx.DB) {
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
	return service, db
}

func cleanupUserServiceTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE users, blocked_users, friendships CASCADE")
}

func createUserForServiceTest(t *testing.T, db *sqlx.DB, username string) *model.User {
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

func TestUserService_GetByID(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "testuser")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	ctx := context.Background()

	_, err := service.GetByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestUserService_GetProfile(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "testuser")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "testuser")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	createUserForServiceTest(t, db, "alice")
	createUserForServiceTest(t, db, "bob")
	createUserForServiceTest(t, db, "charlie")
	ctx := context.Background()

	results, err := service.Search(ctx, "ali", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search users: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Username != "alice" {
		t.Errorf("Expected 'alice', got '%s'", results[0].Username)
	}
}

func TestUserService_UpdateStatus(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "testuser")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	blocker := createUserForServiceTest(t, db, "blocker")
	blocked := createUserForServiceTest(t, db, "blocked")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	ctx := context.Background()

	err := service.BlockUser(ctx, user.ID, user.ID)
	if err == nil {
		t.Error("Expected error when blocking self")
	}
}

func TestUserService_UnblockUser(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	blocker := createUserForServiceTest(t, db, "blocker")
	blocked := createUserForServiceTest(t, db, "blocked")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user1 := createUserForServiceTest(t, db, "user1")
	user2 := createUserForServiceTest(t, db, "user2")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	blocker := createUserForServiceTest(t, db, "blocker")
	blocked1 := createUserForServiceTest(t, db, "blocked1")
	blocked2 := createUserForServiceTest(t, db, "blocked2")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	friend := createUserForServiceTest(t, db, "friend")
	ctx := context.Background()

	err := service.SendFriendRequest(ctx, user.ID, friend.ID)
	if err != nil {
		t.Fatalf("Failed to send friend request: %v", err)
	}
}

func TestUserService_SendFriendRequest_ToSelf(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	ctx := context.Background()

	err := service.SendFriendRequest(ctx, user.ID, user.ID)
	if err == nil {
		t.Error("Expected error when sending friend request to self")
	}
}

func TestUserService_SendFriendRequest_ToBlocked(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	friend := createUserForServiceTest(t, db, "friend")
	ctx := context.Background()

	service.BlockUser(ctx, friend.ID, user.ID)

	err := service.SendFriendRequest(ctx, user.ID, friend.ID)
	if err == nil {
		t.Error("Expected error when sending friend request to user who blocked you")
	}
}

func TestUserService_AcceptFriendRequest(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	friend := createUserForServiceTest(t, db, "friend")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	friend := createUserForServiceTest(t, db, "friend")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	friend := createUserForServiceTest(t, db, "friend")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	friend1 := createUserForServiceTest(t, db, "friend1")
	friend2 := createUserForServiceTest(t, db, "friend2")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	requester1 := createUserForServiceTest(t, db, "requester1")
	requester2 := createUserForServiceTest(t, db, "requester2")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user := createUserForServiceTest(t, db, "user")
	target1 := createUserForServiceTest(t, db, "target1")
	target2 := createUserForServiceTest(t, db, "target2")
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
	service, db := setupTestUserService(t)
	defer db.Close()
	defer cleanupUserServiceTestDB(t, db)

	user1 := createUserForServiceTest(t, db, "user1")
	createUserForServiceTest(t, db, "user2")
	ctx := context.Background()

	service.UpdateStatus(ctx, user1.ID, model.UserStatusOnline)

	onlineUsers, err := service.GetOnlineUsers(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get online users: %v", err)
	}

	if len(onlineUsers) != 1 {
		t.Errorf("Expected 1 online user, got %d", len(onlineUsers))
	}
}
