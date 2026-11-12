package media

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"

	"suki/xqyun/app/core/application"
	"suki/xqyun/app/core/response"
	"suki/xqyun/app/dto"
	"suki/xqyun/app/models"
	mediarepository "suki/xqyun/app/repository/media"
	mediaservice "suki/xqyun/app/services/media"
	mediaimpl "suki/xqyun/app/services/media/impl"
	"suki/xqyun/middleware"
)

// newService 装配素材用例：先 NewFileUploader 创建上传能力，再嵌入 Service 并补上素材记录。
func newService(app application.Application) (*mediaimpl.Service, error) {
	cfg := app.Config().Upload
	storage, err := mediarepository.NewStorage(
		cfg.Provider,
		cfg.Dir,
		cfg.URLPrefix,
		cfg.PublicBaseURL,
	)
	if err != nil {
		return nil, err
	}
	uploader := mediaimpl.NewFileUploader(storage, mediaservice.Limits{
		MaxVideoBytes: cfg.MaxVideoBytes,
		MaxImageBytes: cfg.MaxImageBytes,
	})
	return mediaimpl.New(mediarepository.New(app.DB()), uploader), nil
}

// currentUserID 从登录会话读取当前管理员 ID。
func currentUserID(c *app.RequestContext) *uint64 {
	identity, ok := middleware.AdminIdentityFromContext(c)
	if !ok || identity.UserID == 0 {
		return nil
	}
	id := identity.UserID
	return &id
}

// pathID 解析路径参数中的数字 ID，失败时直接写入参数错误响应。
func pathID(c *app.RequestContext, name string, message string) (uint64, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, response.CodeBadRequest, message)
		return 0, false
	}
	return id, true
}

// toMediaItem 把素材模型转换为响应结构。
func toMediaItem(item *models.Media) dto.AdminMediaItem {
	return dto.AdminMediaItem{
		ID:           strconv.FormatUint(item.ID, 10),
		Kind:         item.Kind,
		URL:          item.URL,
		Filename:     item.Filename,
		OriginalName: item.OriginalName,
		Mime:         item.Mime,
		Size:         int64(item.Size),
		CreatedAt:    item.CreatedAt,
	}
}

// fail 把业务错误转换为统一响应；未知错误返回 500 并记录日志。
func fail(ctx context.Context, c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, mediaservice.ErrMediaNotFound):
		response.Fail(c, response.CodeNotFound, err.Error())
	case errors.Is(err, mediaservice.ErrEmptyFile),
		errors.Is(err, mediaservice.ErrUnsupportedFile),
		errors.Is(err, mediaservice.ErrUnsupportedKind),
		errors.Is(err, mediaservice.ErrOriginalNameTooLong),
		errors.Is(err, mediaservice.ErrOriginalNameRequired):
		response.Fail(c, response.CodeBadRequest, err.Error())
	case errors.Is(err, mediaservice.ErrFileTooLarge):
		response.Fail(c, response.CodeBadRequest, "文件超过大小上限")
	default:
		hlog.CtxErrorf(ctx, "admin media: %v", err)
		response.Fail(c, response.CodeInternal, "操作失败，请稍后重试")
	}
}
