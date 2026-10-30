// Package submission 提供管理后台「线索 → 表单留言」的 HTTP 控制器。
//
// 路由前缀：/api/v1/admin/submissions
//   - list.go   留言列表与状态统计
//   - detail.go 留言详情
//   - status.go 标记处理状态
//   - export.go 导出 CSV
//   - unread_count.go 待处理数量（侧边栏红点轮询）
//
// 控制器只负责参数解析与响应组装，业务逻辑在 app/services/submission，
// 数据访问在 app/repository/submission。
package submission
