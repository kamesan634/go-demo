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
const roomNonExistentUUID = "00000000-0000-0000-0000-000000000000"

func setupRoomTestDBIsolated(t *testing.T) (*sqlx.DB, string) {
	t.Helper()
	return SetupIsolatedTestDB(t)
}

func cleanupRoomTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	CleanupTestDataByPrefix(t, db, prefix)
}

func createTestUserForRoomIsolated(t *testing.T, db *sqlx.DB, prefix, username string) *model.User {
	t.Helper()
	return CreateIsolatedTestUser(t, db, prefix, username)
}

func TestRoomRepository_Create(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:        "Test Room",
		Description: sql.NullString{String: "A test room", Valid: true},
		Type:        model.RoomTypePublic,
		OwnerID:     user.ID,
		MaxMembers:  100,
	}

	err := repo.Create(ctx, room)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	if room.ID == "" {
		t.Error("Expected room ID to be set")
	}
	if room.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}
}

func TestRoomRepository_GetByID(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	found, err := repo.GetByID(ctx, room.ID)
	if err != nil {
		t.Fatalf("Failed to get room: %v", err)
	}

	if found.Name != room.Name {
		t.Errorf("Expected name %s, got %s", room.Name, found.Name)
	}

	// Test not found
	_, err = repo.GetByID(ctx, roomNonExistentUUID)
	if err != ErrRoomNotFound {
		t.Errorf("Expected ErrRoomNotFound, got %v", err)
	}
}

func TestRoomRepository_GetByIDWithMemberCount(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	// Add owner as member
	member := &model.RoomMember{
		RoomID: room.ID,
		UserID: user.ID,
		Role:   model.MemberRoleOwner,
	}
	_ = repo.AddMember(ctx, member)

	found, err := repo.GetByIDWithMemberCount(ctx, room.ID)
	if err != nil {
		t.Fatalf("Failed to get room with member count: %v", err)
	}

	if found.MemberCount != 1 {
		t.Errorf("Expected member count 1, got %d", found.MemberCount)
	}
}

func TestRoomRepository_Update(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	room.Name = "Updated Room"
	room.Description = sql.NullString{String: "Updated description", Valid: true}

	err := repo.Update(ctx, room)
	if err != nil {
		t.Fatalf("Failed to update room: %v", err)
	}

	found, _ := repo.GetByID(ctx, room.ID)
	if found.Name != "Updated Room" {
		t.Errorf("Expected name 'Updated Room', got '%s'", found.Name)
	}
}

func TestRoomRepository_Delete(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	err := repo.Delete(ctx, room.ID)
	if err != nil {
		t.Fatalf("Failed to delete room: %v", err)
	}

	_, err = repo.GetByID(ctx, room.ID)
	if err != ErrRoomNotFound {
		t.Errorf("Expected ErrRoomNotFound after delete, got %v", err)
	}
}

func TestRoomRepository_ListPublic(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	// Create public and private rooms
	publicRoom := &model.Room{
		Name:       "Public Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, publicRoom)

	privateRoom := &model.Room{
		Name:       "Private Room",
		Type:       model.RoomTypePrivate,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, privateRoom)

	rooms, err := repo.ListPublic(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list public rooms: %v", err)
	}

	if len(rooms) != 1 {
		t.Errorf("Expected 1 public room, got %d", len(rooms))
	}

	if len(rooms) > 0 && rooms[0].Type != model.RoomTypePublic {
		t.Error("Expected only public rooms")
	}
}

func TestRoomRepository_ListByUserID(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "User's Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	member := &model.RoomMember{
		RoomID: room.ID,
		UserID: user.ID,
		Role:   model.MemberRoleOwner,
	}
	_ = repo.AddMember(ctx, member)

	rooms, err := repo.ListByUserID(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list user rooms: %v", err)
	}

	if len(rooms) != 1 {
		t.Errorf("Expected 1 room, got %d", len(rooms))
	}
}

