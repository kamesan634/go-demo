package service

import (
	"context"
	"database/sql"

	"github.com/go-demo/chat/internal/model"
	apperrors "github.com/go-demo/chat/internal/pkg/errors"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/go-demo/chat/internal/repository"
	"go.uber.org/zap"
)

type AuthService struct {
	userRepo   *repository.UserRepository
	jwtManager *utils.JWTManager
	logger     *zap.Logger
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *utils.JWTManager, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// RegisterInput represents registration input
type RegisterInput struct {
	Username string
	Email    string
	Password string
}

// RegisterResult represents registration result
type RegisterResult struct {
	User      *model.User
	TokenPair *utils.TokenPair
}

// Register registers a new user
func (s *AuthService) Register(ctx context.Context, input *RegisterInput) (*RegisterResult, error) {
	// Check if username exists
	exists, err := s.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		s.logger.Error("Failed to check username", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	if exists {
		return nil, apperrors.ErrUsernameExists
	}

	// Check if email exists
	exists, err = s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		s.logger.Error("Failed to check email", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	if exists {
		return nil, apperrors.ErrEmailExists
	}

	// Hash password
	passwordHash, err := utils.HashPassword(input.Password)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	// Create user
	user := &model.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: passwordHash,
		Status:       model.UserStatusOffline,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	// Generate tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		s.logger.Error("Failed to generate token pair", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	s.logger.Info("User registered",
		zap.String("user_id", user.ID),
		zap.String("username", user.Username),
	)

	return &RegisterResult{
		User:      user,
		TokenPair: tokenPair,
	}, nil
}

// LoginInput represents login input
type LoginInput struct {
	Username string
	Password string
}

// LoginResult represents login result
type LoginResult struct {
	User      *model.User
	TokenPair *utils.TokenPair
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, input *LoginInput) (*LoginResult, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, input.Username)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.ErrInvalidPassword
		}
		s.logger.Error("Failed to get user", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	// Check password
	if !utils.CheckPassword(input.Password, user.PasswordHash) {
		return nil, apperrors.ErrInvalidPassword
	}

	// Generate tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		s.logger.Error("Failed to generate token pair", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	// Update status to online
	if err := s.userRepo.UpdateStatus(ctx, user.ID, model.UserStatusOnline); err != nil {
		s.logger.Warn("Failed to update user status", zap.Error(err))
	}

	s.logger.Info("User logged in",
		zap.String("user_id", user.ID),
		zap.String("username", user.Username),
	)

	return &LoginResult{
		User:      user,
		TokenPair: tokenPair,
	}, nil
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		if err == utils.ErrExpiredToken {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}

	// Generate new token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(claims.UserID, claims.Username)
	if err != nil {
		s.logger.Error("Failed to generate token pair", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return tokenPair, nil
}

// Logout logs out a user
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	// Update status to offline
	if err := s.userRepo.UpdateStatus(ctx, userID, model.UserStatusOffline); err != nil {
		s.logger.Warn("Failed to update user status on logout", zap.Error(err))
	}

	s.logger.Info("User logged out", zap.String("user_id", userID))
	return nil
}

// ChangePasswordInput represents change password input
type ChangePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

// ChangePassword changes user password
func (s *AuthService) ChangePassword(ctx context.Context, input *ChangePasswordInput) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.ErrUserNotFound
		}
		s.logger.Error("Failed to get user", zap.Error(err))
		return apperrors.ErrInternal
	}

	// Check current password
	if !utils.CheckPassword(input.CurrentPassword, user.PasswordHash) {
		return apperrors.ErrInvalidPassword
	}

	// Validate new password
	if err := utils.ValidatePassword(input.NewPassword); err != nil {
		return apperrors.ErrValidation.WithDetails(map[string]string{
			"new_password": err.Error(),
		})
	}

	// Hash new password
	passwordHash, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return apperrors.ErrInternal
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, input.UserID, passwordHash); err != nil {
		s.logger.Error("Failed to update password", zap.Error(err))
		return apperrors.ErrInternal
	}

	s.logger.Info("User changed password", zap.String("user_id", input.UserID))
	return nil
}

// ValidateToken validates an access token and returns user info
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*model.User, error) {
	claims, err := s.jwtManager.ValidateAccessToken(token)
	if err != nil {
		if err == utils.ErrExpiredToken {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.ErrUserNotFound
		}
		s.logger.Error("Failed to get user", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	return user, nil
}

// GetUserByID retrieves a user by ID (for internal use)
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// UpdateUserStatus updates user online status
func (s *AuthService) UpdateUserStatus(ctx context.Context, userID string, status model.UserStatus) error {
	return s.userRepo.UpdateStatus(ctx, userID, status)
}

// UpdateProfile updates user profile
func (s *AuthService) UpdateProfile(ctx context.Context, user *model.User) error {
	existingUser, err := s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.ErrUserNotFound
		}
		return apperrors.ErrInternal
	}

	// Update only allowed fields
	existingUser.DisplayName = user.DisplayName
	existingUser.AvatarURL = user.AvatarURL
	existingUser.Bio = user.Bio

	if err := s.userRepo.Update(ctx, existingUser); err != nil {
		s.logger.Error("Failed to update user", zap.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}

// SetDisplayName sets a user's display name
func (s *AuthService) SetDisplayName(ctx context.Context, userID, displayName string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return apperrors.ErrUserNotFound
		}
		return apperrors.ErrInternal
	}

	user.DisplayName = sql.NullString{String: displayName, Valid: displayName != ""}
	return s.userRepo.Update(ctx, user)
}
