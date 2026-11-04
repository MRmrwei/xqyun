package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// UnmatchedRoute 是没有匹配到路由的请求统一使用的标签值。
// 绝不能用原始路径当标签：/api/v1/admin/submissions/123 这类会让标签基数爆炸，
// 最终把 Prometheus 内存打穿。
const UnmatchedRoute = "unmatched"

var (
	// 指标定义见 docs/监控方案.md 4.1。
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "xqyun_http_requests_total",
		Help: "按路由、方法与 HTTP 状态码统计的请求数。",
	}, []string{"method", "route", "http_status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "xqyun_http_request_duration_seconds",
		Help:    "按路由统计的请求耗时。",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"route"})

	businessResponses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "xqyun_business_responses_total",
		Help: "按路由与业务码统计的统一响应数量。",
	}, []string{"route", "code"})
)

// ObserveHTTP 记录一次 HTTP 请求。route 必须传路由模板（c.FullPath()），
// 传空时归入 UnmatchedRoute。
func ObserveHTTP(method, route string, status int, seconds float64) {
	if route == "" {
		route = UnmatchedRoute
	}
	httpRequestsTotal.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	httpDuration.WithLabelValues(route).Observe(seconds)
}

// ObserveBusiness 记录一次统一响应的业务码。
// 本项目 HTTP 状态恒为 200，只有这里能拿到真实的业务成功 / 失败。
func ObserveBusiness(route string, code int) {
	if route == "" {
		route = UnmatchedRoute
	}
	businessResponses.WithLabelValues(route, strconv.Itoa(code)).Inc()
}
