package utils

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validator provides validation methods
type Validator struct {
	errors ValidationErrors
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, &ValidationError{
		Field:   field,
		Message: message,
	})
}

// Errors returns all validation errors
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// HasErrors returns true if there are any validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Required checks if a string is not empty
func (v *Validator) Required(field, value string) bool {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "此欄位為必填")
		return false
	}
	return true
}

// MinLength checks if a string has minimum length
func (v *Validator) MinLength(field, value string, min int) bool {
	if utf8.RuneCountInString(value) < min {
		v.AddError(field, "長度至少需要 "+string(rune('0'+min))+" 個字元")
		return false
	}
	return true
}

// MaxLength checks if a string doesn't exceed maximum length
func (v *Validator) MaxLength(field, value string, max int) bool {
	if utf8.RuneCountInString(value) > max {
		v.AddError(field, "長度不能超過 "+string(rune('0'+max))+" 個字元")
		return false
	}
	return true
}

// ValidateUsername validates a username
func (v *Validator) ValidateUsername(field, value string) bool {
	if !v.Required(field, value) {
		return false
	}
	if !usernameRegex.MatchString(value) {
		v.AddError(field, "使用者名稱只能包含字母、數字、底線和連字符，長度 3-50 字元")
		return false
	}
	return true
}

// ValidateEmail validates an email address
func (v *Validator) ValidateEmail(field, value string) bool {
	if !v.Required(field, value) {
		return false
	}
	if !emailRegex.MatchString(value) {
		v.AddError(field, "請輸入有效的電子郵件地址")
		return false
	}
	return true
}

// ValidatePassword validates a password
func (v *Validator) ValidatePassword(field, value string) bool {
	if !v.Required(field, value) {
		return false
	}
	if len(value) < 8 {
		v.AddError(field, "密碼長度至少需要 8 個字元")
		return false
	}
	if len(value) > 72 {
		v.AddError(field, "密碼長度不能超過 72 個字元")
		return false
	}
	return true
}

// ValidateRoomName validates a room name
func (v *Validator) ValidateRoomName(field, value string) bool {
	if !v.Required(field, value) {
		return false
	}
	length := utf8.RuneCountInString(value)
	if length < 2 {
		v.AddError(field, "聊天室名稱至少需要 2 個字元")
		return false
	}
	if length > 100 {
		v.AddError(field, "聊天室名稱不能超過 100 個字元")
		return false
	}
	return true
}

// ValidateMessageContent validates message content
func (v *Validator) ValidateMessageContent(field, value string) bool {
	if !v.Required(field, value) {
		return false
	}
	if utf8.RuneCountInString(value) > 5000 {
		v.AddError(field, "訊息內容不能超過 5000 個字元")
		return false
	}
	return true
}

// ValidateUUID validates a UUID string
func ValidateUUID(s string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(s)
}

// SanitizeString removes potentially dangerous characters
func SanitizeString(s string) string {
	// Remove null bytes and control characters
	s = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}