func TestRoomRepository_Search(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	rooms := []*model.Room{
		{Name: "Tech Talk", Type: model.RoomTypePublic, OwnerID: user.ID, MaxMembers: 100},
		{Name: "General Chat", Type: model.RoomTypePublic, OwnerID: user.ID, MaxMembers: 100},
		{Name: "Random", Type: model.RoomTypePublic, OwnerID: user.ID, MaxMembers: 100},
	}

	for _, r := range rooms {
		_ = repo.Create(ctx, r)
	}

	results, err := repo.Search(ctx, "Tech", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search rooms: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestRoomRepository_AddMember(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	member := &model.RoomMember{
		RoomID: room.ID,
		UserID: user.ID,
		Role:   model.MemberRoleOwner,
	}

	err := repo.AddMember(ctx, member)
	if err != nil {
		t.Fatalf("Failed to add member: %v", err)
	}

	if member.ID == "" {
		t.Error("Expected member ID to be set")
	}

	// Test duplicate member
	err = repo.AddMember(ctx, member)
	if err != ErrAlreadyRoomMember {
		t.Errorf("Expected ErrAlreadyRoomMember, got %v", err)
	}
}

func TestRoomRepository_AddMember_RoomFull(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user1 := createTestUserForRoomIsolated(t, db, prefix, "user1")
	user2 := createTestUserForRoomIsolated(t, db, prefix, "user2")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Small Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user1.ID,
		MaxMembers: 1, // Only 1 member allowed
	}
	_ = repo.Create(ctx, room)

	// Add first member
	member1 := &model.RoomMember{
		RoomID: room.ID,
		UserID: user1.ID,
		Role:   model.MemberRoleOwner,
	}
	_ = repo.AddMember(ctx, member1)

	// Try to add second member
	member2 := &model.RoomMember{
		RoomID: room.ID,
		UserID: user2.ID,
		Role:   model.MemberRoleMember,
	}
	err := repo.AddMember(ctx, member2)
	if err != ErrRoomFull {
		t.Errorf("Expected ErrRoomFull, got %v", err)
	}
}

func TestRoomRepository_RemoveMember(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	member := &model.RoomMember{
		RoomID: room.ID,
		UserID: user.ID,
		Role:   model.MemberRoleOwner,
	}
	_ = repo.AddMember(ctx, member)

	err := repo.RemoveMember(ctx, room.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to remove member: %v", err)
	}

	// Test remove non-member
	err = repo.RemoveMember(ctx, room.ID, user.ID)
	if err != ErrNotRoomMember {
		t.Errorf("Expected ErrNotRoomMember, got %v", err)
	}
}

func TestRoomRepository_GetMember(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "owner")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	member := &model.RoomMember{
		RoomID: room.ID,
		UserID: user.ID,
		Role:   model.MemberRoleOwner,
	}
	_ = repo.AddMember(ctx, member)

	found, err := repo.GetMember(ctx, room.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to get member: %v", err)
	}

	if found.Role != model.MemberRoleOwner {
		t.Errorf("Expected role 'owner', got '%s'", found.Role)
	}

	// Test not a member
	_, err = repo.GetMember(ctx, room.ID, roomNonExistentUUID)
	if err != ErrNotRoomMember {
		t.Errorf("Expected ErrNotRoomMember, got %v", err)
	}
}

func TestRoomRepository_ListMembers(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user1 := createTestUserForRoomIsolated(t, db, prefix, "user1")
	user2 := createTestUserForRoomIsolated(t, db, prefix, "user2")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user1.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	_ = repo.AddMember(ctx, &model.RoomMember{RoomID: room.ID, UserID: user1.ID, Role: model.MemberRoleOwner})
	_ = repo.AddMember(ctx, &model.RoomMember{RoomID: room.ID, UserID: user2.ID, Role: model.MemberRoleMember})

	members, err := repo.ListMembers(ctx, room.ID)
	if err != nil {
		t.Fatalf("Failed to list members: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestRoomRepository_UpdateMemberRole(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "user")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	member := &model.RoomMember{
		RoomID: room.ID,
		UserID: user.ID,
		Role:   model.MemberRoleMember,
	}
	_ = repo.AddMember(ctx, member)

	err := repo.UpdateMemberRole(ctx, room.ID, user.ID, model.MemberRoleAdmin)
	if err != nil {
		t.Fatalf("Failed to update member role: %v", err)
	}

	found, _ := repo.GetMember(ctx, room.ID, user.ID)
	if found.Role != model.MemberRoleAdmin {
		t.Errorf("Expected role 'admin', got '%s'", found.Role)
	}
}

func TestRoomRepository_IsMember(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user := createTestUserForRoomIsolated(t, db, prefix, "user")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	// Not a member yet
	isMember, err := repo.IsMember(ctx, room.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to check membership: %v", err)
	}
	if isMember {
		t.Error("Expected not to be a member")
	}

	// Add as member
	_ = repo.AddMember(ctx, &model.RoomMember{RoomID: room.ID, UserID: user.ID, Role: model.MemberRoleMember})

	isMember, _ = repo.IsMember(ctx, room.ID, user.ID)
	if !isMember {
		t.Error("Expected to be a member")
	}
}

func TestRoomRepository_CountMembers(t *testing.T) {
	db, prefix := setupRoomTestDBIsolated(t)
	defer db.Close()
	defer cleanupRoomTestByPrefix(t, db, prefix)

	user1 := createTestUserForRoomIsolated(t, db, prefix, "user1")
	user2 := createTestUserForRoomIsolated(t, db, prefix, "user2")
	repo := NewRoomRepository(db)
	ctx := context.Background()

	room := &model.Room{
		Name:       "Test Room",
		Type:       model.RoomTypePublic,
		OwnerID:    user1.ID,
		MaxMembers: 100,
	}
	_ = repo.Create(ctx, room)

	_ = repo.AddMember(ctx, &model.RoomMember{RoomID: room.ID, UserID: user1.ID, Role: model.MemberRoleOwner})
	_ = repo.AddMember(ctx, &model.RoomMember{RoomID: room.ID, UserID: user2.ID, Role: model.MemberRoleMember})

	count, err := repo.CountMembers(ctx, room.ID)
	if err != nil {
		t.Fatalf("Failed to count members: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 members, got %d", count)
	}
}
