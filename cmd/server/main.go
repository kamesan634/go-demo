package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/config"
	"github.com/go-demo/chat/internal/handler"
	"github.com/go-demo/chat/internal/middleware"
	"github.com/go-demo/chat/internal/pkg/cache"
	"github.com/go-demo/chat/internal/pkg/database"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/go-demo/chat/internal/repository"
	"github.com/go-demo/chat/internal/service"
	"github.com/go-demo/chat/internal/ws"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// @title           Chat API
// @version         1.0
// @description     Go 即時聊天室系統 API
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := initLogger(cfg.Log.Level)
	defer logger.Sync()

	logger.Info("Starting chat server",
		zap.String("mode", cfg.Server.Mode),
		zap.Int("port", cfg.Server.Port),
	)

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database
	db, err := database.NewPostgres(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close(db, logger)

	// Initialize Redis
	redisClient, err := cache.NewRedis(&cfg.Redis, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer cache.Close(redisClient, logger)

	// Initialize JWT manager
	jwtManager := utils.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
		cfg.JWT.Issuer,
	)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	dmRepo := repository.NewDirectMessageRepository(db)
	blockedRepo := repository.NewBlockedUserRepository(db)
	friendshipRepo := repository.NewFriendshipRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, jwtManager, logger)
	userService := service.NewUserService(userRepo, blockedRepo, friendshipRepo, logger)
	roomService := service.NewRoomService(roomRepo, userRepo, messageRepo, logger)
	messageService := service.NewMessageService(messageRepo, roomRepo, logger)
	dmService := service.NewDirectMessageService(dmRepo, userRepo, blockedRepo, logger)

	// Initialize WebSocket hub
	hub := ws.NewHub(roomService, messageService, dmService, userService, redisClient, logger)
	go hub.Run()

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	roomHandler := handler.NewRoomHandler(roomService)
	messageHandler := handler.NewMessageHandler(messageService, roomService, dmService)
	uploadHandler := handler.NewUploadHandler(fmt.Sprintf("http://localhost:%d", cfg.Server.Port))
	wsHandler := ws.NewHandler(hub, jwtManager, logger)

	// Setup router
	router := setupRouter(
		cfg,
		logger,
		jwtManager,
		redisClient,
		authHandler,
		userHandler,
		roomHandler,
		messageHandler,
		uploadHandler,
		wsHandler,
	)

	// Create server
	srv := &http.Server{
		Addr:         cfg.Server.GetAddr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server is running",
			zap.String("addr", srv.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return logger
}

func setupRouter(
	cfg *config.Config,
	logger *zap.Logger,
	jwtManager *utils.JWTManager,
	redisClient *redis.Client,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	roomHandler *handler.RoomHandler,
	messageHandler *handler.MessageHandler,
	uploadHandler *handler.UploadHandler,
	wsHandler *ws.Handler,
) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Static files for uploads
	router.Static("/uploads", "./uploads")

	// WebSocket endpoint
	router.GET("/ws", wsHandler.ServeWS)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Auth routes (protected)
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.Auth(jwtManager))
		{
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.PUT("/password", authHandler.ChangePassword)
			authProtected.GET("/me", authHandler.GetMe)
			authProtected.PUT("/profile", authHandler.UpdateProfile)
		}

		// User routes
		users := v1.Group("/users")
		users.Use(middleware.Auth(jwtManager))
		{
			users.GET("/search", userHandler.Search)
			users.GET("/online", userHandler.GetOnlineUsers)
			users.GET("/blocked", userHandler.ListBlockedUsers)
			users.GET("/friends", userHandler.ListFriends)
			users.GET("/friend-requests/pending", userHandler.ListPendingRequests)
			users.GET("/friend-requests/sent", userHandler.ListSentRequests)
			users.GET("/:id", userHandler.GetProfile)
			users.POST("/:id/block", userHandler.BlockUser)
			users.POST("/:id/unblock", userHandler.UnblockUser)
			users.POST("/:id/friend-request", userHandler.SendFriendRequest)
			users.POST("/:id/friend-request/accept", userHandler.AcceptFriendRequest)
			users.POST("/:id/friend-request/reject", userHandler.RejectFriendRequest)
			users.DELETE("/:id/friend", userHandler.RemoveFriend)
		}

		// Room routes
		rooms := v1.Group("/rooms")
		rooms.Use(middleware.Auth(jwtManager))
		{
			rooms.GET("", roomHandler.ListPublic)
			rooms.POST("", roomHandler.Create)
			rooms.GET("/me", roomHandler.ListMyRooms)
			rooms.GET("/search", roomHandler.Search)
			rooms.GET("/:id", roomHandler.GetByID)
			rooms.PUT("/:id", roomHandler.Update)
			rooms.DELETE("/:id", roomHandler.Delete)
			rooms.POST("/:id/join", roomHandler.Join)
			rooms.POST("/:id/leave", roomHandler.Leave)
			rooms.POST("/:id/invite", roomHandler.InviteMember)
			rooms.GET("/:id/members", roomHandler.ListMembers)
			rooms.POST("/:id/members/:user_id/kick", roomHandler.KickMember)
			rooms.POST("/:id/members/:user_id/promote", roomHandler.PromoteMember)
			rooms.POST("/:id/members/:user_id/demote", roomHandler.DemoteMember)

			// Room messages
			rooms.GET("/:room_id/messages", messageHandler.GetMessages)
			rooms.POST("/:room_id/messages", messageHandler.SendMessage)
			rooms.PUT("/:room_id/messages/:message_id", messageHandler.UpdateMessage)
			rooms.DELETE("/:room_id/messages/:message_id", messageHandler.DeleteMessage)
			rooms.GET("/:room_id/messages/search", messageHandler.SearchMessages)
			rooms.POST("/:room_id/messages/read", messageHandler.MarkAsRead)
		}

		// Direct message routes
		dm := v1.Group("/dm")
		dm.Use(middleware.Auth(jwtManager))
		{
			dm.GET("", messageHandler.ListConversations)
			dm.GET("/unread", messageHandler.GetUnreadCount)
			dm.GET("/:user_id", messageHandler.GetConversation)
			dm.POST("/:user_id", messageHandler.SendDirectMessage)
			dm.POST("/:user_id/read", messageHandler.MarkDMAsRead)
		}

		// Upload routes
		upload := v1.Group("/upload")
		upload.Use(middleware.Auth(jwtManager))
		{
			upload.POST("/image", uploadHandler.UploadImage)
			upload.POST("/file", uploadHandler.UploadFile)
			upload.POST("/avatar", uploadHandler.UploadAvatar)
		}

		// WebSocket stats (admin)
		wsStats := v1.Group("/ws")
		wsStats.Use(middleware.Auth(jwtManager))
		{
			wsStats.GET("/stats", wsHandler.GetStats)
			wsStats.GET("/online", wsHandler.GetOnlineUsers)
			wsStats.GET("/online/:user_id", wsHandler.IsUserOnline)
		}
	}

	return router
}
