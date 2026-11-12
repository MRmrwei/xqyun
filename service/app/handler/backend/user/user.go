package user

import (
	"context"
	"errors"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"

	"suki/xqyun/app/core/application"
	"suki/xqyun/app/core/controller"
	"suki/xqyun/app/core/response"
	"suki/xqyun/app/dto"
	"suki/xqyun/app/models"
	userrepository "suki/xqyun/app/repository/user"
	userservice "suki/xqyun/app/services/user"
	userimpl "suki/xqyun/app/services/user/impl"
	sessionutil "suki/xqyun/app/utils/session"
	"suki/xqyun/middleware"
)

// UserController 处理当前管理员的资料与密码。
// 路由前缀：/api/v1/admin/user
type UserController struct {
	*controller.BaseController
}

// NewUserController 创建管理员资料 / 密码控制器。
func NewUserController(app application.Application) *UserController {
	return &UserController{BaseController: controller.New(app)}
}

// UpdateProfile 修改当前管理员资料（显示名称、邮箱、手机号、头像）。
// PUT /api/v1/admin/user/profile
func (u *UserController) UpdateProfile(ctx context.Context, c *app.RequestContext) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, response.CodeUnauthorized, "请先登录")
		return
	}

	var req dto.AdminProfileUpdateRequest
	if err := c.Bind(&req); err != nil {
		response.Fail(c, response.CodeBadRequest, "请求参数格式不正确")
		return
	}

	account, err := userimpl.NewProfileUpdater(
		userrepository.New(u.App().DB()),
	).UpdateProfile(ctx, userservice.UpdateProfileInput{
		UserID: userID,
		Name:   req.Name,
		Email:  req.Email,
		Mobile: req.Mobile,
		Avatar: req.Avatar,
	})
	if err != nil {
		fail(ctx, c, err)
		return
	}
	response.Success(c, toAdminProfile(account))
}

// ChangePassword 修改当前管理员密码。
// PUT /api/v1/admin/user/password
func (u *UserController) ChangePassword(ctx context.Context, c *app.RequestContext) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, response.CodeUnauthorized, "请先登录")
		return
	}

	var req dto.AdminPasswordChangeRequest
	if err := c.Bind(&req); err != nil {
		response.Fail(c, response.CodeBadRequest, "请求参数格式不正确")
		return
	}

	err := userimpl.NewPasswordChanger(
		userrepository.New(u.App().DB()),
	).ChangePassword(ctx, userservice.ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		fail(ctx, c, err)
		return
	}

	// 改密后删除当前登录会话，要求使用新密码重新登录。
	// 其他设备上的会话会在 TTL 到期后失效，当前版本不做按用户批量踢下线。
	sessionStore := sessionutil.New(c, u.App().Config().Session)
	sessionStore.Clear()
	if err := sessionStore.Save(); err != nil {
		hlog.CtxWarnf(ctx, "改密后清除登录会话失败：%v", err)
	}
	response.Success(c, nil)
}

// Me 返回当前登录管理员资料。
// GET /api/v1/admin/me
func (u *UserController) Me(ctx context.Context, c *app.RequestContext) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Fail(c, response.CodeUnauthorized, "请先登录")
		return
	}

	account, err := userrepository.New(u.App().DB()).FindByID(ctx, userID)
	if errors.Is(err, userrepository.ErrNotFound) {
		response.Fail(c, response.CodeNotFound, "账号不存在")
		return
	}
	if err != nil {
		hlog.CtxErrorf(ctx, "admin me: %v", err)
		response.Fail(c, response.CodeInternal, "获取账号信息失败，请稍后重试")
		return
	}
	response.Success(c, toAdminProfile(account))
}

// currentUserID 从登录会话读取当前管理员 ID。
func currentUserID(c *app.RequestContext) (uint64, bool) {
	identity, ok := middleware.AdminIdentityFromContext(c)
	if !ok || identity.UserID == 0 {
		return 0, false
	}
	return identity.UserID, true
}

// toAdminProfile 把账号模型转换为前台响应结构。
func toAdminProfile(account *models.User) dto.AdminProfile {
	profile := dto.AdminProfile{
		ID:       strconv.FormatUint(account.ID, 10),
		Username: account.Username,
		Name:     account.Name,
	}
	if account.Email != nil {
		profile.Email = *account.Email
	}
	if account.Mobile != nil {
		profile.Mobile = *account.Mobile
	}
	if account.Avatar != nil {
		profile.Avatar = *account.Avatar
	}
	return profile
}

// fail 把业务错误映射为统一响应；未知错误记录日志并返回 500。
func fail(ctx context.Context, c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, userservice.ErrUserNotFound):
		response.Fail(c, response.CodeNotFound, err.Error())
	case errors.Is(err, userservice.ErrCurrentPasswordIncorrect),
		errors.Is(err, userservice.ErrPasswordTooShort),
		errors.Is(err, userservice.ErrPasswordTooLong),
		errors.Is(err, userservice.ErrNameRequired),
		errors.Is(err, userservice.ErrNameTooLong),
		errors.Is(err, userservice.ErrEmailInvalid),
		errors.Is(err, userservice.ErrEmailTooLong),
		errors.Is(err, userservice.ErrMobileTooLong),
		errors.Is(err, userservice.ErrAvatarTooLarge):
		response.Fail(c, response.CodeBadRequest, err.Error())
	default:
		hlog.CtxErrorf(ctx, "admin user: %v", err)
		response.Fail(c, response.CodeInternal, "操作失败，请稍后重试")
	}
}
