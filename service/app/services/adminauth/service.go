// Package adminauth 组装管理后台登录用例。
// 输入校验、认证调用、业务错误映射与响应 DTO 组装都在这一层完成，handler 只负责 HTTP 参数与响应。
package adminauth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"suki/xqyun/app/core/response"
	"suki/xqyun/app/dto"
	"suki/xqyun/app/services/auth"
)

// Error 是可直接映射为统一响应结构的业务错误。
// HTTP 状态固定 200，客户端按 Code 判断具体错误。
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

var (
	// ErrInvalidInput 表示账号或密码为空。
	ErrInvalidInput = &Error{
		Code:    response.CodeBadRequest,
		Message: "请输入用户名和密码",
	}
	// ErrInvalidCredentials 表示账号不存在或密码错误，对外不区分两者。
	ErrInvalidCredentials = &Error{
		Code:    response.CodeUnauthorized,
		Message: "用户名或密码错误",
	}
	// ErrAccountDisabled 表示账号已禁用。
	ErrAccountDisabled = &Error{
		Code:    response.CodeForbidden,
		Message: "账号已被禁用",
	}
)

// LoginInput 是登录用例的输入。
type LoginInput struct {
	Username  string
	Password  string
	UserAgent string
	IP        string
}

// Service 是管理后台认证用例服务，覆盖登录与退出登录。
type Service struct {
	authenticator auth.Authenticator
	logouter      auth.Logouter
}

// NewLoginService 创建管理后台登录用例服务。
func NewLoginService(authenticator auth.Authenticator) *Service {
	return &Service{authenticator: authenticator}
}

// NewLogoutService 创建退出登录用例服务。
func NewLogoutService(logouter auth.Logouter) *Service {
	return &Service{logouter: logouter}
}

// Login 校验输入、调用认证服务，并返回可直接输出的登录响应。
func (s *Service) Login(ctx context.Context, input LoginInput) (*dto.AdminLoginResponse, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" || input.Password == "" {
		return nil, ErrInvalidInput
	}
	if s.authenticator == nil {
		return nil, fmt.Errorf("登录服务未初始化")
	}

	result, err := s.authenticator.Login(ctx, auth.Credentials{
		Username:  username,
		Password:  input.Password,
		UserAgent: input.UserAgent,
		IP:        input.IP,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			return nil, ErrInvalidCredentials
		case errors.Is(err, auth.ErrAccountDisabled):
			return nil, ErrAccountDisabled
		default:
			return nil, err
		}
	}
	if result == nil || result.Pair == nil || result.User == nil {
		return nil, fmt.Errorf("登录服务返回了空结果")
	}
	return adminResponse(result), nil
}

// Logout 删除当前登录会话；会话本来不存在时同样成功。
func (s *Service) Logout(ctx context.Context) error {
	if s.logouter == nil {
		return fmt.Errorf("退出登录服务未初始化")
	}
	return s.logouter.Logout(ctx)
}

// adminResponse 把认证结果组装为登录响应 DTO。
func adminResponse(result *auth.LoginResult) *dto.AdminLoginResponse {
	user := result.User
	return &dto.AdminLoginResponse{
		ExpiresIn: result.Pair.ExpiresIn,
		User: dto.AdminUser{
			ID:          strconv.FormatUint(user.ID, 10),
			Username:    user.Username,
			Name:        user.Name,
			Email:       stringValue(user.Email),
			Mobile:      stringValue(user.Mobile),
			Avatar:      stringValue(user.Avatar),
			Roles:       []string{},
			Permissions: []string{},
		},
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
