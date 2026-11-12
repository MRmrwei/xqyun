// Package middleware 提供 Hertz 中间件。
package middleware

import (
	"context"
	"net/url"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	hertzsessions "github.com/hertz-contrib/sessions"

	"suki/xqyun/app/core/response"
	"suki/xqyun/app/services/auth"
)

const adminIdentityKey = "xqyun.admin.identity"

// AdminIdentity 是当前登录管理员的最小身份信息。
type AdminIdentity struct {
	UserID   uint64
	Username string
}

// AdminSessionAuth 校验管理后台的登录会话。
// 必须挂在 hertz-contrib/sessions 中间件之后，否则 Default 会 panic。
type AdminSessionAuth struct{}

// NewAdminSessionAuth 创建会话认证中间件。
func NewAdminSessionAuth() *AdminSessionAuth {
	return &AdminSessionAuth{}
}

// Handle 从会话读取管理员 ID，写入请求上下文后放行；未登录返回 40100。
// 对非安全方法额外校验 Origin，配合 SameSite=Lax 防御 CSRF。
func (m *AdminSessionAuth) Handle(ctx context.Context, c *app.RequestContext) {
	if !isSafeMethod(c.Method()) && !sameOrigin(c) {
		response.Fail(c, response.CodeForbidden, "请求来源不合法")
		c.Abort()
		return
	}
	session := hertzsessions.Default(c)
	value := session.Get(auth.SessionKeyAdminUserID)
	userID, ok := value.(uint64)
	if !ok || userID == 0 {
		response.Fail(c, response.CodeUnauthorized, "请先登录")
		c.Abort()
		return
	}

	username, _ := session.Get(auth.SessionKeyAdminUsername).(string)
	c.Set(adminIdentityKey, &AdminIdentity{UserID: userID, Username: username})
	c.Next(ctx)
}

func isSafeMethod(method []byte) bool {
	switch strings.ToUpper(string(method)) {
	case "GET", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

// sameOrigin 校验 Origin 与当前请求 Host 一致；没有 Origin 时放行（非浏览器客户端）。
func sameOrigin(c *app.RequestContext) bool {
	origin := strings.TrimSpace(string(c.GetHeader("Origin")))
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, string(c.Host()))
}

// AdminIdentityFromContext 读取当前请求的管理员身份。
func AdminIdentityFromContext(c *app.RequestContext) (*AdminIdentity, bool) {
	value, exists := c.Get(adminIdentityKey)
	if !exists {
		return nil, false
	}
	identity, ok := value.(*AdminIdentity)
	return identity, ok
}
