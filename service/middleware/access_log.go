package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/logger/accesslog"
)

// ssrClientHeader 由官网 SSR 请求携带，值为 ssrClientValue。
const (
	ssrClientHeader = "X-Xqyun-Client"
	ssrClientValue  = "ssr"
)

// NewAccessLog 返回 HTTP 访问日志中间件：时间、状态、耗时、客户端 IP、方法、路径与 query。
// 不要加入 ${body}、Authorization 或 Cookie，登录请求体包含密码。
func NewAccessLog() app.HandlerFunc {
	return accesslog.New(
		accesslog.WithFormat("[${time}] ${status} - ${latency} ${clientIP} ${method} ${path} ${queryParams}"),
		accesslog.WithTimeFormat("2006-01-02 15:04:05"),
		accesslog.WithAccessLogFunc(hlog.CtxInfof),
		accesslog.WithLogConditionFunc(shouldLog),
	)
}

// shouldLog 在响应返回后判断是否写访问日志。
//   - /ping 由健康检查每 10 秒调用一次，不记录。
//   - 官网 SSR 每打开一个页面都会请求 site/config、site/nav、render/page，
//     次数远高于人工访问，成功（<400）时不记录，失败仍保留。
//     注意统一响应下业务错误也是 HTTP 200，这类问题要看 handler 的 hlog 错误日志。
func shouldLog(_ context.Context, c *app.RequestContext) bool {
	if c.FullPath() == "/ping" {
		return false
	}
	silent := c.Request.Header.Get(ssrClientHeader) == ssrClientValue &&
		c.Response.StatusCode() < 400
	return !silent
}
