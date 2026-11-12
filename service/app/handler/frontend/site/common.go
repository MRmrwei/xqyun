package site

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"

	"suki/xqyun/app/core/application"
	"suki/xqyun/app/core/response"
	menurepository "suki/xqyun/app/repository/menu"
	pagerepository "suki/xqyun/app/repository/page"
	renderrepository "suki/xqyun/app/repository/render"
	settingrepository "suki/xqyun/app/repository/setting"
	userrepository "suki/xqyun/app/repository/user"
	siteservice "suki/xqyun/app/services/site"
	siteimpl "suki/xqyun/app/services/site/impl"
)

// newRenderCache 按配置创建前台渲染缓存；未启用时返回不读写 Redis 的空实现。
func newRenderCache(app application.Application) *renderrepository.Cache {
	cfg := app.Config().RenderCache
	if !cfg.Enabled {
		return renderrepository.New(nil, 0)
	}
	return renderrepository.New(app.Redis(), cfg.TTL)
}

// newMenuService 按请求组装导航用例服务。
func newMenuService(app application.Application) *siteservice.MenuService {
	return siteservice.NewMenuService(menurepository.New(app.DB()), newRenderCache(app))
}

// newSettingService 按请求组装站点配置用例服务。
func newSettingService(app application.Application) *siteservice.SettingService {
	return siteservice.NewSettingService(settingrepository.New(app.DB()), newRenderCache(app))
}

// newRenderer 按请求组装前台页面渲染实现，避免在路由注册阶段提前连接数据层。
func newRenderer(app application.Application) *siteimpl.Renderer {
	return siteimpl.NewRenderer(
		pagerepository.New(app.DB()),
		newMenuService(app),
		settingrepository.New(app.DB()),
		userrepository.New(app.DB()),
		newRenderCache(app),
	)
}

// fail 把用例错误转换为统一响应；未知错误返回 500 并记录日志。
func fail(ctx context.Context, c *app.RequestContext, err error) {
	var businessError *siteservice.Error
	if errors.As(err, &businessError) {
		response.Fail(c, businessError.Code, businessError.Message)
		return
	}

	hlog.CtxErrorf(ctx, "frontend site: %v", err)
	response.Fail(c, response.CodeInternal, "页面加载失败，请稍后重试")
}
