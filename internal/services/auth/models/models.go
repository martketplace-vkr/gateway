package models

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	InviteToken string `json:"invite_token"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthMeResponse struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type VendorResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type VendorsResponse struct {
	Vendors []VendorResponse `json:"vendors"`
}

type UpdateClientStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type ClientModerationEvent struct {
	ID        int64  `json:"id"`
	AdminID   int64  `json:"admin_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

type ClientProfile struct {
	ID               int64                   `json:"id"`
	Email            string                  `json:"email"`
	FirstName        string                  `json:"first_name,omitempty"`
	LastName         string                  `json:"last_name,omitempty"`
	AvatarURL        string                  `json:"avatar_url,omitempty"`
	EmailVerified    bool                    `json:"email_verified"`
	Status           string                  `json:"status"`
	StatusReason     string                  `json:"status_reason,omitempty"`
	CreatedAt        string                  `json:"created_at"`
	UpdatedAt        string                  `json:"updated_at,omitempty"`
	LastActivityAt   string                  `json:"last_activity_at,omitempty"`
	Addresses        any                     `json:"addresses,omitempty"`
	ModerationEvents []ClientModerationEvent `json:"moderation_events,omitempty"`
}

type ClientsResponse struct {
	Clients []ClientProfile `json:"clients"`
	Total   uint64          `json:"total"`
}
