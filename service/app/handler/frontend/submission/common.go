package submission

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"

	"suki/xqyun/app/core/response"
	submissionservice "suki/xqyun/app/services/submission"
	"suki/xqyun/app/core/application"
	submissionrepository "suki/xqyun/app/repository/submission"
	submissionimpl "suki/xqyun/app/services/submission/impl"
)

// newService 按请求组装前台提交用例。
// 返回接口类型：前台只该拿到提交能力，拿不到后台的列表 / 详情 / 状态管理。
func newService(app application.Application) submissionservice.Submitter {
	return submissionimpl.New(submissionrepository.New(app.DB()))
}

// fail 把用例错误转换为统一响应；未知错误返回 500 并记录日志。
func fail(ctx context.Context, c *app.RequestContext, err error) {
	var businessError *submissionservice.Error
	if errors.As(err, &businessError) {
		response.Fail(c, businessError.Code, businessError.Message)
		return
	}

	hlog.CtxErrorf(ctx, "frontend submission: %v", err)
	response.Fail(c, response.CodeInternal, "提交失败，请稍后重试")
}
