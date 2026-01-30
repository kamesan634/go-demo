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

func setupTestRoomServiceIsolated(t *testing.T) (*RoomService, *sqlx.DB, string) {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=chat_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test, could not connect to test database: %v", err)
	}

	roomRepo := repository.NewRoomRepository(db)
	userRepo := repository.NewUserRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	logger := zap.NewNop()

	service := NewRoomService(roomRepo, userRepo, messageRepo, logger)
	prefix := repository.GenerateUniquePrefix()
	return service, db, prefix
}

func cleanupRoomServiceTestByPrefix(t *testing.T, db *sqlx.DB, prefix string) {
	t.Helper()
	repository.CleanupTestDataByPrefix(t, db, prefix)
}

func createUserForRoomServiceTestIsolated(t *testing.T, db *sqlx.DB, prefix, username string) *model.User {
	t.Helper()
	return repository.CreateIsolatedTestUser(t, db, prefix, username)
}

func createRoomForRoomServiceTestIsolated(t *testing.T, service *RoomService, prefix string, owner *model.User, roomType model.RoomType) *model.Room {
	t.Helper()
	ctx := context.Background()
	room, err := service.Create(ctx, &CreateRoomInput{
		Name:    prefix + "_test_room",
		Type:    roomType,
		OwnerID: owner.ID,
	})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	return room
}

func TestRoomService_Create(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	room, err := service.Create(ctx, &CreateRoomInput{
		Name:        prefix + "_Test Room",
		Description: "A test room",
		Type:        model.RoomTypePublic,
		OwnerID:     owner.ID,
	})

	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	if room.ID == "" {
		t.Error("Expected room ID to be set")
	}
	if room.Name != prefix+"_Test Room" {
		t.Errorf("Expected name '%s_Test Room', got '%s'", prefix, room.Name)
	}
}

func TestRoomService_Create_DefaultMaxMembers(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    prefix + "_Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	if room.MaxMembers != 100 {
		t.Errorf("Expected default max members 100, got %d", room.MaxMembers)
	}
}

func TestRoomService_GetByID(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	created := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	found, err := service.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get room: %v", err)
	}

	if found.Name != created.Name {
		t.Errorf("Expected name '%s', got '%s'", created.Name, found.Name)
	}
}

func TestRoomService_GetByID_NotFound(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	ctx := context.Background()

	_, err := service.GetByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent room")
	}
}

func TestRoomService_GetByIDWithDetails(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	created := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	detail, err := service.GetByIDWithDetails(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get room with details: %v", err)
	}

	if detail.MemberCount != 1 {
		t.Errorf("Expected member count 1, got %d", detail.MemberCount)
	}
	if detail.Owner == nil {
		t.Error("Expected owner to be set")
	}
}

func TestRoomService_Update(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	newName := prefix + "_Updated Name"
	updated, err := service.Update(ctx, &UpdateRoomInput{
		RoomID: room.ID,
		UserID: owner.ID,
		Name:   &newName,
	})

	if err != nil {
		t.Fatalf("Failed to update room: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("Expected name '%s', got '%s'", newName, updated.Name)
	}
}

func TestRoomService_Update_NoPermission(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	otherUser := createUserForRoomServiceTestIsolated(t, db, prefix, "other")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	newName := prefix + "_Updated Name"
	_, err := service.Update(ctx, &UpdateRoomInput{
		RoomID: room.ID,
		UserID: otherUser.ID,
		Name:   &newName,
	})

	if err == nil {
		t.Error("Expected permission denied error")
	}
}

func TestRoomService_Delete(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	err := service.Delete(ctx, room.ID, owner.ID)
	if err != nil {
		t.Fatalf("Failed to delete room: %v", err)
	}

	_, err = service.GetByID(ctx, room.ID)
	if err == nil {
		t.Error("Expected room to be deleted")
	}
}

func TestRoomService_Delete_NoPermission(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	otherUser := createUserForRoomServiceTestIsolated(t, db, prefix, "other")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	err := service.Delete(ctx, room.ID, otherUser.ID)
	if err == nil {
		t.Error("Expected permission denied error")
	}
}

func TestRoomService_ListPublic(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	service.Create(ctx, &CreateRoomInput{Name: prefix + "_Public 1", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: prefix + "_Public 2", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: prefix + "_Private", Type: model.RoomTypePrivate, OwnerID: owner.ID})

	rooms, err := service.ListPublic(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list public rooms: %v", err)
	}

	// Count rooms with our prefix
	count := 0
	for _, r := range rooms {
		if len(r.Name) > len(prefix) && r.Name[:len(prefix)] == prefix {
			count++
		}
	}

	if count != 2 {
		t.Errorf("Expected 2 public rooms with prefix, got %d", count)
	}
}

func TestRoomService_ListByUserID(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	service.Create(ctx, &CreateRoomInput{Name: prefix + "_Room 1", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: prefix + "_Room 2", Type: model.RoomTypePublic, OwnerID: owner.ID})

	rooms, err := service.ListByUserID(ctx, owner.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list user rooms: %v", err)
	}

	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(rooms))
	}
}

