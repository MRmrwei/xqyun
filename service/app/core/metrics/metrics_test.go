package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("读取指标失败: %v", err)
	}
	return m.GetCounter().GetValue()
}

// 空路由（没匹配到路由的请求）必须归入 UnmatchedRoute，不能写入空标签。
func TestObserveHTTPFallsBackToUnmatchedRoute(t *testing.T) {
	counter := httpRequestsTotal.WithLabelValues("GET", UnmatchedRoute, "404")
	before := counterValue(t, counter)

	ObserveHTTP("GET", "", 404, 0.01)

	if got := counterValue(t, counter); got != before+1 {
		t.Fatalf("UnmatchedRoute 计数 = %v，期望 %v", got, before+1)
	}
}

// 动态路由必须用模板，同一模板的不同请求累计到同一个标签。
func TestObserveHTTPCountsByRouteTemplate(t *testing.T) {
	const route = "/api/v1/admin/submissions/:id"
	counter := httpRequestsTotal.WithLabelValues("GET", route, "200")
	before := counterValue(t, counter)

	ObserveHTTP("GET", route, 200, 0.02)
	ObserveHTTP("GET", route, 200, 0.03)

	if got := counterValue(t, counter); got != before+2 {
		t.Fatalf("路由模板计数 = %v，期望 %v", got, before+2)
	}
}

// 业务码要单独计数，失败请求也算在对应 code 下。
func TestObserveBusinessCountsByCode(t *testing.T) {
	const route = "/api/v1/admin/auth/login"
	okCounter := businessResponses.WithLabelValues(route, "0")
	failCounter := businessResponses.WithLabelValues(route, "40100")
	okBefore := counterValue(t, okCounter)
	failBefore := counterValue(t, failCounter)

	ObserveBusiness(route, 0)
	ObserveBusiness(route, 40100)
	ObserveBusiness(route, 40100)

	if got := counterValue(t, okCounter); got != okBefore+1 {
		t.Fatalf("成功计数 = %v，期望 %v", got, okBefore+1)
	}
	if got := counterValue(t, failCounter); got != failBefore+2 {
		t.Fatalf("失败计数 = %v，期望 %v", got, failBefore+2)
	}
}
