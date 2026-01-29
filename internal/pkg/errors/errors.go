package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status code
type AppError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Err     error       `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// Common errors
var (
	// 400 Bad Request
	ErrBadRequest = New(http.StatusBadRequest, "請求格式錯誤")
	ErrValidation = New(http.StatusBadRequest, "驗證失敗")

	// 401 Unauthorized
	ErrUnauthorized    = New(http.StatusUnauthorized, "未授權的請求")
	ErrInvalidToken    = New(http.StatusUnauthorized, "無效的 Token")
	ErrTokenExpired    = New(http.StatusUnauthorized, "Token 已過期")
	ErrInvalidPassword = New(http.StatusUnauthorized, "密碼錯誤")

	// 403 Forbidden
	ErrForbidden        = New(http.StatusForbidden, "禁止存取")
	ErrPermissionDenied = New(http.StatusForbidden, "權限不足")

	// 404 Not Found
	ErrNotFound     = New(http.StatusNotFound, "資源不存在")
	ErrUserNotFound = New(http.StatusNotFound, "用戶不存在")
	ErrRoomNotFound = New(http.StatusNotFound, "聊天室不存在")

	// 409 Conflict
	ErrConflict           = New(http.StatusConflict, "資源衝突")
	ErrUsernameExists     = New(http.StatusConflict, "使用者名稱已存在")
	ErrEmailExists        = New(http.StatusConflict, "電子郵件已存在")
	ErrAlreadyRoomMember  = New(http.StatusConflict, "已經是聊天室成員")
	ErrAlreadyFriend      = New(http.StatusConflict, "已經是好友")
	ErrAlreadyBlocked     = New(http.StatusConflict, "已經封鎖該用戶")
	ErrFriendRequestSent  = New(http.StatusConflict, "已發送好友請求")

	// 422 Unprocessable Entity
	ErrRoomFull         = New(http.StatusUnprocessableEntity, "聊天室已滿")
	ErrCannotBlockSelf  = New(http.StatusUnprocessableEntity, "無法封鎖自己")
	ErrCannotMessageSelf = New(http.StatusUnprocessableEntity, "無法給自己發送訊息")
	ErrUserBlocked      = New(http.StatusUnprocessableEntity, "您已被該用戶封鎖")

	// 429 Too Many Requests
	ErrTooManyRequests = New(http.StatusTooManyRequests, "請求過於頻繁，請稍後再試")

	// 500 Internal Server Error
	ErrInternal = New(http.StatusInternalServerError, "伺服器內部錯誤")
)

// Is checks if an error is of a specific type
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// GetHTTPStatus returns the HTTP status code for an error
func GetHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return http.StatusInternalServerError
}

// GetMessage returns the error message
func GetMessage(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return "伺服器內部錯誤"
}
