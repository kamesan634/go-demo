package request

// CreateRoomRequest represents a room creation request
type CreateRoomRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description,omitempty" binding:"omitempty,max=500"`
	Type        string `json:"type,omitempty" binding:"omitempty,oneof=public private"` // default: public
	MaxMembers  int    `json:"max_members,omitempty" binding:"omitempty,min=2,max=1000"`
}

// UpdateRoomRequest represents a room update request
type UpdateRoomRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=500"`
	MaxMembers  *int    `json:"max_members,omitempty" binding:"omitempty,min=2,max=1000"`
}

// InviteMemberRequest represents an invite member request
type InviteMemberRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

// UpdateMemberRoleRequest represents a member role update request
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
}
