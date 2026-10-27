package middleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestShouldLog(t *testing.T) {
	cases := []struct {
		name       string
		fullPath   string
		clientTag  string
		status     int
		wantLogged bool
	}{
		{name: "健康检查不记录", fullPath: "/ping", status: http.StatusOK, wantLogged: false},
		{name: "SSR 成功请求不记录", fullPath: "/api/v1/frontend/render/page", clientTag: ssrClientValue, status: http.StatusOK, wantLogged: false},
		{name: "SSR 失败请求保留", fullPath: "/api/v1/frontend/render/page", clientTag: ssrClientValue, status: http.StatusInternalServerError, wantLogged: true},
		{name: "SSR 重定向不记录", fullPath: "/api/v1/frontend/render/page", clientTag: ssrClientValue, status: http.StatusMovedPermanently, wantLogged: false},
		{name: "无标记的前台请求记录", fullPath: "/api/v1/frontend/render/page", status: http.StatusOK, wantLogged: true},
		{name: "标记值不匹配仍记录", fullPath: "/api/v1/frontend/render/page", clientTag: "browser", status: http.StatusOK, wantLogged: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			c := app.NewContext(1)
			c.SetFullPath(testCase.fullPath)
			c.SetStatusCode(testCase.status)
			if testCase.clientTag != "" {
				c.Request.Header.Set(ssrClientHeader, testCase.clientTag)
			}

			if got := shouldLog(context.Background(), c); got != testCase.wantLogged {
				t.Fatalf("shouldLog() = %v，期望 %v", got, testCase.wantLogged)
			}
		})
	}
}
