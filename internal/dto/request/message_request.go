package request

// SendMessageRequest represents a message sending request
type SendMessageRequest struct {
	Content   string `json:"content" binding:"required,max=5000"`
	Type      string `json:"type,omitempty" binding:"omitempty,oneof=text image file"` // default: text
	ReplyToID string `json:"reply_to_id,omitempty" binding:"omitempty,uuid"`
}

// UpdateMessageRequest represents a message update request
type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required,max=5000"`
}

// SendDirectMessageRequest represents a direct message sending request
type SendDirectMessageRequest struct {
	Content string `json:"content" binding:"required,max=5000"`
	Type    string `json:"type,omitempty" binding:"omitempty,oneof=text image file"` // default: text
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page  int `form:"page,default=1" binding:"min=1"`
	Limit int `form:"limit,default=20" binding:"min=1,max=100"`
}

// Offset calculates the offset for database queries
func (p *PaginationRequest) Offset() int {
	return (p.Page - 1) * p.Limit
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query string `form:"q" binding:"required,min=1,max=100"`
	PaginationRequest
}
