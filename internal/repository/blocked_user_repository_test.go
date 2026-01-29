package repository

import (
	"context"
	"testing"

	"github.com/go-demo/chat/internal/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func setupBlockedTestDBIsolated(t *testing.T) (*sqlx.DB, string) {
	t.Helper()
	return SetupIsolatedTestDB(t)
}

func cleanupBlockedTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	CleanupTestDataByPrefix(t, db, prefix)
}

func createTestUserForBlockedIsolated(t *testing.T, db *sqlx.DB, prefix, name string) *model.User {
	t.Helper()
	return CreateIsolatedTestUser(t, db, prefix, name)
}

// ==================== BlockedUserRepository Tests ====================

func TestBlockedUserRepository_Block(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	blocker := createTestUserForBlockedIsolated(t, db, prefix, "blocker")
	blocked := createTestUserForBlockedIsolated(t, db, prefix, "blocked")
	repo := NewBlockedUserRepository(db)
	ctx := context.Background()

	err := repo.Block(ctx, blocker.ID, blocked.ID)
	if err != nil {
		t.Fatalf("Failed to block user: %v", err)
	}

	// Verify blocked
	isBlocked, _ := repo.IsBlocked(ctx, blocker.ID, blocked.ID)
	if !isBlocked {
		t.Error("Expected user to be blocked")
	}
}

func TestBlockedUserRepository_Block_Self(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	repo := NewBlockedUserRepository(db)
	ctx := context.Background()

	err := repo.Block(ctx, user.ID, user.ID)
	if err != ErrCannotBlockSelf {
		t.Errorf("Expected ErrCannotBlockSelf, got %v", err)
	}
}

func TestBlockedUserRepository_Block_AlreadyBlocked(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	blocker := createTestUserForBlockedIsolated(t, db, prefix, "blocker")
	blocked := createTestUserForBlockedIsolated(t, db, prefix, "blocked")
	repo := NewBlockedUserRepository(db)
	ctx := context.Background()

	err := repo.Block(ctx, blocker.ID, blocked.ID)
	if err != nil {
		t.Fatalf("Failed to first block: %v", err)
	}

	// Try to block again
	err = repo.Block(ctx, blocker.ID, blocked.ID)
	if err != ErrAlreadyBlocked {
		t.Errorf("Expected ErrAlreadyBlocked, got %v", err)
	}
}

func TestBlockedUserRepository_Unblock(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	blocker := createTestUserForBlockedIsolated(t, db, prefix, "blocker")
	blocked := createTestUserForBlockedIsolated(t, db, prefix, "blocked")
	repo := NewBlockedUserRepository(db)
	ctx := context.Background()

	err := repo.Block(ctx, blocker.ID, blocked.ID)
	if err != nil {
		t.Fatalf("Failed to block: %v", err)
	}

	err = repo.Unblock(ctx, blocker.ID, blocked.ID)
	if err != nil {
		t.Fatalf("Failed to unblock user: %v", err)
	}

	// Verify unblocked
	isBlocked, _ := repo.IsBlocked(ctx, blocker.ID, blocked.ID)
	if isBlocked {
		t.Error("Expected user to be unblocked")
	}
}

func TestBlockedUserRepository_Unblock_NotBlocked(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user1 := createTestUserForBlockedIsolated(t, db, prefix, "user1")
	user2 := createTestUserForBlockedIsolated(t, db, prefix, "user2")
	repo := NewBlockedUserRepository(db)
	ctx := context.Background()

	err := repo.Unblock(ctx, user1.ID, user2.ID)
	if err != ErrBlockNotFound {
		t.Errorf("Expected ErrBlockNotFound, got %v", err)
	}
}

func TestBlockedUserRepository_IsBlocked(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user1 := createTestUserForBlockedIsolated(t, db, prefix, "user1")
	user2 := createTestUserForBlockedIsolated(t, db, prefix, "user2")
	repo := NewBlockedUserRepository(db)
	ctx := context.Background()

	// Initially not blocked
	isBlocked, err := repo.IsBlocked(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("Failed to check if blocked: %v", err)
	}
	if isBlocked {
		t.Error("Expected not to be blocked initially")
	}

	// Block and check
	err = repo.Block(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("Failed to block: %v", err)
	}

	isBlocked, _ = repo.IsBlocked(ctx, user1.ID, user2.ID)
	if !isBlocked {
		t.Error("Expected to be blocked")
	}

	// Reverse direction should not be blocked
	isBlocked, _ = repo.IsBlocked(ctx, user2.ID, user1.ID)
	if isBlocked {
		t.Error("Expected reverse direction not to be blocked")
	}
}

