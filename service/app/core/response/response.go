package response

import (
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"suki/xqyun/app/core/metrics"
)

// 业务码：0 表示成功，其余按 HTTP 语义分段，便于前端区分。
const (
	CodeOK           = 0
	CodeBadRequest   = 40000
	CodeUnauthorized = 40100
	CodeForbidden    = 40300
	CodeNotFound     = 40400
	CodeConflict     = 40900
	CodeInternal     = 50000
)

// Body 是统一的 JSON 返回结构。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Success 返回成功响应（HTTP 200）。
func Success(c *app.RequestContext, data any) {
	SuccessWithMsg(c, data, "ok")
}

func SuccessWithMsg(c *app.RequestContext, data any, message string) {
	writeJSON(c, CodeOK, message, data)
}

// Fail 返回失败响应。
// HTTP 状态统一为 200，成功 / 失败由业务码 code 判断，避免客户端把 HTTP 4xx/5xx 当成不同的错误通道。
func Fail(c *app.RequestContext, code int, message string) {
	FailWithData(c, code, message, nil)
}

// FailWithData 返回带附加数据的失败响应，例如字段校验详情。
func FailWithData(c *app.RequestContext, code int, message string, data any) {
	writeJSON(c, code, message, data)
}

func writeJSON(c *app.RequestContext, code int, message string, data any) {
	metrics.ObserveBusiness(c.FullPath(), code)
	c.JSON(http.StatusOK, Body{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
