// Package submission 定义表单留言的用例契约：前台提交接口、入参、状态与错误。
//
// 具体实现见 app/services/submission/impl；后台管理能力是具体方法，不在这里抽接口。
package submission

import (
	"context"

	"suki/xqyun/app/dto"
)

// 留言处理状态。
const (
	StatusNew        = "new"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusSpam       = "spam"
)

// ListInput 是留言列表 / 导出的查询条件。
type ListInput struct {
	Status      string
	FormKey     string
	Locale      string
	Keyword     string
	CreatedFrom int64
	CreatedTo   int64
	Page        int
	PageSize    int
}

// SubmitInput 是前台提交表单的原始内容与请求信息。
type SubmitInput struct {
	FormKey   string
	Locale    string
	Name      string
	Email     string
	Company   string
	Phone     string
	Message   string
	IP        string
	UserAgent string
	Now       int64
}

// Submitter 是前台提交能力。
//
// 只保留这一个接口：前台是公开入口，控制器拿接口类型就调不到后台的列表 / 状态管理能力，
// 以后加防刷、人机验证、限流也只需替换实现。后台列表 / 详情 / 状态是同一套管理逻辑，
// 没有替换需求，直接用 impl.Service 的具体方法。
type Submitter interface {
	Submit(ctx context.Context, input SubmitInput) (dto.FrontendSubmissionResult, error)
}
