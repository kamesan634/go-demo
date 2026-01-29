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

type MessageHandler struct {
	messageService *service.MessageService
	roomService    *service.RoomService
	dmService      *service.DirectMessageService
}

func NewMessageHandler(
	messageService *service.MessageService,
	roomService *service.RoomService,
	dmService *service.DirectMessageService,
) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		roomService:    roomService,
		dmService:      dmService,
	}
}

// SendMessage godoc
// @Summary 發送訊息
// @Description 在聊天室中發送訊息
// @Tags 訊息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param room_id path string true "聊天室 ID"
// @Param request body request.SendMessageRequest true "訊息內容"
// @Success 201 {object} response.Response{data=response.MessageResponse}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{room_id}/messages [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	roomID := c.Param("room_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	var req request.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	// Validate content
	v := utils.NewValidator()
	v.ValidateMessageContent("content", req.Content)
	if v.HasErrors() {
		response.ValidationError(c, v.Errors())
		return
	}

	// Default type
	msgType := model.MessageTypeText
	if req.Type == "image" {
		msgType = model.MessageTypeImage
	} else if req.Type == "file" {
		msgType = model.MessageTypeFile
	}

	msg, err := h.messageService.SendMessage(c.Request.Context(), &service.SendMessageInput{
		RoomID:    roomID,
		UserID:    userID,
		Content:   req.Content,
		Type:      msgType,
		ReplyToID: req.ReplyToID,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, response.NewMessageResponse(msg))
}

// GetMessages godoc
// @Summary 獲取訊息列表
// @Description 獲取聊天室的訊息列表
// @Tags 訊息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param room_id path string true "聊天室 ID"
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(50)
// @Success 200 {object} response.Response{data=[]response.MessageResponse}
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{room_id}/messages [get]
func (h *MessageHandler) GetMessages(c *gin.Context) {
	roomID := c.Param("room_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 50}
	}

	messages, err := h.messageService.ListByRoomID(c.Request.Context(), roomID, userID, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	messageResponses := make([]*response.MessageResponse, len(messages))
	for i, m := range messages {
		messageResponses[i] = response.NewMessageResponse(m)
	}

	response.Success(c, messageResponses)
}

// UpdateMessage godoc
// @Summary 編輯訊息
// @Description 編輯已發送的訊息
// @Tags 訊息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param room_id path string true "聊天室 ID"
// @Param message_id path string true "訊息 ID"
// @Param request body request.UpdateMessageRequest true "更新內容"
// @Success 200 {object} response.Response{data=response.MessageResponse}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{room_id}/messages/{message_id} [put]
func (h *MessageHandler) UpdateMessage(c *gin.Context) {
	messageID := c.Param("message_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(messageID) {
		response.BadRequest(c, "無效的訊息 ID")
		return
	}

	var req request.UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	msg, err := h.messageService.UpdateMessage(c.Request.Context(), messageID, userID, req.Content)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, response.NewMessageResponse(msg))
}

// DeleteMessage godoc
// @Summary 刪除訊息
// @Description 刪除訊息（自己的訊息或管理員可刪除）
// @Tags 訊息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param room_id path string true "聊天室 ID"
// @Param message_id path string true "訊息 ID"
// @Success 204
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{room_id}/messages/{message_id} [delete]
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	messageID := c.Param("message_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(messageID) {
		response.BadRequest(c, "無效的訊息 ID")
		return
	}

	if err := h.messageService.DeleteMessage(c.Request.Context(), messageID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContent(c)
}

// SearchMessages godoc
// @Summary 搜尋訊息
// @Description 在聊天室中搜尋訊息
// @Tags 訊息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param room_id path string true "聊天室 ID"
// @Param q query string true "搜尋關鍵字"
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.MessageResponse}
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/v1/rooms/{room_id}/messages/search [get]
func (h *MessageHandler) SearchMessages(c *gin.Context) {
	roomID := c.Param("room_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	var req request.SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	messages, err := h.messageService.Search(c.Request.Context(), roomID, userID, req.Query, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	messageResponses := make([]*response.MessageResponse, len(messages))
	for i, m := range messages {
		messageResponses[i] = response.NewMessageResponse(m)
	}

	response.Success(c, messageResponses)
}

// MarkAsRead godoc
// @Summary 標記已讀
// @Description 標記聊天室訊息為已讀
// @Tags 訊息
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param room_id path string true "聊天室 ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/rooms/{room_id}/messages/read [post]
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	roomID := c.Param("room_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(roomID) {
		response.BadRequest(c, "無效的聊天室 ID")
		return
	}

	if err := h.roomService.UpdateLastRead(c.Request.Context(), roomID, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "已標記為已讀", nil)
}

