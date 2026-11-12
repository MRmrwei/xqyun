package user

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"

	"suki/xqyun/app/core/application"
	"suki/xqyun/app/core/controller"
	"suki/xqyun/app/core/response"
	"suki/xqyun/app/services/adminauth"
	authimpl "suki/xqyun/app/services/auth/impl"
	sessionutil "suki/xqyun/app/utils/session"
)

// LogoutController 处理管理后台退出登录接口。
type LogoutController struct {
	*controller.BaseController
}

// NewLogoutController 创建退出登录控制器，只保存 Application。
func NewLogoutController(app application.Application) *LogoutController {
	return &LogoutController{BaseController: controller.New(app)}
}

// Logout 删除当前登录会话并清除会话 Cookie。
func (l *LogoutController) Logout(ctx context.Context, c *app.RequestContext) {
	sessionStore := sessionutil.New(c, l.App().Config().Session)
	logoutService := adminauth.NewLogoutService(authimpl.NewAdminSessionLogout(sessionStore))

	if err := logoutService.Logout(ctx); err != nil {
		hlog.CtxErrorf(ctx, "admin logout: %v", err)
		response.Fail(c, response.CodeInternal, "退出登录失败，请稍后重试")
		return
	}

	response.Success(c, nil)
}
