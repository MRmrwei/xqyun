package middleware

import (
	"github.com/cloudwego/hertz/pkg/app"
	hertzsessions "github.com/hertz-contrib/sessions"

	"suki/xqyun/app/core/application"
	sessionutil "suki/xqyun/app/utils/session"
)

func NewAdminSession(app application.Application) app.HandlerFunc {
	cfg := app.Config()
	store, err := sessionutil.NewRedisStore(cfg.Session, cfg.Redis)
	if err != nil {
		panic(err.Error())
	}
	return hertzsessions.New(cfg.Session.Name, store)
}
