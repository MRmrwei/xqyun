package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"suki/xqyun/app/core/metrics"
)

// NewHTTPMetrics 记录每个请求的路由模板、方法、状态码与耗时。
// route 必须用 FullPath() 而不是 Path()，否则动态路由会变成每个 ID 一个标签。
func NewHTTPMetrics() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		c.Next(ctx)
		metrics.ObserveHTTP(
			string(c.Method()),
			c.FullPath(),
			c.Response.StatusCode(),
			time.Since(start).Seconds(),
		)
	}
}
