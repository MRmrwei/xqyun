# 小鹊云 · 官网与内容管理平台

小鹊云是一套「后台编排、前台渲染」的企业官网系统：

- 运营人员在管理后台搭建页面、维护导航与站点配置、管理素材、跟进留言
- 访客看到的是支持中英文、SEO 友好、响应式的官网
- Go 后端提供认证、页面、站点、素材、留言与前台渲染接口

当前初版已完成，并在测试服务器完成 Docker 部署与压测验证。

## 项目结构

| 目录 | 说明 | 技术栈 |
| --- | --- | --- |
| [`frontend/`](./frontend/README.md) | 官网前台，SSR | Nuxt 3 + Vue 3 + TypeScript |
| [`admin/`](./admin/README.md) | 管理后台，SPA | Vue 3 + Vite + Element Plus + Pinia |
| [`service/`](./service/README.md) | API 服务 | Go 1.25 + CloudWeGo Hertz + GORM |
| [`deploy/`](./deploy) | 数据库初始化、迁移、Docker 部署 | MySQL 8 / Redis 7 / Nginx |
| [`docs/`](./docs) | 方案、接口、运营手册与压测报告 | Markdown / Word |

## 架构

```text
                     ┌──────────────────────┐
                     │        浏览器         │
                     └──────────┬───────────┘
                                │
                     ┌──────────▼───────────┐
                     │   Nginx 统一入口      │
                     │  80 端口              │
                     └───┬───────┬───────┬──┘
                         │       │       │
              ┌──────────▼─┐ ┌───▼────┐ ┌▼────────────┐
              │ 官网 Nuxt  │ │ /admin │ │ /api/v1     │
              │ SSR :3000  │ │ 静态   │ │ Go :8888    │
              └──────┬─────┘ └────────┘ └──────┬──────┘
                     │                         │
                     └──────────┬──────────────┘
                                │
                    ┌───────────▼────────────┐
                    │ MySQL 8  /  Redis 7    │
                    │ 业务数据 / 会话 / 缓存  │
                    └────────────────────────┘
```

生产入口：

| 路径 | 内容 |
| --- | --- |
| `/` | 官网 |
| `/admin/` | 管理后台 |
| `/api/v1/` | Go API |
| `/uploads/` | 素材文件 |

## 功能概览

### 官网

- Nuxt 3 SSR，服务端输出完整 HTML
- 中英文，默认中文，英文使用 `/en`
- 页面、导航、站点配置来自 Go API
- 搜索 SEO：title、description、canonical、hreflang、robots
- 未发布页面返回 404 + `noindex`
- 联系表单提交到后台表单留言
- Logo、品牌色、favicon 跟随站点配置

### 管理后台

- 登录、Session 会话（HttpOnly Cookie）与 `/admin/me` 启动校验
- 页面内容、顶部导航、底部导航、站点配置
- 三栏页面编辑器、iframe 实时预览、拖拽排序
- 自动保存草稿、手动保存、发布 / 下架
- 中英文翻译配对
- 14 种预置区块
- 素材管理：视频 / 图片上传、搜索、重命名、批量删除
- 表单留言：红点提醒、筛选、状态标记、CSV 导出
- 管理员资料、头像、修改密码

### Go 后端

- 管理后台认证（hertz-contrib/sessions + Redis）与前台内容渲染
- 页面、菜单、站点配置 CRUD
- 媒体素材上传与本地磁盘存储
- 表单提交与留言管理
- 前台渲染聚合接口
- Redis 渲染缓存与主动失效

## 技术栈

| 层 | 技术 |
| --- | --- |
| 官网 | Nuxt 3、Vue 3、TypeScript、Nitro、`@nuxtjs/i18n` |
| 管理后台 | Vue 3、Vite、Element Plus、Pinia、TipTap |
| 后端 | Go 1.25、CloudWeGo Hertz、GORM、hertz-contrib/sessions（JWT 保留给后续前台用户） |
| 数据库 | MySQL 8 |
| 缓存 / 会话 | Redis 7 |
| 迁移 | golang-migrate |
| 部署 | Docker Compose、Nginx、Node 22 |

## 本地开发

前置条件：

- Node.js 22+
- Go 1.25+
- MySQL 8
- Redis 7

### 1. 启动 Go 后端