// SendDirectMessage godoc
// @Summary 發送私訊
// @Description 向指定用戶發送私人訊息
// @Tags 私訊
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "接收者 ID"
// @Param request body request.SendDirectMessageRequest true "訊息內容"
// @Success 201 {object} response.Response{data=response.DirectMessageResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Router /api/v1/dm/{user_id} [post]
func (h *MessageHandler) SendDirectMessage(c *gin.Context) {
	receiverID := c.Param("user_id")
	senderID := middleware.GetUserID(c)

	if !utils.ValidateUUID(receiverID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	var req request.SendDirectMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "請求格式錯誤")
		return
	}

	// Validate content
	v := utils.NewValidator()
	v.ValidateMessageContent("content", req.Content)
	if v.HasErrors() {
		response.ValidationError(c, v.Errors())
		return
	}

	// Default type
	msgType := model.MessageTypeText
	if req.Type == "image" {
		msgType = model.MessageTypeImage
	} else if req.Type == "file" {
		msgType = model.MessageTypeFile
	}

	msg, err := h.dmService.SendMessage(c.Request.Context(), &service.SendDMInput{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    req.Content,
		Type:       msgType,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, response.NewDirectMessageResponse(msg))
}

// GetConversation godoc
// @Summary 獲取私訊對話
// @Description 獲取與指定用戶的私訊對話記錄
// @Tags 私訊
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "對方用戶 ID"
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(50)
// @Success 200 {object} response.Response{data=[]response.DirectMessageResponse}
// @Failure 404 {object} response.Response
// @Router /api/v1/dm/{user_id} [get]
func (h *MessageHandler) GetConversation(c *gin.Context) {
	otherUserID := c.Param("user_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(otherUserID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 50}
	}

	messages, err := h.dmService.GetConversation(c.Request.Context(), userID, otherUserID, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	messageResponses := make([]*response.DirectMessageResponse, len(messages))
	for i, m := range messages {
		messageResponses[i] = response.NewDirectMessageResponse(m)
	}

	response.Success(c, messageResponses)
}

// ListConversations godoc
// @Summary 獲取對話列表
// @Description 獲取所有私訊對話
// @Tags 私訊
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "頁碼" default(1)
// @Param limit query int false "每頁數量" default(20)
// @Success 200 {object} response.Response{data=[]response.ConversationResponse}
// @Router /api/v1/dm [get]
func (h *MessageHandler) ListConversations(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req request.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = request.PaginationRequest{Page: 1, Limit: 20}
	}

	conversations, err := h.dmService.ListConversations(c.Request.Context(), userID, req.Limit, req.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}

	conversationResponses := make([]*response.ConversationResponse, len(conversations))
	for i, c := range conversations {
		conversationResponses[i] = response.NewConversationResponse(c)
	}

	response.Success(c, conversationResponses)
}

// MarkDMAsRead godoc
// @Summary 標記私訊已讀
// @Description 標記與指定用戶的私訊為已讀
// @Tags 私訊
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "對方用戶 ID"
// @Success 200 {object} response.Response
// @Router /api/v1/dm/{user_id}/read [post]
func (h *MessageHandler) MarkDMAsRead(c *gin.Context) {
	senderID := c.Param("user_id")
	userID := middleware.GetUserID(c)

	if !utils.ValidateUUID(senderID) {
		response.BadRequest(c, "無效的用戶 ID")
		return
	}

	if err := h.dmService.MarkAsRead(c.Request.Context(), userID, senderID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, "已標記為已讀", nil)
}

// GetUnreadCount godoc
// @Summary 獲取未讀數量
// @Description 獲取未讀私訊數量
// @Tags 私訊
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=map[string]int}
// @Router /api/v1/dm/unread [get]
func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	userID := middleware.GetUserID(c)

	count, err := h.dmService.CountUnread(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"count": count})
}
