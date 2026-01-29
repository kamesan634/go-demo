package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/dto/request"
	"github.com/go-demo/chat/internal/dto/response"
	"github.com/go-demo/chat/internal/middleware"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/go-demo/chat/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register godoc
// @Summary 用戶註冊
// @Description 創建新用戶帳號
// @Tags 認證
// @Accept json
// @Produce json
// @Param request body request.RegisterRequest true "註冊資料"
// @Success 201 {object} response.Response{data=response.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	// Validate input
	v := utils.NewValidator()
	v.ValidateUsername("username", req.Username)
	v.ValidateEmail("email", req.Email)
	v.ValidatePassword("password", req.Password)

	if v.HasErrors() {
		response.ValidationError(c, v.Errors())
		return
	}

	result, err := h.authService.Register(c.Request.Context(), &service.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, &response.AuthResponse{
		User: response.NewUserResponse(result.User, true),
		Token: &response.TokenResponse{
			AccessToken:  result.TokenPair.AccessToken,
			RefreshToken: result.TokenPair.RefreshToken,
			ExpiresAt:    result.TokenPair.ExpiresAt,
			TokenType:    "Bearer",
		},
	})
}

// Login godoc
// @Summary 用戶登入
// @Description 用戶登入並獲取 Token
// @Tags 認證
// @Accept json
// @Produce json
// @Param request body request.LoginRequest true "登入資料"
// @Success 200 {object} response.Response{data=response.AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	result, err := h.authService.Login(c.Request.Context(), &service.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, &response.AuthResponse{
		User: response.NewUserResponse(result.User, true),
		Token: &response.TokenResponse{
			AccessToken:  result.TokenPair.AccessToken,
			RefreshToken: result.TokenPair.RefreshToken,
			ExpiresAt:    result.TokenPair.ExpiresAt,
			TokenType:    "Bearer",
		},
	})
}

// Logout godoc
// @Summary 用戶登出
// @Description 用戶登出
// @Tags 認證
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.authService.Logout(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "登出成功", nil)
}

// RefreshToken godoc
// @Summary 刷新 Token
// @Description 使用 Refresh Token 獲取新的 Access Token
// @Tags 認證
// @Accept json
// @Produce json
// @Param request body request.RefreshTokenRequest true "刷新資料"
// @Success 200 {object} response.Response{data=response.TokenResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req request.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, &response.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    "Bearer",
	})
}

// ChangePassword godoc
// @Summary 修改密碼
// @Description 修改當前用戶密碼
// @Tags 認證
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.ChangePasswordRequest true "密碼資料"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req request.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	userID := middleware.GetUserID(c)

	err := h.authService.ChangePassword(c.Request.Context(), &service.ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "密碼修改成功", nil)
}

// GetMe godoc
// @Summary 獲取當前用戶資訊
// @Description 獲取當前登入用戶的資訊
// @Tags 認證
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=response.UserResponse}
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, response.NewUserResponse(user, true))
}

// UpdateProfile godoc
// @Summary 更新個人資料
// @Description 更新當前用戶的個人資料
// @Tags 認證
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.UpdateProfileRequest true "個人資料"
// @Success 200 {object} response.Response{data=response.UserResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req request.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	userID := middleware.GetUserID(c)

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// Update fields
	if req.DisplayName != nil {
		if err := h.authService.SetDisplayName(c.Request.Context(), userID, *req.DisplayName); err != nil {
			response.Error(c, err)
			return
		}
	}

	// Reload user
	user, err = h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, response.NewUserResponse(user, true))
}
