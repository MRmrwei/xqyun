package impl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"suki/xqyun/app/models"
	userrepository "suki/xqyun/app/repository/user"
	"suki/xqyun/app/services/auth"
)

// AdminSessionConfig 描述管理后台 Session 登录服务的参数。
// TTL 由 HTTP 层写入会话 Cookie 与存储时使用，这里只用于组装响应中的 expires_in。
type AdminSessionConfig struct {
	TTL time.Duration
}

// AdminSession 基于 hertz-contrib/sessions 的会话存储实现管理后台登录、续期与退出。
// 会话本身（Cookie、Redis 记录）由 HTTP 层适配的 auth.SessionStore 负责，
// service 层只读写会话键，不关心具体存储实现。
type AdminSession struct {
	users userrepository.UserRepository
	store auth.SessionStore
	ttl   time.Duration
	now   func() time.Time
}

var (
	_ auth.Authenticator = (*AdminSession)(nil)
	_ auth.Logouter      = (*AdminSession)(nil)
)

// NewAdminSession 创建管理后台 Session 认证服务。
func NewAdminSession(
	users userrepository.UserRepository,
	store auth.SessionStore,
	cfg AdminSessionConfig,
) *AdminSession {
	return &AdminSession{
		users: users,
		store: store,
		ttl:   cfg.TTL,
		now:   time.Now,
	}
}

// Login 校验账号密码，并把管理员信息写入当前请求的登录会话。
func (s *AdminSession) Login(ctx context.Context, credentials auth.Credentials) (*auth.LoginResult, error) {
	username := strings.TrimSpace(credentials.Username)
	if username == "" || credentials.Password == "" {
		return nil, auth.ErrInvalidCredentials
	}
	if s.store == nil {
		return nil, fmt.Errorf("认证服务：会话存储未初始化")
	}

	user, err := authenticateAdmin(ctx, s.users, username, credentials.Password)
	if err != nil {
		return nil, err
	}

	now := s.now()
	ip := optionalString(credentials.IP, maxIPLength)
	s.store.Set(auth.SessionKeyAdminUserID, user.ID)
	s.store.Set(auth.SessionKeyAdminUsername, user.Username)
	if err := s.store.Save(); err != nil {
		return nil, err
	}

	// 登录信息只用于展示；写入失败时清掉刚建立的会话，避免留下不一致的登录状态。
	if err := persistSessionLoginState(ctx, s.users, s.store, user, now.Unix(), ip); err != nil {
		return nil, err
	}

	return &auth.LoginResult{
		Pair: &auth.TokenPair{
			// Session 模式没有令牌，凭证仅存在于 HttpOnly Cookie 中。
			ExpiresIn: int(s.ttl / time.Second),
		},
		User: user,
	}, nil
}

// NewAdminSessionLogout 创建只用于退出登录的 AdminSession，不需要用户仓库与 TTL 配置。
func NewAdminSessionLogout(store auth.SessionStore) *AdminSession {
	return &AdminSession{store: store, now: time.Now}
}

// Logout 清空并删除当前会话；会话本来不存在时同样返回成功（幂等）。
func (s *AdminSession) Logout(_ context.Context) error {
	if s.store == nil {
		return fmt.Errorf("认证服务：会话存储未初始化")
	}
	s.store.Clear()
	return s.store.Save()
}

// persistSessionLoginState 记录最近登录信息；写入失败时删除刚建立的会话。
func persistSessionLoginState(
	ctx context.Context,
	users userrepository.UserRepository,
	store auth.SessionStore,
	user *models.User,
	nowUnix int64,
	ip *string,
) error {
	if err := users.UpdateLoginState(ctx, user.ID, nowUnix, ip); err != nil {
		store.Clear()
		_ = store.Save()
		return err
	}

	user.LastLoginAt = nowUnix
	user.LastLoginIP = ip
	user.UpdatedAt = nowUnix
	return nil
}
