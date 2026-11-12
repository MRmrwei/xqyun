package application

import (
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 聚合应用启动所需的基础设施配置。
type Config struct {
	DB          DBConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Session     SessionConfig
	RenderCache RenderCacheConfig
	Upload      UploadConfig
}

// UploadConfig 描述媒体文件存储。
type UploadConfig struct {
	// Provider 是存储实现，当前支持 local，后续可扩展对象存储。
	Provider      string
	Dir           string
	URLPrefix     string
	PublicBaseURL string
	MaxVideoBytes int64
	MaxImageBytes int64
}

// RenderCacheConfig 描述前台渲染数据（页面 / 导航 / 站点配置）的 Redis 缓存。
type RenderCacheConfig struct {
	// Enabled 为 false 时不读写缓存，所有请求直接查数据库。
	Enabled bool
	// TTL 是缓存有效期，作为失效逻辑漏删时的兜底。
	TTL time.Duration
}

// SessionConfig 描述管理后台登录会话（hertz-contrib/sessions）。
type SessionConfig struct {
	// Name 是会话 Cookie 名。
	Name string
	// Path / Domain / Secure / SameSite 是 Cookie 属性。
	Path     string
	Domain   string
	Secure   bool
	SameSite string
	// TTL 是会话有效期，Cookie 与 Redis 记录使用同一个值。
	TTL time.Duration
	// Secret 是 Cookie 签名密钥，由 JWT_SECRET 派生。
	Secret []byte
}

// JWTConfig 描述 JWT 签发方与各客户端令牌参数。
type JWTConfig struct {
	Issuer  string
	Clients map[string]JWTClientConfig
	Cookie  CookieConfig
}

// CookieConfig 描述 Refresh Token 的 HttpOnly Cookie。
type CookieConfig struct {
	Name     string
	Path     string
	Domain   string
	Secure   bool
	SameSite string
}

// JWTClientConfig 描述单个客户端的令牌签名与有效期参数。
type JWTClientConfig struct {
	Secret     []byte
	Audience   string
	TokenType  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// Client 返回指定客户端的 JWT 配置。
func (c JWTConfig) Client(name string) (JWTClientConfig, bool) {
	client, ok := c.Clients[name]
	return client, ok
}

// DBConfig 描述 MySQL 连接参数。
type DBConfig struct {
	DSN string
	// MaxOpenConns 最大打开连接数，<=0 时使用驱动默认值。
	MaxOpenConns int
	// MaxIdleConns 最大空闲连接数，<=0 时使用驱动默认值。
	MaxIdleConns int
	// ConnMaxLifetime 连接最长存活时间，<=0 时不限制。
	ConnMaxLifetime time.Duration
	// LogLevel GORM 日志级别，默认 logger.Warn。
	LogLevel logger.LogLevel
}

// RedisConfig 描述 Redis 连接参数。
type RedisConfig struct {
	// Enabled 为 false 时不连接 Redis，Application.Redis() 返回 nil。
	Enabled bool
	Addr    string
	// Username / Password 按 Redis 实例配置填写，未设置 ACL 时留空。
	Username string
	Password string
	DB       int
	PoolSize int
	// DialTimeout / ReadTimeout / WriteTimeout 为 0 时使用 go-redis 默认值。
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Application 定义应用级基础设施的统一访问入口。
// 业务层只依赖这个接口，便于替换实现或在测试中使用 mock。
type Application interface {
	// DB 返回 MySQL 连接（GORM）。
	DB() *gorm.DB

	// Redis 返回 Redis 客户端；未启用 Redis 时返回 nil。
	Redis() *redis.Client

	// Config 返回应用配置（MySQL / Redis / JWT / Session / 渲染缓存 / 上传）。
	// 返回值是拷贝，调用方修改不会影响内部状态。
	Config() Config

	// Close 释放数据库与 Redis 连接。
	Close() error
}
