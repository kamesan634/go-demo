package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-demo/chat/internal/config"
	"github.com/go-demo/chat/internal/model"
	"github.com/go-demo/chat/internal/pkg/database"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/go-demo/chat/internal/repository"
	"go.uber.org/zap"
)

func main() {
	log.Println("Starting database seed...")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	logger := zap.NewNop()
	db, err := database.NewPostgres(&cfg.Database, logger)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// Seed users
	log.Println("Creating users...")
	users := []struct {
		username    string
		email       string
		password    string
		displayName string
	}{
		{"alice", "alice@example.com", "password123", "Alice Chen"},
		{"bob", "bob@example.com", "password123", "Bob Wang"},
		{"charlie", "charlie@example.com", "password123", "Charlie Lin"},
		{"diana", "diana@example.com", "password123", "Diana Wu"},
		{"evan", "evan@example.com", "password123", "Evan Lee"},
	}

	var createdUsers []*model.User
	for _, u := range users {
		hash, _ := utils.HashPassword(u.password)
		user := &model.User{
			Username:     u.username,
			Email:        u.email,
			PasswordHash: hash,
			DisplayName:  sql.NullString{String: u.displayName, Valid: true},
			Status:       model.UserStatusOffline,
		}

		if err := userRepo.Create(ctx, user); err != nil {
			log.Printf("User %s might already exist: %v", u.username, err)
			existing, _ := userRepo.GetByUsername(ctx, u.username)
			if existing != nil {
				createdUsers = append(createdUsers, existing)
			}
		} else {
			createdUsers = append(createdUsers, user)
			log.Printf("Created user: %s", u.username)
		}
	}

	if len(createdUsers) < 2 {
		log.Println("Not enough users, skipping room and message creation")
		return
	}

	// Seed rooms
	log.Println("Creating rooms...")
	rooms := []struct {
		name        string
		description string
		roomType    model.RoomType
		ownerIndex  int
	}{
		{"General", "一般討論區", model.RoomTypePublic, 0},
		{"Tech Talk", "技術討論區", model.RoomTypePublic, 1},
		{"Random", "隨便聊聊", model.RoomTypePublic, 2},
		{"Team Alpha", "Alpha 團隊專用", model.RoomTypePrivate, 0},
	}

	var createdRooms []*model.Room
	for _, r := range rooms {
		if r.ownerIndex >= len(createdUsers) {
			continue
		}

		room := &model.Room{
			Name:       r.name,
			Type:       r.roomType,
			OwnerID:    createdUsers[r.ownerIndex].ID,
			MaxMembers: 100,
		}
		if r.description != "" {
			room.Description = sql.NullString{String: r.description, Valid: true}
		}

		if err := roomRepo.Create(ctx, room); err != nil {
			log.Printf("Room %s might already exist: %v", r.name, err)
		} else {
			createdRooms = append(createdRooms, room)
			log.Printf("Created room: %s", r.name)

			// Add owner as member
			member := &model.RoomMember{
				RoomID: room.ID,
				UserID: createdUsers[r.ownerIndex].ID,
				Role:   model.MemberRoleOwner,
			}
			_ = roomRepo.AddMember(ctx, member)
		}
	}

	// Add members to rooms
	log.Println("Adding members to rooms...")
	for _, room := range createdRooms {
		for i, user := range createdUsers {
			if user.ID == room.OwnerID {
				continue // Skip owner, already added
			}

			// Add some users to each room
			if i%2 == 0 || room.Type == model.RoomTypePublic {
				member := &model.RoomMember{
					RoomID: room.ID,
					UserID: user.ID,
					Role:   model.MemberRoleMember,
				}
				if err := roomRepo.AddMember(ctx, member); err == nil {
					log.Printf("Added %s to room %s", user.Username, room.Name)
				}
			}
		}
	}

	// Seed messages
	log.Println("Creating messages...")
	messages := []struct {
		roomIndex int
		userIndex int
		content   string
	}{
		{0, 0, "大家好！歡迎來到聊天室！"},
		{0, 1, "Hello everyone!"},
		{0, 2, "很高興認識大家 👋"},
		{0, 3, "這個聊天室真不錯"},
		{1, 1, "有人最近在用什麼新技術嗎？"},
		{1, 0, "我最近在學 Go 語言"},
		{1, 2, "Go 語言的 goroutine 真的很強大"},
		{2, 2, "今天天氣真好"},
		{2, 4, "週末有什麼計劃嗎？"},
	}

	for _, m := range messages {
		if m.roomIndex >= len(createdRooms) || m.userIndex >= len(createdUsers) {
			continue
		}

		msg := &model.Message{
			RoomID:  createdRooms[m.roomIndex].ID,
			UserID:  createdUsers[m.userIndex].ID,
			Content: m.content,
			Type:    model.MessageTypeText,
		}

		if err := messageRepo.Create(ctx, msg); err != nil {
			log.Printf("Failed to create message: %v", err)
		} else {
			log.Printf("Created message in %s", createdRooms[m.roomIndex].Name)
		}

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	log.Println("Seed completed successfully!")
	fmt.Println("\n--- Test Accounts ---")
	fmt.Println("All accounts have password: password123")
	for _, u := range users {
		fmt.Printf("Username: %s, Email: %s\n", u.username, u.email)
	}
}
