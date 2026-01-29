package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/dto/request"
	"github.com/go-demo/chat/internal/dto/response"
	"github.com/go-demo/chat/internal/middleware"
	"github.com/go-demo/chat/internal/model"
	"github.com/go-demo/chat/internal/pkg/utils"
	"github.com/go-demo/chat/internal/service"
)

type RoomHandler struct {
	roomService *service.RoomService
}

func NewRoomHandler(roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{
		roomService: roomService,
	}
}

// Create godoc
// @Summary 創建聊天室
// @Description 創建新的聊天室
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CreateRoomRequest true "聊天室資料"
// @Success 201 {object} response.Response{data=response.RoomDetailResponse}
// @Failure 400 {object} response.Response
// @Router /api/v1/rooms [post]
func (h *RoomHandler) Create(c *gin.Context) {
	var req request.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	userID := middleware.GetUserID(c)

	// Validate room name
	v := utils.NewValidator()
	v.ValidateRoomName("name", req.Name)
	if v.HasErrors() {
		response.ValidationError(c, v.Errors())
		return
	}

	// Default type
	roomType := model.RoomTypePublic
	if req.Type == "private" {
		roomType = model.RoomTypePrivate
	}

	room, err := h.roomService.Create(c.Request.Context(), &service.CreateRoomInput{
		Name:        req.Name,
		Description: req.Description,
		Type:        roomType,
		OwnerID:     userID,
		MaxMembers:  req.MaxMembers,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	detail, err := h.roomService.GetByIDWithDetails(c.Request.Context(), room.ID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, response.NewRoomDetailResponse(detail))
}

// GetByID godoc
// @Summary 獲取聊天室詳情
// @Description 獲取指定聊天室的詳細資訊
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Success 200 {object} response.Response{data=response.RoomDetailResponse}
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{id} [get]
func (h *RoomHandler) GetByID(c *gin.Context) {
	roomID := c.Param("id")

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	detail, err := h.roomService.GetByIDWithDetails(c.Request.Context(), roomID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, response.NewRoomDetailResponse(detail))
}

// Update godoc
// @Summary 更新聊天室
// @Description 更新聊天室資訊（需要管理員權限）
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Param request body request.UpdateRoomRequest true "更新資料"
// @Success 200 {object} response.Response{data=response.RoomDetailResponse}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{id} [put]
func (h *RoomHandler) Update(c *gin.Context) {
	roomID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	var req request.UpdateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	_, err := h.roomService.Update(c.Request.Context(), &service.UpdateRoomInput{
		RoomID:      roomID,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		MaxMembers:  req.MaxMembers,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	detail, err := h.roomService.GetByIDWithDetails(c.Request.Context(), roomID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, response.NewRoomDetailResponse(detail))
}

// Delete godoc
// @Summary 刪除聊天室
// @Description 刪除聊天室（僅房主可操作）
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Success 204
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{id} [delete]
func (h *RoomHandler) Delete(c *gin.Context) {
	roomID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	if err := h.roomService.Delete(c.Request.Context(), roomID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

// ListPublic godoc
// @Summary 獲取公開聊天室列表
// @Description 獲取所有公開的聊天室
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.RoomResponse}
// @Router /api/v1/rooms [get]
func (h *RoomHandler) ListPublic(c *gin.Context) {
	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 20}
	}

	rooms, err := h.roomService.ListPublic(c.Request.Context(), req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	roomResponses := make([]*response.RoomResponse, len(rooms))
	for i, r := range rooms {
		roomResponses[i] = response.NewRoomResponse(r)
	}

	response.Success(c, roomResponses)
}

// ListMyRooms godoc
// @Summary 獲取我的聊天室
// @Description 獲取當前用戶加入的聊天室
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.RoomResponse}
// @Router /api/v1/rooms/me [get]
func (h *RoomHandler) ListMyRooms(c *gin.Context) {
	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 20}
	}

	userID := middleware.GetUserID(c)

	rooms, err := h.roomService.ListByUserID(c.Request.Context(), userID, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	roomResponses := make([]*response.RoomResponse, len(rooms))
	for i, r := range rooms {
		roomResponses[i] = response.NewRoomResponse(r)
	}

	response.Success(c, roomResponses)
}

// Search godoc
// @Summary 搜尋聊天室
// @Description 根據名稱搜尋公開聊天室
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param q query string true "搜尋關鍵字"
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.RoomResponse}
// @Failure 400 {object} response.Response
// @Router /api/v1/rooms/search [get]
func (h *RoomHandler) Search(c *gin.Context) {
	var req request.SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	rooms, err := h.roomService.Search(c.Request.Context(), req.Query, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	roomResponses := make([]*response.RoomResponse, len(rooms))
	for i, r := range rooms {
		roomResponses[i] = response.NewRoomResponse(r)
	}

	response.Success(c, roomResponses)
}

// Join godoc
// @Summary 加入聊天室
// @Description 加入公開聊天室
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Success 200 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 422 {object} response.Response
// @Router /api/v1/rooms/{id}/join [post]
func (h *RoomHandler) Join(c *gin.Context) {
	roomID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	if err := h.roomService.Join(c.Request.Context(), roomID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "已加入聊天室", nil)
}

// Leave godoc
// @Summary 離開聊天室
// @Description 離開聊天室
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{id}/leave [post]
func (h *RoomHandler) Leave(c *gin.Context) {
	roomID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	if err := h.roomService.Leave(c.Request.Context(), roomID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "已離開聊天室", nil)
}

// InviteMember godoc
// @Summary 邀請成員
// @Description 邀請用戶加入私人聊天室（需要管理員權限）
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Param request body request.InviteMemberRequest true "邀請資料"
// @Success 200 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/v1/rooms/{id}/invite [post]
func (h *RoomHandler) InviteMember(c *gin.Context) {
	roomID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	var req request.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	if err := h.roomService.InviteMember(c.Request.Context(), roomID, userID, req.UserID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "已邀請用戶", nil)
}

// KickMember godoc
// @Summary 踢出成員
// @Description 將成員踢出聊天室（需要管理員權限）
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Param user_id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{id}/members/{user_id}/kick [post]
func (h *RoomHandler) KickMember(c *gin.Context) {
	roomID := c.Param("id")
	targetID := c.Param("user_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) || !utils.ValidateUUID(targetID) {
		response.BadRequest(c, "無效的 ID")
		return
	}

	if err := h.roomService.KickMember(c.Request.Context(), roomID, userID, targetID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "成員已被踢出", nil)
}

// ListMembers godoc
// @Summary 獲取成員列表
// @Description 獲取聊天室成員列表
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Success 200 {object} response.Response{data=[]response.RoomMemberResponse}
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{id}/members [get]
func (h *RoomHandler) ListMembers(c *gin.Context) {
	roomID := c.Param("id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	members, err := h.roomService.ListMembers(c.Request.Context(), roomID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	memberResponses := make([]*response.RoomMemberResponse, len(members))
	for i, m := range members {
		memberResponses[i] = response.NewRoomMemberResponse(m)
	}

	response.Success(c, memberResponses)
}

// PromoteMember godoc
// @Summary 提升成員為管理員
// @Description 將成員提升為管理員（僅房主可操作）
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Param user_id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{id}/members/{user_id}/promote [post]
func (h *RoomHandler) PromoteMember(c *gin.Context) {
	roomID := c.Param("id")
	targetID := c.Param("user_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) || !utils.ValidateUUID(targetID) {
		response.BadRequest(c, "無效的 ID")
		return
	}

	if err := h.roomService.PromoteMember(c.Request.Context(), roomID, userID, targetID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "成員已被提升為管理員", nil)
}

// DemoteMember godoc
// @Summary 降級管理員為成員
// @Description 將管理員降級為普通成員（僅房主可操作）
// @Tags 聊天室
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "聊天室 ID"
// @Param user_id path string true "用戶 ID"
// @Success 200 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{id}/members/{user_id}/demote [post]
func (h *RoomHandler) DemoteMember(c *gin.Context) {
	roomID := c.Param("id")
	targetID := c.Param("user_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) || !utils.ValidateUUID(targetID) {
		response.BadRequest(c, "無效的 ID")
		return
	}

	if err := h.roomService.DemoteMember(c.Request.Context(), roomID, userID, targetID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "管理員已被降級為成員", nil)
}
