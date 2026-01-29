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

func setupTestRoomService(t *testing.T) (*RoomService, *sqlx.DB) {
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
	return service, db
}

func cleanupRoomServiceTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	db.Exec("TRUNCATE rooms, room_members, users, messages CASCADE")
}

func createUserForRoomServiceTest(t *testing.T, db *sqlx.DB, username string) *model.User {
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

func TestRoomService_Create(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	room, err := service.Create(ctx, &CreateRoomInput{
		Name:        "Test Room",
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
	if room.Name != "Test Room" {
		t.Errorf("Expected name 'Test Room', got '%s'", room.Name)
	}
}

func TestRoomService_Create_DefaultMaxMembers(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	if room.MaxMembers != 100 {
		t.Errorf("Expected default max members 100, got %d", room.MaxMembers)
	}
}

func TestRoomService_GetByID(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	created, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	found, err := service.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get room: %v", err)
	}

	if found.Name != created.Name {
		t.Errorf("Expected name '%s', got '%s'", created.Name, found.Name)
	}
}

func TestRoomService_GetByID_NotFound(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	ctx := context.Background()

	_, err := service.GetByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent room")
	}
}

func TestRoomService_GetByIDWithDetails(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	created, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Original Name",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	newName := "Updated Name"
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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	otherUser := createUserForRoomServiceTest(t, db, "other")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	newName := "Updated Name"
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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	otherUser := createUserForRoomServiceTest(t, db, "other")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	err := service.Delete(ctx, room.ID, otherUser.ID)
	if err == nil {
		t.Error("Expected permission denied error")
	}
}

func TestRoomService_ListPublic(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	service.Create(ctx, &CreateRoomInput{Name: "Public 1", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: "Public 2", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: "Private", Type: model.RoomTypePrivate, OwnerID: owner.ID})

	rooms, err := service.ListPublic(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list public rooms: %v", err)
	}

	if len(rooms) != 2 {
		t.Errorf("Expected 2 public rooms, got %d", len(rooms))
	}
}

func TestRoomService_ListByUserID(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	service.Create(ctx, &CreateRoomInput{Name: "Room 1", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: "Room 2", Type: model.RoomTypePublic, OwnerID: owner.ID})

	rooms, err := service.ListByUserID(ctx, owner.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list user rooms: %v", err)
	}

	if len(rooms) != 2 {
		t.Errorf("Expected 2 rooms, got %d", len(rooms))
	}
}

func TestRoomService_Search(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	service.Create(ctx, &CreateRoomInput{Name: "Tech Talk", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: "General", Type: model.RoomTypePublic, OwnerID: owner.ID})
	service.Create(ctx, &CreateRoomInput{Name: "Random", Type: model.RoomTypePublic, OwnerID: owner.ID})

	rooms, err := service.Search(ctx, "Tech", 10, 0)
	if err != nil {
		t.Fatalf("Failed to search rooms: %v", err)
	}

	if len(rooms) != 1 {
		t.Errorf("Expected 1 room, got %d", len(rooms))
	}
}

func TestRoomService_Join(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	member := createUserForRoomServiceTest(t, db, "member")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Public Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	member := createUserForRoomServiceTest(t, db, "member")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Private Room",
		Type:    model.RoomTypePrivate,
		OwnerID: owner.ID,
	})

	err := service.Join(ctx, room.ID, member.ID)
	if err == nil {
		t.Error("Expected permission denied for private room")
	}
}

func TestRoomService_Leave(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	member := createUserForRoomServiceTest(t, db, "member")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Public Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	err := service.Leave(ctx, room.ID, owner.ID)
	if err == nil {
		t.Error("Expected error when owner tries to leave")
	}
}

func TestRoomService_InviteMember(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	invitee := createUserForRoomServiceTest(t, db, "invitee")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Private Room",
		Type:    model.RoomTypePrivate,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	member := createUserForRoomServiceTest(t, db, "member")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	admin := createUserForRoomServiceTest(t, db, "admin")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

	service.Join(ctx, room.ID, admin.ID)
	service.PromoteMember(ctx, room.ID, owner.ID, admin.ID)

	err := service.KickMember(ctx, room.ID, admin.ID, owner.ID)
	if err == nil {
		t.Error("Expected error when trying to kick owner")
	}
}

func TestRoomService_PromoteMember(t *testing.T) {
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	member := createUserForRoomServiceTest(t, db, "member")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	admin := createUserForRoomServiceTest(t, db, "admin")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	member := createUserForRoomServiceTest(t, db, "member")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
	service, db := setupTestRoomService(t)
	defer db.Close()
	defer cleanupRoomServiceTestDB(t, db)

	owner := createUserForRoomServiceTest(t, db, "owner")
	member := createUserForRoomServiceTest(t, db, "member")
	ctx := context.Background()

	room, _ := service.Create(ctx, &CreateRoomInput{
		Name:    "Test Room",
		Type:    model.RoomTypePublic,
		OwnerID: owner.ID,
	})

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
