package site

import (
	"context"
	"errors"
	"strconv"
	"strings"

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
	"suki/xqyun/middleware"
)

// newRenderCache 按配置创建前台渲染缓存；未启用时返回不读写 Redis 的空实现。
func newRenderCache(app application.Application) *renderrepository.Cache {
	cfg := app.Config().RenderCache
	if !cfg.Enabled {
		return renderrepository.New(nil, 0)
	}
	return renderrepository.New(app.Redis(), cfg.TTL)
}

// newPageService 按请求组装页面用例服务，避免在路由注册阶段提前连接数据层。
func newPageService(app application.Application) *siteservice.PageService {
	return siteservice.NewPageService(
		pagerepository.New(app.DB()),
		userrepository.New(app.DB()),
		newRenderCache(app),
	)
}

// newMenuService 按请求组装导航菜单用例服务。
func newMenuService(app application.Application) *siteservice.MenuService {
	return siteservice.NewMenuService(menurepository.New(app.DB()), newRenderCache(app))
}

// newSettingService 按请求组装站点配置用例服务。
func newSettingService(app application.Application) *siteservice.SettingService {
	return siteservice.NewSettingService(settingrepository.New(app.DB()), newRenderCache(app))
}

// adminID 从登录会话读取当前管理员 ID。
// 取不到时返回 nil：记录仍会写入，只是没有操作人。
func adminID(c *app.RequestContext) *uint64 {
	identity, ok := middleware.AdminIdentityFromContext(c)
	if !ok || identity.UserID == 0 {
		return nil
	}
	id := identity.UserID
	return &id
}

// fail 把用例错误转换为统一响应；未知错误返回 500 并记录日志。
func fail(ctx context.Context, c *app.RequestContext, err error) {
	var businessError *siteservice.Error
	if errors.As(err, &businessError) {
		response.Fail(c, businessError.Code, businessError.Message)
		return
	}

	hlog.CtxErrorf(ctx, "admin site: %v", err)
	response.Fail(c, response.CodeInternal, "操作失败，请稍后重试")
}

// pathID 解析路径参数中的数字 ID，失败时直接写入参数错误响应。
func pathID(c *app.RequestContext, name string, message string) (uint64, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, response.CodeBadRequest, message)
		return 0, false
	}
	return id, true
}

// pagination 读取 page / page_size，默认 1 / 20，最大 100。
func pagination(c *app.RequestContext) (int, int, bool) {
	page, pageSize := 1, 20

	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			response.Fail(c, response.CodeBadRequest, "分页参数不正确")
			return 0, 0, false
		}
		page = value
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			response.Fail(c, response.CodeBadRequest, "分页参数不正确")
			return 0, 0, false
		}
		if value > 100 {
			value = 100
		}
		pageSize = value
	}
	return page, pageSize, true
}
