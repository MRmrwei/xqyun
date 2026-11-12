package dto

// AdminLoginRequest 是管理后台登录请求。
type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AdminLoginResponse 是管理后台登录响应。
// 会话票据由后端写入 HttpOnly Cookie，响应体只返回有效期与用户信息。
type AdminLoginResponse struct {
	ExpiresIn int       `json:"expires_in"`
	User      AdminUser `json:"user"`
}

// AdminUser 是管理后台登录用户信息。
type AdminUser struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Mobile      string   `json:"mobile"`
	Avatar      string   `json:"avatar"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}
