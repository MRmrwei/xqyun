package middleware

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"

	"suki/xqyun/app/core/response"
)

// metricValue 从默认 registry 里取一条 counter 的值；没有该样本时返回 0。
func metricValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("读取指标失败: %v", err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, m := range family.GetMetric() {
			if sameLabels(m.GetLabel(), labels) && m.GetCounter() != nil {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func sameLabels(pairs []*dto.LabelPair, want map[string]string) bool {
	if len(pairs) != len(want) {
		return false
	}
	for _, pair := range pairs {
		if want[pair.GetName()] != pair.GetValue() {
			return false
		}
	}
	return true
}

// 动态路由必须记录成路由模板，而不是带具体 ID 的原始路径。
func TestHTTPMetricsUsesRouteTemplate(t *testing.T) {
	h := server.New(server.WithHostPorts(":0"))
	h.Use(NewHTTPMetrics())
	h.GET("/api/v1/admin/submissions/:id", func(ctx context.Context, c *app.RequestContext) {
		response.Success(c, "ok")
	})

	httpLabels := map[string]string{
		"method":      "GET",
		"route":       "/api/v1/admin/submissions/:id",
		"http_status": "200",
	}
	businessLabels := map[string]string{
		"route": "/api/v1/admin/submissions/:id",
		"code":  "0",
	}
	httpBefore := metricValue(t, "xqyun_http_requests_total", httpLabels)
	businessBefore := metricValue(t, "xqyun_business_responses_total", businessLabels)

	ut.PerformRequest(h.Engine, "GET", "/api/v1/admin/submissions/123", nil)

	if got := metricValue(t, "xqyun_http_requests_total", httpLabels); got != httpBefore+1 {
		t.Fatalf("路由模板 HTTP 计数 = %v，期望 %v；动态路由可能被记成了原始路径", got, httpBefore+1)
	}
	if got := metricValue(t, "xqyun_business_responses_total", businessLabels); got != businessBefore+1 {
		t.Fatalf("业务码计数 = %v，期望 %v", got, businessBefore+1)
	}

	rawLabels := map[string]string{
		"method":      "GET",
		"route":       "/api/v1/admin/submissions/123",
		"http_status": "200",
	}
	if got := metricValue(t, "xqyun_http_requests_total", rawLabels); got != 0 {
		t.Fatalf("出现原始路径标签 %v，会导致标签基数爆炸", rawLabels)
	}
}

// /metrics 在 r.Use 之前注册，因此不经过指标中间件：
// 既不会每 15 秒被 Prometheus 抓取时产生一条访问日志，也不会统计自己。
// 该用例必须走真实监听：promhttp 会用 chunked writer，ut.PerformRequest 提供的上下文没有连接。
func TestMetricsEndpointSkipsInstrumentation(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer ln.Close()

	h := server.New(server.WithListener(ln))
	h.GET("/metrics", adaptor.HertzHandler(promhttp.Handler()))
	h.Use(NewHTTPMetrics())
	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "pong")
	})

	go h.Spin()
	defer h.Close()

	base := "http://" + ln.Addr().String()
	// 等待监听就绪：先打一次业务请求，同时保证指标里有样本
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := http.Get(base + "/ping")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("服务未就绪: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	metricsLabels := map[string]string{"method": "GET", "route": "/metrics", "http_status": "200"}
	before := metricValue(t, "xqyun_http_requests_total", metricsLabels)

	resp, err := http.Get(base + "/metrics")
	if err != nil {
		t.Fatalf("请求 /metrics 失败: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取 /metrics 响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/metrics 状态码 = %d，期望 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), "xqyun_http_requests_total") {
		t.Fatalf("/metrics 未输出 xqyun_http_requests_total，实际长度 %d", len(body))
	}

	if after := metricValue(t, "xqyun_http_requests_total", metricsLabels); after != before {
		t.Fatalf("/metrics 被指标中间件统计了（%v → %v）", before, after)
	}
}
