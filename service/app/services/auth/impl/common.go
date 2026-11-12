// Package impl 提供管理后台认证接口的具体实现。
// 管理后台使用 Session（admin_session.go），实现 auth.Authenticator / auth.Logouter；
// JWT 相关工具保留在 auth 包中，供后续前台用户登录使用。
package impl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"suki/xqyun/app/models"
	sessionrepository "suki/xqyun/app/repository/session"
	userrepository "suki/xqyun/app/repository/user"
	"suki/xqyun/app/services/auth"
)

const (
	maxUserAgentLength = 500
	maxIPLength        = 45
)

// authenticateAdmin 校验后台账号密码与账号状态，供 JWT / Session 两种登录实现共用。
func authenticateAdmin(
	ctx context.Context,
	users userrepository.UserRepository,
	username string,
	password string,
) (*models.User, error) {
	if users == nil {
		return nil, fmt.Errorf("认证服务：数据访问层未初始化")
	}

	user, err := users.FindByUsername(ctx, username)
	if errors.Is(err, userrepository.ErrNotFound) {
		return nil, auth.ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !auth.VerifyPassword(password, user.PasswordHash) {
		return nil, auth.ErrInvalidCredentials
	}
	if user.Status != 1 {
		return nil, auth.ErrAccountDisabled
	}
	return user, nil
}

// newSessionRecord 构造一条 Redis 登录会话记录，时间戳统一以秒保存。
func newSessionRecord(
	userID uint64,
	tokenHash string,
	tokenType string,
	userAgent *string,
	ip *string,
	now time.Time,
	ttl time.Duration,
) sessionrepository.Token {
	return sessionrepository.Token{
		UserID:    userID,
		TokenHash: tokenHash,
		TokenType: tokenType,
		UserAgent: userAgent,
		IP:        ip,
		ExpiresAt: now.Add(ttl).Unix(),
		CreatedAt: now.Unix(),
	}
}

// persistLoginState 记录最近登录信息；写入失败时回滚刚创建的会话，避免留下不可用的凭证。
func persistLoginState(
	ctx context.Context,
	users userrepository.UserRepository,
	sessions sessionrepository.Repository,
	user *models.User,
	tokenHash string,
	tokenType string,
	nowUnix int64,
	ip *string,
) error {
	if err := users.UpdateLoginState(ctx, user.ID, nowUnix, ip); err != nil {
		_ = sessions.Revoke(ctx, tokenHash, tokenType)
		return err
	}

	user.LastLoginAt = nowUnix
	user.LastLoginIP = ip
	user.UpdatedAt = nowUnix
	return nil
}

// optionalString 去除首尾空白并截断，空值返回 nil。
func optionalString(value string, max int) *string {
	value = truncate(strings.TrimSpace(value), max)
	if value == "" {
		return nil
	}
	return &value
}

func truncate(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
