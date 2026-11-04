# 小鹊云 Go 后端

基于 CloudWeGo Hertz 的官网与管理后台 API 服务。

提供后台认证、页面编排、导航与站点配置、媒体素材、表单留言、管理员资料与前台渲染接口。

## 技术栈

- Go 1.25
- CloudWeGo Hertz
- GORM + MySQL 8
- Redis 7（登录会话与渲染缓存）
- JWT（HS256 Access Token + Redis Refresh Token）
- golang-migrate（数据库迁移）
- bcrypt（密码哈希）

## 分层

```text
app/handler/       控制器：参数绑定、调用 service、组装统一响应
app/services/      业务用例与接口
app/repository/    数据访问：MySQL / Redis / 本地文件
app/models/        GORM 模型
app/dto/           接口请求 / 响应结构
app/core/          配置、启动装配、日志、统一响应、公共控制器
middleware/        管理后台 JWT 认证
router/            后台 / 前台路由注册
migration/         数据库迁移（仓库根目录）
```

规则：

- handler 不写业务逻辑，不直接访问 GORM / Redis。
- service 不依赖 DTO 入参，数据访问统一走 repository。
- 路由统一在 `router/` 注册；静态素材路由在 `router/static.go`。

## 目录

```text
app/
├── core/
│   ├── application/   MySQL / Redis 生命周期
│   ├── bootstrap/     启动装配
│   ├── config/        YAML + 环境变量
│   ├── controller/    基础控制器
│   ├── logger/        日志初始化与轮转
│   └── response/      统一响应
├── dto/               请求 / 响应结构
├── handler/
│   ├── backend/       管理后台接口
│   └── frontend/      官网公开接口
├── models/            GORM 模型
├── repository/        数据访问
├── services/          业务用例
└── utils/             Cookie、ULID 等工具
middleware/            管理后台 Bearer JWT
router/                路由注册
```

## 已实现接口

### 管理后台（`/api/v1/admin`）

- 认证：登录、刷新令牌、退出登录
- 管理员：修改资料、修改密码
- 页面：列表、新建、详情、保存草稿、删除、翻译配对、发布、下架
- 菜单：列表、详情、整体保存
- 站点配置：读取、保存
- 媒体素材：上传、列表、详情、重命名、单个删除、批量删除
- 表单留言：列表、详情、状态标记、CSV 导出

### 官网前台（`/api/v1/frontend`）

- `GET /render/page`：页面 + 导航 + 站点配置 + 内容集合
- `GET /site/config`：站点配置
- `GET /site/nav`：顶部 / 页脚导航
- `POST /forms/:key/submissions`：公开表单提交

尚未实现：

- `GET /admin/me`
- 内容库与分类（`render/content`、`contents`、`categories`）
- RBAC 角色权限、审计日志
- 服务端图形验证码、登录 / 表单限流

## 配置

非敏感配置在 `config.yaml`：

- 服务端口（默认 `:8888`）
- MySQL 地址、数据库名、连接池
- Redis 地址与 DB
- JWT 客户端与 Cookie
- 渲染缓存 TTL
- 上传目录、访问前缀、大小限制
- 日志级别、目录与轮转

敏感配置只从 `.env` 读取：

```text
MYSQL_USERNAME
MYSQL_PASSWORD
REDIS_PASSWORD
JWT_SECRET
```

`JWT_SECRET` 至少 32 字节，缺失时服务启动失败。`.env` 不提交仓库。

常用覆盖变量：

```text
APP_ENV
SERVER_ADDR
MYSQL_HOST / MYSQL_PORT / MYSQL_DATABASE
REDIS_ENABLED / REDIS_HOST / REDIS_PORT / REDIS_DB
JWT_COOKIE_SECURE / JWT_COOKIE_SAME_SITE
UPLOAD_PROVIDER / UPLOAD_DIR / UPLOAD_URL_PREFIX / UPLOAD_PUBLIC_BASE_URL
LOG_LEVEL / LOG_DIR
```

## 本地运行

前置条件：

- Go 1.25+
- MySQL 8
- Redis 7

```bash
cp .env.example .env
# 填写数据库、Redis 与 JWT 配置

go run .          # http://127.0.0.1:8888/ping
go build ./...
go vet ./...
go test ./...
```

数据库初始化和迁移：

```bash
mysql -uroot -p < ../deploy/init.sql

migrate -path ../migration \
  -database "mysql://用户名:密码@tcp(127.0.0.1:3306)/xqyun" up
```

迁移文件在仓库根目录 `migration/`，命名：

```text
时间戳_描述.up.sql
时间戳_描述.down.sql
```

已执行过的迁移不要直接修改；需要变更时新增迁移，并同步已有环境。

## 认证与会话

- 登录成功返回 Access Token，默认 15 分钟
- Refresh Token 只存 Redis，首次登录通过 HttpOnly Cookie 下发
- 刷新时轮换旧令牌，旧令牌立即失效
- 修改密码后吊销该账号全部刷新令牌
- 后台受保护接口统一经过 `middleware/admin_auth.go`

## 渲染缓存

前台渲染数据按页面、导航、站点配置分片缓存在 Redis：

- 默认 TTL 10 分钟
- 发布、下架、删除页面、修改页面路径、保存导航 / 站点配置时主动失效
- Redis 不可用时自动回源数据库

## 媒体存储

- `repository/media.Storage` 是存储接口
- 默认实现为本地磁盘，按 `yyyy/mm` 分目录
- `services/media.Service` 抽象上传、列表、重命名、删除
- 静态文件由 `router.RegisterUploadStatic` 提供，支持 Range，视频可拖动进度

后续可替换为对象存储实现，业务层不需要改动。

## 统一响应

所有响应 HTTP 状态码均为 200，业务结果看 `code`：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

常见错误码：

- `0` 成功
- `40000` 参数错误
- `40100` 未登录 / 令牌无效
- `40300` 无权限
- `40400` 不存在
- `40900` 冲突
- `50000` 服务端错误

## 生产部署

Docker 部署使用官方 Alpine 镜像挂载 Linux 二进制，配置与数据目录由 compose 管理。详见：

- [`deploy/docker/xqyun/README.md`](../deploy/docker/xqyun/README.md)
- [`docs/Go后端接口设计.md`](../docs/Go后端接口设计.md)

## 相关文档

- [项目总体说明](../README.md)
- [项目方案](../docs/项目方案.md)
- [接口设计](../docs/Go后端接口设计.md)
- [数据库表设计](../docs/MySQL数据库表设计.md)
