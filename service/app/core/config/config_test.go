package config

import (
	"strings"
	"testing"
	"time"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func TestJWTClientDefaults(t *testing.T) {
	cfg := &Config{JWT: JWTConfig{Secret: testJWTSecret}}
	cfg.setDefaults()

	if _, ok := cfg.JWT.Clients["admin"]; ok {
		t.Fatal("管理后台已改用 Session，不应再生成 admin JWT 客户端")
	}

	frontend := cfg.JWT.Clients["frontend"]
	if frontend.Audience != "frontend-api" || frontend.TokenType != "frontend" {
		t.Fatalf("frontend 受众 / 类型不正确：%#v", frontend)
	}
	if frontend.AccessTTL != "30m" || frontend.RefreshTTL != "720h" {
		t.Fatalf("frontend TTL 不正确：%#v", frontend)
	}
}

func TestApplicationConfigBuildsJWTClient(t *testing.T) {
	cfg := &Config{
		Redis: RedisConfig{Enabled: true},
		JWT:   JWTConfig{Secret: testJWTSecret, Issuer: "xiaoque-admin"},
	}
	cfg.setDefaults()

	appCfg, err := cfg.ApplicationConfig()
	if err != nil {
		t.Fatalf("ApplicationConfig() 返回错误：%v", err)
	}

	frontend, ok := appCfg.JWT.Client("frontend")
	if !ok {
		t.Fatal("frontend 客户端未生成")
	}
	if frontend.AccessTTL != 30*time.Minute || frontend.RefreshTTL != 720*time.Hour {
		t.Fatalf("frontend TTL 不正确：%#v", frontend)
	}
	if string(frontend.Secret) != testJWTSecret {
		t.Fatal("frontend 密钥未正确传递")
	}
}

func TestApplicationConfigRejectsInvalidJWT(t *testing.T) {
	t.Run("short secret", func(t *testing.T) {
		cfg := &Config{
			Redis: RedisConfig{Enabled: true},
			JWT:   JWTConfig{Secret: strings.Repeat("s", 31)},
		}
		cfg.setDefaults()
		if _, err := cfg.ApplicationConfig(); err == nil {
			t.Fatal("密钥长度不足时应返回错误")
		}
	})

	t.Run("invalid ttl", func(t *testing.T) {
		cfg := &Config{Redis: RedisConfig{Enabled: true}, JWT: JWTConfig{
			Secret: testJWTSecret,
			Issuer: "xiaoque-admin",
			Clients: map[string]JWTClientConfig{
				"frontend": {
					Audience:   "frontend-api",
					TokenType:  "frontend",
					AccessTTL:  "invalid",
					RefreshTTL: "720h",
				},
			},
		}}
		if _, err := cfg.ApplicationConfig(); err == nil {
			t.Fatal("非法 TTL 应返回错误")
		}
	})
}

func TestApplicationConfigRequiresRedis(t *testing.T) {
	cfg := &Config{JWT: JWTConfig{Secret: testJWTSecret}}
	cfg.setDefaults()

	if _, err := cfg.ApplicationConfig(); err == nil {
		t.Fatal("Redis 未启用时应返回错误")
	}
}

func TestSessionConfigDefaults(t *testing.T) {
	cfg := &Config{
		Redis: RedisConfig{Enabled: true},
		JWT:   JWTConfig{Secret: testJWTSecret},
	}
	cfg.setDefaults()

	appCfg, err := cfg.ApplicationConfig()
	if err != nil {
		t.Fatalf("ApplicationConfig() 返回错误：%v", err)
	}

	session := appCfg.Session
	if session.Name != "xqyun_admin_session" || session.Path != "/api/v1/admin" {
		t.Fatalf("Session Cookie 默认值不正确：%#v", session)
	}
	if session.TTL != 12*time.Hour {
		t.Fatalf("Session TTL = %v，期望 12h", session.TTL)
	}
	if len(session.Secret) != 32 {
		t.Fatalf("Session 签名密钥长度 = %d，期望 32", len(session.Secret))
	}
}

func TestJWTClientEnvOverride(t *testing.T) {
	t.Setenv("JWT_FRONTEND_ACCESS_TTL", "5m")
	t.Setenv("JWT_FRONTEND_REFRESH_TTL", "168h")

	cfg := &Config{JWT: JWTConfig{Secret: testJWTSecret}}
	cfg.setDefaults()
	if err := cfg.applyEnv(); err != nil {
		t.Fatalf("applyEnv() 返回错误：%v", err)
	}

	if got := cfg.JWT.Clients["frontend"].AccessTTL; got != "5m" {
		t.Fatalf("frontend access_ttl = %q，期望 5m", got)
	}
	if got := cfg.JWT.Clients["frontend"].RefreshTTL; got != "168h" {
		t.Fatalf("frontend refresh_ttl = %q，期望 168h", got)
	}
}
