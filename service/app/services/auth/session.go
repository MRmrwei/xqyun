package auth

// SessionStore 是当前请求登录会话的读写能力。
// HTTP 层用 hertz-contrib/sessions 适配实现，service 层只依赖这个接口，
// 不感知 Cookie、请求上下文与具体存储后端。
type SessionStore interface {
	// Set 写入会话值；需调用 Save 后才会持久化。
	Set(key string, value any)

	// Get 读取会话值，不存在时第二个返回值为 false。
	Get(key string) (any, bool)

	// Clear 清空会话并标记为删除；需调用 Save 后才会删除 Cookie 与服务端记录。
	Clear()

	// Save 持久化会话：普通写入会下发 / 刷新 Cookie，清空后则删除会话。
	Save() error
}

// 管理后台 Session 模式使用的会话键。
const (
	// SessionKeyAdminUserID 保存管理员 ID（xqyun_users.id）。
	SessionKeyAdminUserID = "admin_user_id"
	// SessionKeyAdminUsername 保存管理员登录名，用于日志与展示。
	SessionKeyAdminUsername = "admin_username"
)
