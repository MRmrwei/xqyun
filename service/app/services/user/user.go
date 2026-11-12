// Package user 定义管理后台账号资料与密码相关的业务接口。
package user

import (
	"context"
	"errors"

	"suki/xqyun/app/models"
)

var (
	// ErrUserNotFound 表示账号不存在或已删除。
	ErrUserNotFound = errors.New("账号不存在")
	// ErrCurrentPasswordIncorrect 表示当前密码校验失败。
	ErrCurrentPasswordIncorrect = errors.New("当前密码不正确")
	// ErrPasswordTooShort 表示新密码少于 8 个字符。
	ErrPasswordTooShort = errors.New("新密码至少需要 8 个字符")
	// ErrPasswordTooLong 表示新密码超过 bcrypt 的 72 字节上限。
	ErrPasswordTooLong = errors.New("新密码不能超过 72 字节")
	// ErrNameRequired 表示显示名称为空。
	ErrNameRequired = errors.New("请填写显示名称")
	// ErrNameTooLong 表示显示名称超过 100 个字符。
	ErrNameTooLong = errors.New("显示名称不能超过 100 个字符")
	// ErrEmailInvalid 表示邮箱格式不正确。
	ErrEmailInvalid = errors.New("邮箱格式不正确")
	// ErrEmailTooLong 表示邮箱超过 255 个字符。
	ErrEmailTooLong = errors.New("邮箱不能超过 255 个字符")
	// ErrMobileTooLong 表示手机号超过 32 个字符。
	ErrMobileTooLong = errors.New("手机号不能超过 32 个字符")
	// ErrAvatarTooLarge 表示头像超过 2MB。
	ErrAvatarTooLarge = errors.New("头像不能超过 2MB")
)

// ChangePasswordInput 是修改管理员密码入参。
// UserID 来自登录会话，由控制器解析后传入，不接受前端指定。
type ChangePasswordInput struct {
	UserID          uint64
	CurrentPassword string
	NewPassword     string
}

// UpdateProfileInput 是修改管理员资料入参。
// 邮箱、手机号、头像传空字符串表示清空。
type UpdateProfileInput struct {
	UserID uint64
	Name   string
	Email  string
	Mobile string
	Avatar string
}

// PasswordChanger 是修改管理后台密码的用例契约。
// 实现负责校验当前密码、校验新密码规则（至少 8 个字符、不超过 bcrypt 的 72 字节上限）、
// 用 bcrypt 重新哈希并写入 xqyun_users；成功后吊销该用户全部刷新令牌，要求重新登录。
// 密码错误时返回 ErrCurrentPasswordIncorrect。
type PasswordChanger interface {
	ChangePassword(ctx context.Context, input ChangePasswordInput) error
}

// ProfileUpdater 是修改管理后台资料的用例契约。
// 实现负责校验显示名称、邮箱、手机号与头像大小，更新后返回最新账号信息。
type ProfileUpdater interface {
	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*models.User, error)
}