func TestBlockedUserRepository_IsBlockedEither(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user1 := createTestUserForBlockedIsolated(t, db, prefix, "user1")
	user2 := createTestUserForBlockedIsolated(t, db, prefix, "user2")
	repo := NewBlockedUserRepository(db)
	ctx := context.Background()

	// Initially not blocked
	isBlocked, err := repo.IsBlockedEither(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("Failed to check if blocked either: %v", err)
	}
	if isBlocked {
		t.Error("Expected not to be blocked initially")
	}

	// Block in one direction
	err = repo.Block(ctx, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("Failed to block: %v", err)
	}

	// Check both directions
	isBlocked, _ = repo.IsBlockedEither(ctx, user1.ID, user2.ID)
	if !isBlocked {
		t.Error("Expected to be blocked (user1 -> user2)")
	}

	isBlocked, _ = repo.IsBlockedEither(ctx, user2.ID, user1.ID)
	if !isBlocked {
		t.Error("Expected to be blocked (user2 -> user1)")
	}
}

func TestBlockedUserRepository_ListBlocked(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	blocker := createTestUserForBlockedIsolated(t, db, prefix, "blocker")
	blocked1 := createTestUserForBlockedIsolated(t, db, prefix, "blocked1")
	blocked2 := createTestUserForBlockedIsolated(t, db, prefix, "blocked2")
	repo := NewBlockedUserRepository(db)
	ctx := context.Background()

	if err := repo.Block(ctx, blocker.ID, blocked1.ID); err != nil {
		t.Fatalf("Failed to block user 1: %v", err)
	}
	if err := repo.Block(ctx, blocker.ID, blocked2.ID); err != nil {
		t.Fatalf("Failed to block user 2: %v", err)
	}

	blockedUsers, err := repo.ListBlocked(ctx, blocker.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list blocked users: %v", err)
	}

	if len(blockedUsers) != 2 {
		t.Errorf("Expected 2 blocked users, got %d", len(blockedUsers))
	}
}

// ==================== FriendshipRepository Tests ====================

func TestFriendshipRepository_Create(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	friend := createTestUserForBlockedIsolated(t, db, prefix, "friend")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, user.ID, friend.ID)
	if err != nil {
		t.Fatalf("Failed to create friend request: %v", err)
	}

	// Verify request exists
	friendship, _ := repo.GetFriendship(ctx, user.ID, friend.ID)
	if friendship == nil {
		t.Error("Expected friendship to exist")
	}
	if friendship != nil && friendship.Status != model.FriendshipStatusPending {
		t.Errorf("Expected status 'pending', got '%s'", friendship.Status)
	}
}

func TestFriendshipRepository_Accept(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	friend := createTestUserForBlockedIsolated(t, db, prefix, "friend")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	// Create friend request
	if err := repo.Create(ctx, user.ID, friend.ID); err != nil {
		t.Fatalf("Failed to create friend request: %v", err)
	}

	// Accept the request
	err := repo.Accept(ctx, friend.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to accept friend request: %v", err)
	}

	// Verify both directions are accepted
	areFriends, _ := repo.AreFriends(ctx, user.ID, friend.ID)
	if !areFriends {
		t.Error("Expected users to be friends")
	}

	areFriends, _ = repo.AreFriends(ctx, friend.ID, user.ID)
	if !areFriends {
		t.Error("Expected users to be friends (reverse)")
	}
}

func TestFriendshipRepository_Accept_NotFound(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	friend := createTestUserForBlockedIsolated(t, db, prefix, "friend")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	// Try to accept non-existent request
	err := repo.Accept(ctx, friend.ID, user.ID)
	if err != ErrFriendshipNotFound {
		t.Errorf("Expected ErrFriendshipNotFound, got %v", err)
	}
}

func TestFriendshipRepository_Reject(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	friend := createTestUserForBlockedIsolated(t, db, prefix, "friend")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	// Create friend request
	if err := repo.Create(ctx, user.ID, friend.ID); err != nil {
		t.Fatalf("Failed to create friend request: %v", err)
	}

	// Reject the request
	err := repo.Reject(ctx, friend.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to reject friend request: %v", err)
	}

	// Verify not friends
	areFriends, _ := repo.AreFriends(ctx, user.ID, friend.ID)
	if areFriends {
		t.Error("Expected users not to be friends")
	}
}

func TestFriendshipRepository_Remove(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	friend := createTestUserForBlockedIsolated(t, db, prefix, "friend")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	// Create and accept friend request
	if err := repo.Create(ctx, user.ID, friend.ID); err != nil {
		t.Fatalf("Failed to create friend request: %v", err)
	}
	if err := repo.Accept(ctx, friend.ID, user.ID); err != nil {
		t.Fatalf("Failed to accept friend request: %v", err)
	}

	// Remove friendship
	err := repo.Remove(ctx, user.ID, friend.ID)
	if err != nil {
		t.Fatalf("Failed to remove friendship: %v", err)
	}

	// Verify not friends
	areFriends, _ := repo.AreFriends(ctx, user.ID, friend.ID)
	if areFriends {
		t.Error("Expected users not to be friends")
	}
}