func TestRoomService_Search(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	service.Create(ctx, &CreateRoomInput{Name: prefix + "_Tech Talk", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: prefix + "_General", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: prefix + "_Random", Type: model.RoomTypePublic, OwnerID: owner.ID})

	rooms, err := service.Search(ctx, prefix+"_Tech", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search rooms: %v", err)
	}

	if len(rooms) != 1 {
		t.Errorf("Expected 1 room, got %d", len(rooms))
	}
}

func TestRoomService_Join(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	member := createUserForRoomServiceTestIsolated(t, db, prefix, "member")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	err := service.Join(ctx, room.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to join room: %v", err)
	}

	isMember, _ := service.IsMember(ctx, room.ID, member.ID)
	if !isMember {
		t.Error("Expected user to be a member")
	}
}

func TestRoomService_Join_PrivateRoom(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	member := createUserForRoomServiceTestIsolated(t, db, prefix, "member")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePrivate)

	err := service.Join(ctx, room.ID, member.ID)
	if err == nil {
		t.Error("Expected permission denied for private room")
	}
}

func TestRoomService_Leave(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	member := createUserForRoomServiceTestIsolated(t, db, prefix, "member")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	service.Join(ctx, room.ID, member.ID)

	err := service.Leave(ctx, room.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to leave room: %v", err)
	}

	isMember, _ := service.IsMember(ctx, room.ID, member.ID)
	if isMember {
		t.Error("Expected user not to be a member")
	}
}

func TestRoomService_Leave_OwnerCannotLeave(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	err := service.Leave(ctx, room.ID, owner.ID)
	if err == nil {
		t.Error("Expected error when owner tries to leave")
	}
}

func TestRoomService_InviteMember(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	invitee := createUserForRoomServiceTestIsolated(t, db, prefix, "invitee")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePrivate)

	err := service.InviteMember(ctx, room.ID, owner.ID, invitee.ID)
	if err != nil {
		t.Fatalf("Failed to invite member: %v", err)
	}

	isMember, _ := service.IsMember(ctx, room.ID, invitee.ID)
	if !isMember {
		t.Error("Expected invitee to be a member")
	}
}

func TestRoomService_KickMember(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	member := createUserForRoomServiceTestIsolated(t, db, prefix, "member")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	service.Join(ctx, room.ID, member.ID)

	err := service.KickMember(ctx, room.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to kick member: %v", err)
	}

	isMember, _ := service.IsMember(ctx, room.ID, member.ID)
	if isMember {
		t.Error("Expected member to be kicked")
	}
}

func TestRoomService_KickMember_CannotKickOwner(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	admin := createUserForRoomServiceTestIsolated(t, db, prefix, "admin")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	service.Join(ctx, room.ID, admin.ID)
	service.PromoteMember(ctx, room.ID, owner.ID, admin.ID)

	err := service.KickMember(ctx, room.ID, admin.ID, owner.ID)
	if err == nil {
		t.Error("Expected error when trying to kick owner")
	}
}

func TestRoomService_PromoteMember(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	member := createUserForRoomServiceTestIsolated(t, db, prefix, "member")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	service.Join(ctx, room.ID, member.ID)

	err := service.PromoteMember(ctx, room.ID, owner.ID, member.ID)
	if err != nil {
		t.Fatalf("Failed to promote member: %v", err)
	}

	memberInfo, _ := service.GetMember(ctx, room.ID, member.ID)
	if memberInfo.Role != model.MemberRoleAdmin {
		t.Errorf("Expected role 'admin', got '%s'", memberInfo.Role)
	}
}

func TestRoomService_DemoteMember(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	admin := createUserForRoomServiceTestIsolated(t, db, prefix, "admin")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	service.Join(ctx, room.ID, admin.ID)
	service.PromoteMember(ctx, room.ID, owner.ID, admin.ID)

	err := service.DemoteMember(ctx, room.ID, owner.ID, admin.ID)
	if err != nil {
		t.Fatalf("Failed to demote member: %v", err)
	}

	memberInfo, _ := service.GetMember(ctx, room.ID, admin.ID)
	if memberInfo.Role != model.MemberRoleMember {
		t.Errorf("Expected role 'member', got '%s'", memberInfo.Role)
	}
}

func TestRoomService_ListMembers(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	member := createUserForRoomServiceTestIsolated(t, db, prefix, "member")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	service.Join(ctx, room.ID, member.ID)

	members, err := service.ListMembers(ctx, room.ID, owner.ID)
	if err != nil {
		t.Fatalf("Failed to list members: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestRoomService_IsMember(t *testing.T) {
	service, db, prefix := setupTestRoomServiceIsolated(t)
	defer db.Close()
	defer cleanupRoomServiceTestByPrefix(t, db, prefix)

	owner := createUserForRoomServiceTestIsolated(t, db, prefix, "owner")
	member := createUserForRoomServiceTestIsolated(t, db, prefix, "member")
	ctx := context.Background()

	room := createRoomForRoomServiceTestIsolated(t, service, prefix, owner, model.RoomTypePublic)

	// Member not joined yet
	isMember, _ := service.IsMember(ctx, room.ID, member.ID)
	if isMember {
		t.Error("Expected not to be a member")
	}

	// After joining
	service.Join(ctx, room.ID, member.ID)
	isMember, _ = service.IsMember(ctx, room.ID, member.ID)
	if !isMember {
		t.Error("Expected to be a member")
	}
}
