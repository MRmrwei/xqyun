package impl

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"suki/xqyun/app/core/application"
)

// Application 是 application.Application 的默认实现。
type Application struct {
	db     *gorm.DB
	redis  *redis.Client
	config application.Config
}

var _ application.Application = (*Application)(nil)

// New 创建 Application，并完成 DB / Redis 的初始化。
// Redis 初始化失败时会先关闭已经建立的 DB 连接，避免资源泄漏。
func New(ctx context.Context, cfg application.Config) (*Application, error) {
	db, err := NewDB(ctx, cfg.DB)
	if err != nil {
		return nil, err
	}

	rdb, err := NewRedis(ctx, cfg.Redis)
	if err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	return &Application{
		db:     db,
		redis:  rdb,
		config: cfg,
	}, nil
}

// DB 返回 MySQL 连接。
func (a *Application) DB() *gorm.DB {
	return a.db
}

// Redis 返回 Redis 客户端；未启用 Redis 时为 nil。
func (a *Application) Redis() *redis.Client {
	return a.redis
}

// Config 返回应用配置拷贝；密钥与 map 都复制，避免调用方修改内部状态。
func (a *Application) Config() application.Config {
	config := a.config
	config.Session.Secret = append([]byte(nil), a.config.Session.Secret...)
	config.JWT.Clients = make(map[string]application.JWTClientConfig, len(a.config.JWT.Clients))
	for name, client := range a.config.JWT.Clients {
		client.Secret = append([]byte(nil), client.Secret...)
		config.JWT.Clients[name] = client
	}
	return config
}

// Close 释放 DB 与 Redis 连接，并合并返回所有关闭错误。
func (a *Application) Close() error {
	var errs []error

	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err != nil {
			errs = append(errs, fmt.Errorf("application: get sql db: %w", err))
		} else if err := sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("application: close db: %w", err))
		}
		a.db = nil
	}

	if a.redis != nil {
		if err := a.redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("application: close redis: %w", err))
		}
		a.redis = nil
	}

	return errors.Join(errs...)
}