```bash
cd service
cp .env.example .env
# 填写数据库、Redis 与 JWT 配置
go run .
```

健康检查：

```text
http://127.0.0.1:8888/ping
```

首次运行需要建库并执行迁移：

```bash
mysql -uroot -p < ../deploy/init.sql

migrate -path ../migration \
  -database "mysql://用户名:密码@tcp(127.0.0.1:3306)/xqyun" up
```

### 2. 启动管理后台

```bash
cd admin
npm install
npm run dev
```

访问：

```text
http://localhost:5174/#/pages
```

Vite 会把 `/api` 代理到 `http://127.0.0.1:8888`。

### 3. 启动官网

```bash
cd frontend
npm install
npm run dev
```

访问：

```text
http://localhost:3000
```

## Docker 部署

部署包在：

```text
deploy/docker/xqyun/
```

包含管理后台静态文件、官网 `.output`、Linux Go 二进制、Nginx 配置与数据库迁移。

```bash
cd deploy/docker/xqyun
cp .env.example .env
# 填写 MySQL 密码、JWT_SECRET、素材公开地址
docker compose up -d
```

使用官方镜像：

- `nginx:1.27-alpine`
- `node:22-alpine`
- `alpine:3.22`
- `mysql:8.0`
- `redis:7-alpine`
- `migrate/migrate`

不需要在服务器上构建，也不需要上传源码。详细说明见 [`deploy/docker/xqyun/README.md`](./deploy/docker/xqyun/README.md)。

数据目录：

```text
deploy/docker/xqyun/data/
├── mysql/
├── redis/
├── uploads/
└── logs/
```

## 目录

```text
.
├── admin/                 管理后台
├── frontend/              官网
├── service/               Go 后端
├── deploy/
│   ├── init.sql           建库
│   ├── docker/xqyun/      Docker 部署包
│   └── ...
├── docs/                  设计与运营文档
├── migration/             数据库迁移
└── README.md
```

## 文档

| 文档 | 内容 |
| --- | --- |
| [项目方案](./docs/项目方案.md) | 产品、架构、数据模型、SEO 与多语言 |
| [Go 后端接口设计](./docs/Go后端接口设计.md) | 接口清单、请求响应、错误码 |
| [MySQL 数据库表设计](./docs/MySQL数据库表设计.md) | 表结构与字段说明 |
| [初版上线方案](./docs/初版上线方案.md) | 初版范围、接口顺序、建表清单 |
| [权限控制方案](./docs/权限控制方案.md) | 后台 RBAC 设计（Casbin 纯 URL 策略），仅设计未实现 |
| [运营人员使用手册](./docs/运营人员使用手册.docx) | 后台日常操作说明（Word） |
| [压测报告](./docs/压测报告.md) | 性能测试结果与容量建议 |
| [管理后台说明](./admin/README.md) | 后台功能与开发 |
| [官网说明](./frontend/README.md) | 前台功能与开发 |
| [后端说明](./service/README.md) | 后端分层与接口 |

## 当前状态

版本：`v1.0.0`（2026-10-22 发布，tag `v1.0.0`）。

初版已完成：

- 官网 SSR、中英文、SEO、404
- 管理后台主要功能
- Go 后端主要接口
- 素材库与本地磁盘存储
- 表单提交与留言管理
- Docker 部署包与部署验证

尚未包含：

- 新闻 / 产品 / 案例独立内容库与详情页
- RBAC 角色权限（二期采用 Casbin，设计见 [权限控制方案](./docs/权限控制方案.md)）
- 服务端图形验证码
- 对象存储 / CDN / 正式 HTTPS
- 前台用户登录（JWT 配置已预留）

## 安全说明

- `.env`、数据库密码、`JWT_SECRET`、服务器地址等敏感信息不进入仓库
- 后台 API 使用 HttpOnly 会话 Cookie `xqyun_admin_session`，并非 Bearer Token
- 会话 Cookie 默认 `Path=/api/v1/admin`、`SameSite=Lax`、`HttpOnly`
- 非 GET / HEAD / OPTIONS 的后台请求会校验 `Origin`
- 访问日志不记录 Authorization、Cookie 与请求体
- 初始管理员账号为 `admin` / `123456`，首次登录后请立即修改
- 生产环境请配置 HTTPS，并把 `SESSION_COOKIE_SECURE` 设为 `true`
