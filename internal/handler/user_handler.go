package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/dto/request"
	"github.com/go-demo/chat/internal/dto/response"
	"github.com/go-demo/chat/internal/middleware"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/go-demo/chat/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetProfile godoc
// @Summary 獲取用戶資料
// @Description 獲取指定用戶的公開資料
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用戶 ID"
// @Success 200 {object} response.Response{data=response.ProfileResponse}
// @Failure 404 {object} response.Response
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.Param("id")

	if !utils.ValidateUUID(userID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	profile, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, response.NewProfileResponse(profile))
}

// Search godoc
// @Summary 搜尋用戶
// @Description 根據用戶名或顯示名稱搜尋用戶
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param q query string true "搜尋關鍵字"
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.ProfileResponse}
// @Failure 400 {object} response.Response
// @Router /api/v1/users/search [get]
func (h *UserHandler) Search(c *gin.Context) {
	var req request.SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	profiles, err := h.userService.Search(c.Request.Context(), req.Query, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	profileResponses := make([]*response.ProfileResponse, len(profiles))
	for i, p := range profiles {
		profileResponses[i] = response.NewProfileResponse(p)
	}

	response.Success(c, profileResponses)
}

// BlockUser godoc
// @Summary 封鎖用戶
// @Description 封鎖指定用戶
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/users/{id}/block [post]
func (h *UserHandler) BlockUser(c *gin.Context) {
	blockedID := c.Param("id")
	blockerID := middleware.GetUserID(c)

	if !utils.ValidateUUID(blockedID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	if err := h.userService.BlockUser(c.Request.Context(), blockerID, blockedID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "用戶已封鎖", nil)
}

// UnblockUser godoc
// @Summary 解除封鎖用戶
// @Description 解除封鎖指定用戶
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/users/{id}/unblock [post]
func (h *UserHandler) UnblockUser(c *gin.Context) {
	blockedID := c.Param("id")
	blockerID := middleware.GetUserID(c)

	if !utils.ValidateUUID(blockedID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	if err := h.userService.UnblockUser(c.Request.Context(), blockerID, blockedID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "已解除封鎖", nil)
}

// ListBlockedUsers godoc
// @Summary 獲取封鎖列表
// @Description 獲取當前用戶封鎖的用戶列表
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.ProfileResponse}
// @Router /api/v1/users/blocked [get]
func (h *UserHandler) ListBlockedUsers(c *gin.Context) {
	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 20}
	}

	userID := middleware.GetUserID(c)

	profiles, err := h.userService.ListBlockedUsers(c.Request.Context(), userID, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	profileResponses := make([]*response.ProfileResponse, len(profiles))
	for i, p := range profiles {
		profileResponses[i] = response.NewProfileResponse(p)
	}

	response.Success(c, profileResponses)
}

// SendFriendRequest godoc
// @Summary 發送好友請求
// @Description 向指定用戶發送好友請求
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/users/{id}/friend-request [post]
func (h *UserHandler) SendFriendRequest(c *gin.Context) {
	friendID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(friendID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	if err := h.userService.SendFriendRequest(c.Request.Context(), userID, friendID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "好友請求已發送", nil)
}

// AcceptFriendRequest godoc
// @Summary 接受好友請求
// @Description 接受來自指定用戶的好友請求
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/users/{id}/friend-request/accept [post]
func (h *UserHandler) AcceptFriendRequest(c *gin.Context) {
	friendID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(friendID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	if err := h.userService.AcceptFriendRequest(c.Request.Context(), userID, friendID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "已接受好友請求", nil)
}

// RejectFriendRequest godoc
// @Summary 拒絕好友請求
// @Description 拒絕來自指定用戶的好友請求
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/users/{id}/friend-request/reject [post]
func (h *UserHandler) RejectFriendRequest(c *gin.Context) {
	friendID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(friendID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	if err := h.userService.RejectFriendRequest(c.Request.Context(), userID, friendID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "已拒絕好友請求", nil)
}

// RemoveFriend godoc
// @Summary 移除好友
// @Description 移除指定好友
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/users/{id}/friend [delete]
func (h *UserHandler) RemoveFriend(c *gin.Context) {
	friendID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(friendID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	if err := h.userService.RemoveFriend(c.Request.Context(), userID, friendID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "好友已移除", nil)
}

// ListFriends godoc
// @Summary 獲取好友列表
// @Description 獲取當前用戶的好友列表
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.FriendResponse}
// @Router /api/v1/users/friends [get]
func (h *UserHandler) ListFriends(c *gin.Context) {
	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 20}
	}

	userID := middleware.GetUserID(c)

	friends, err := h.userService.ListFriends(c.Request.Context(), userID, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	friendResponses := make([]*response.FriendResponse, len(friends))
	for i, f := range friends {
		friendResponses[i] = response.NewFriendResponse(f)
	}

	response.Success(c, friendResponses)
}

// ListPendingRequests godoc
// @Summary 獲取待處理的好友請求
// @Description 獲取收到的待處理好友請求
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.FriendRequestResponse}
// @Router /api/v1/users/friend-requests/pending [get]
func (h *UserHandler) ListPendingRequests(c *gin.Context) {
	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 20}
	}

	userID := middleware.GetUserID(c)

	requests, err := h.userService.ListPendingRequests(c.Request.Context(), userID, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	requestResponses := make([]*response.FriendRequestResponse, len(requests))
	for i, r := range requests {
		requestResponses[i] = response.NewFriendRequestResponse(r)
	}

	response.Success(c, requestResponses)
}

// ListSentRequests godoc
// @Summary 獲取已發送的好友請求
// @Description 獲取已發送的待處理好友請求
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.FriendRequestResponse}
// @Router /api/v1/users/friend-requests/sent [get]
func (h *UserHandler) ListSentRequests(c *gin.Context) {
	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 20}
	}

	userID := middleware.GetUserID(c)

	requests, err := h.userService.ListSentRequests(c.Request.Context(), userID, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	requestResponses := make([]*response.FriendRequestResponse, len(requests))
	for i, r := range requests {
		requestResponses[i] = response.NewFriendRequestResponse(r)
	}

	response.Success(c, requestResponses)
}

// GetOnlineUsers godoc
// @Summary 獲取在線用戶
// @Description 獲取當前在線的用戶列表
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.ProfileResponse}
// @Router /api/v1/users/online [get]
func (h *UserHandler) GetOnlineUsers(c *gin.Context) {
	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 20}
	}

	profiles, err := h.userService.GetOnlineUsers(c.Request.Context(), req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	profileResponses := make([]*response.ProfileResponse, len(profiles))
	for i, p := range profiles {
		profileResponses[i] = response.NewProfileResponse(p)
	}

	response.Success(c, profileResponses)
}
