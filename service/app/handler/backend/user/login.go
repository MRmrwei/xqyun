// Package user 提供管理后台用户认证相关的 HTTP 控制器（登录、退出登录、当前管理员）。
package user

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"

	"suki/xqyun/app/core/application"
	"suki/xqyun/app/core/controller"
	"suki/xqyun/app/core/response"
	"suki/xqyun/app/dto"
	userrepository "suki/xqyun/app/repository/user"
	"suki/xqyun/app/services/adminauth"
	authimpl "suki/xqyun/app/services/auth/impl"
	sessionutil "suki/xqyun/app/utils/session"
)

// LoginController 处理管理后台登录接口。
type LoginController struct {
	*controller.BaseController
}

// NewLoginController 创建登录控制器，只保存 Application。
func NewLoginController(app application.Application) *LoginController {
	return &LoginController{BaseController: controller.New(app)}
}

// Login 解析请求、创建登录用例服务并输出响应。
// 登录成功后会话由 hertz-contrib/sessions 写入 HttpOnly Cookie。
func (l *LoginController) Login(ctx context.Context, c *app.RequestContext) {
	var req dto.AdminLoginRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, response.CodeBadRequest, "请求参数格式不正确")
		return
	}

	sessionConfig := l.App().Config().Session
	sessionStore := sessionutil.New(c, sessionConfig)
	authenticator := authimpl.NewAdminSession(
		userrepository.New(l.App().DB()),
		sessionStore,
		authimpl.AdminSessionConfig{TTL: sessionConfig.TTL},
	)
	loginService := adminauth.NewLoginService(authenticator)

	result, err := loginService.Login(ctx, adminauth.LoginInput{
		Username:  req.Username,
		Password:  req.Password,
		UserAgent: string(c.UserAgent()),
		IP:        c.ClientIP(),
	})
	if err != nil {
		l.fail(ctx, c, err)
		return
	}

	response.Success(c, result)
}

// fail 把登录用例的业务错误转换为统一响应；未知错误统一返回 500 并记录日志。
func (l *LoginController) fail(ctx context.Context, c *app.RequestContext, err error) {
	var businessError *adminauth.Error
	if errors.As(err, &businessError) {
		response.Fail(c, businessError.Code, businessError.Message)
		return
	}

	hlog.CtxErrorf(ctx, "admin login: %v", err)
	response.Fail(c, response.CodeInternal, "登录失败，请稍后重试")
}
