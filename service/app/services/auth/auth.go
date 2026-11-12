// Package auth 定义登录业务接口与令牌结构。
package auth

import (
	"context"

	"suki/xqyun/app/models"
)

// TokenPair 是登录成功后签发的令牌。
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	// ExpiresIn 是 Access Token 的有效期，单位为秒。
	ExpiresIn int
}

// LoginResult 是登录成功后的令牌与账号信息。
type LoginResult struct {
	Pair *TokenPair
	User *models.User
}

// Credentials 是登录凭据与请求元信息。
type Credentials struct {
	Username  string
	Password  string
	UserAgent string
	IP        string
}

// Authenticator 定义登录能力。
type Authenticator interface {
	Login(ctx context.Context, credentials Credentials) (*LoginResult, error)
}

// Logouter 定义退出登录能力。
// 实现删除当前登录会话；会话缺失或已失效都应视为成功（幂等），
// 以便 handler 无论结果如何都能完成响应。
type Logouter interface {
	Logout(ctx context.Context) error
}
