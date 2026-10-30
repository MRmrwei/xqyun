package submission

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"suki/xqyun/app/core/response"
	"suki/xqyun/app/dto"
)

// UnreadCount 返回待处理留言数量，供后台侧边栏红点轮询。
// GET /api/v1/admin/submissions/unread-count
// 只返回一个数字，不带列表与其余状态统计。
func (s *Controller) UnreadCount(ctx context.Context, c *app.RequestContext) {
	count, err := newService(s.App()).UnreadCount(ctx)
	if err != nil {
		fail(ctx, c, err)
		return
	}
	response.Success(c, dto.AdminSubmissionUnreadCountResponse{Count: count})
}
