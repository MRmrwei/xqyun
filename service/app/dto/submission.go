package dto

// AdminSubmission 是后台表单留言列表项 / 详情。
type AdminSubmission struct {
	ID        string         `json:"id"`
	FormKey   string         `json:"form_key"`
	Locale    string         `json:"locale"`
	Name      string         `json:"name"`
	Email     string         `json:"email"`
	Phone     string         `json:"phone"`
	Company   string         `json:"company"`
	Message   string         `json:"message"`
	Data      map[string]any `json:"data"`
	Status    string         `json:"status"`
	IP        string         `json:"ip"`
	UserAgent string         `json:"user_agent"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

// AdminSubmissionStats 是各处理状态的留言数量，便于后台顶部统计卡片。
type AdminSubmissionStats struct {
	Total      int64 `json:"total"`
	New        int64 `json:"new"`
	Processing int64 `json:"processing"`
	Done       int64 `json:"done"`
	Spam       int64 `json:"spam"`
}

// AdminSubmissionListResponse 是表单留言分页响应。
type AdminSubmissionListResponse struct {
	Items    []AdminSubmission    `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Stats    AdminSubmissionStats `json:"stats"`
}

// AdminSubmissionUnreadCountResponse 是待处理留言数量，供后台侧边栏红点轮询。
type AdminSubmissionUnreadCountResponse struct {
	Count int64 `json:"count"`
}

// AdminSubmissionStatusRequest 是标记留言处理状态请求。
type AdminSubmissionStatusRequest struct {
	Status string `json:"status"`
}
