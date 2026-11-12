// Package session 把 hertz-contrib/sessions 适配成 service 层使用的 auth.SessionStore。
package session

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	hertzsessions "github.com/hertz-contrib/sessions"
	sessionredis "github.com/hertz-contrib/sessions/redis"

	"suki/xqyun/app/core/application"
	"suki/xqyun/app/services/auth"
)

// cookieOptions 把配置转换为 hertz-contrib/sessions 的 Cookie 属性。
func cookieOptions(cfg application.SessionConfig) hertzsessions.Options {
	return hertzsessions.Options{
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		MaxAge:   int(cfg.TTL.Seconds()),
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: parseSameSite(cfg.SameSite),
	}
}

// NewRedisStore 创建基于 Redis 的会话存储，多实例部署共享同一份会话数据。
func NewRedisStore(
	cfg application.SessionConfig,
	redisCfg application.RedisConfig,
) (hertzsessions.Store, error) {
	if redisCfg.Addr == "" {
		return nil, fmt.Errorf("会话存储：Redis 地址未配置")
	}
	if len(cfg.Secret) < 32 {
		return nil, fmt.Errorf("会话存储：Cookie 签名密钥至少需要 32 字节")
	}

	poolSize := redisCfg.PoolSize
	if poolSize <= 0 {
		poolSize = 10
	}
	store, err := sessionredis.NewStoreWithDB(
		poolSize,
		"tcp",
		redisCfg.Addr,
		redisCfg.Password,
		strconv.Itoa(redisCfg.DB),
		cfg.Secret,
	)
	if err != nil {
		return nil, fmt.Errorf("会话存储：初始化 Redis 失败：%w", err)
	}
	store.Options(cookieOptions(cfg))
	return store, nil
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// Store 是 auth.SessionStore 的 hertz-contrib/sessions 实现。
type Store struct {
	session hertzsessions.Session
	options hertzsessions.Options
}

var _ auth.SessionStore = (*Store)(nil)

// New 创建当前请求的会话适配器。
// 调用前必须在该路由上挂载 sessions.New(...)，否则 hertzsessions.Default 会 panic。
func New(c *app.RequestContext, cfg application.SessionConfig) *Store {
	return NewWithSession(hertzsessions.Default(c), cfg)
}

// NewWithSession 用已经获取的会话创建适配器。
// 需要自行控制会话来源（例如非 HTTP 场景或测试）时使用；cfg 决定 TTL 与 Cookie 属性。
func NewWithSession(session hertzsessions.Session, cfg application.SessionConfig) *Store {
	return &Store{session: session, options: cookieOptions(cfg)}
}

// Set 写入会话值，调用 Save 后生效。
func (s *Store) Set(key string, value any) {
	s.session.Set(key, value)
}

// Get 读取会话值，不存在时第二个返回值为 false。
func (s *Store) Get(key string) (any, bool) {
	value := s.session.Get(key)
	return value, value != nil
}

// Clear 清空会话，并把 MaxAge 设为 -1；
// hertz-contrib/sessions 在 Save 时发现 MaxAge <= 0 会删除服务端记录并清除 Cookie。
func (s *Store) Clear() {
	s.session.Clear()

	options := s.options
	options.MaxAge = -1
	s.session.Options(options)
}

// Save 持久化会话。
func (s *Store) Save() error {
	return s.session.Save()
}