func TestFriendshipRepository_ListFriends(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	friend1 := createTestUserForBlockedIsolated(t, db, prefix, "friend1")
	friend2 := createTestUserForBlockedIsolated(t, db, prefix, "friend2")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	// Create and accept friend requests
	if err := repo.Create(ctx, user.ID, friend1.ID); err != nil {
		t.Fatalf("Failed to create friend request 1: %v", err)
	}
	if err := repo.Accept(ctx, friend1.ID, user.ID); err != nil {
		t.Fatalf("Failed to accept friend request 1: %v", err)
	}
	if err := repo.Create(ctx, user.ID, friend2.ID); err != nil {
		t.Fatalf("Failed to create friend request 2: %v", err)
	}
	if err := repo.Accept(ctx, friend2.ID, user.ID); err != nil {
		t.Fatalf("Failed to accept friend request 2: %v", err)
	}

	friends, err := repo.ListFriends(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list friends: %v", err)
	}

	if len(friends) != 2 {
		t.Errorf("Expected 2 friends, got %d", len(friends))
	}
}

func TestFriendshipRepository_ListPendingRequests(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	requester1 := createTestUserForBlockedIsolated(t, db, prefix, "requester1")
	requester2 := createTestUserForBlockedIsolated(t, db, prefix, "requester2")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	// Create pending requests to user
	if err := repo.Create(ctx, requester1.ID, user.ID); err != nil {
		t.Fatalf("Failed to create request 1: %v", err)
	}
	if err := repo.Create(ctx, requester2.ID, user.ID); err != nil {
		t.Fatalf("Failed to create request 2: %v", err)
	}

	pending, err := repo.ListPendingRequests(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list pending requests: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending requests, got %d", len(pending))
	}
}

func TestFriendshipRepository_ListSentRequests(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	target1 := createTestUserForBlockedIsolated(t, db, prefix, "target1")
	target2 := createTestUserForBlockedIsolated(t, db, prefix, "target2")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	// Create sent requests from user
	if err := repo.Create(ctx, user.ID, target1.ID); err != nil {
		t.Fatalf("Failed to create request 1: %v", err)
	}
	if err := repo.Create(ctx, user.ID, target2.ID); err != nil {
		t.Fatalf("Failed to create request 2: %v", err)
	}

	sent, err := repo.ListSentRequests(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list sent requests: %v", err)
	}

	if len(sent) != 2 {
		t.Errorf("Expected 2 sent requests, got %d", len(sent))
	}
}

func TestFriendshipRepository_AreFriends(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	friend := createTestUserForBlockedIsolated(t, db, prefix, "friend")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	// Initially not friends
	areFriends, err := repo.AreFriends(ctx, user.ID, friend.ID)
	if err != nil {
		t.Fatalf("Failed to check friendship: %v", err)
	}
	if areFriends {
		t.Error("Expected not to be friends initially")
	}

	// Send request (still not friends)
	if err := repo.Create(ctx, user.ID, friend.ID); err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	areFriends, _ = repo.AreFriends(ctx, user.ID, friend.ID)
	if areFriends {
		t.Error("Expected not to be friends with pending request")
	}

	// Accept request (now friends)
	if err := repo.Accept(ctx, friend.ID, user.ID); err != nil {
		t.Fatalf("Failed to accept request: %v", err)
	}
	areFriends, _ = repo.AreFriends(ctx, user.ID, friend.ID)
	if !areFriends {
		t.Error("Expected to be friends after accept")
	}
}

func TestFriendshipRepository_GetFriendship(t *testing.T) {
	db, prefix := setupBlockedTestDBIsolated(t)
	defer db.Close()
	defer cleanupBlockedTestByPrefix(t, db, prefix)

	user := createTestUserForBlockedIsolated(t, db, prefix, "user")
	friend := createTestUserForBlockedIsolated(t, db, prefix, "friend")
	repo := NewFriendshipRepository(db)
	ctx := context.Background()

	if err := repo.Create(ctx, user.ID, friend.ID); err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	friendship, err := repo.GetFriendship(ctx, user.ID, friend.ID)
	if err != nil {
		t.Fatalf("Failed to get friendship: %v", err)
	}

	if friendship.UserID != user.ID {
		t.Errorf("Expected user_id '%s', got '%s'", user.ID, friendship.UserID)
	}
	if friendship.FriendID != friend.ID {
		t.Errorf("Expected friend_id '%s', got '%s'", friend.ID, friendship.FriendID)
	}

	// Test not found
	_, err = repo.GetFriendship(ctx, friend.ID, user.ID)
	if err != ErrFriendshipNotFound {
		t.Errorf("Expected ErrFriendshipNotFound, got %v", err)
	}
}
