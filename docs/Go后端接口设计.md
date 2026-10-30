# 小鹊云 Go 后端接口设计

> 服务目录：`service/`
> Web 框架：CloudWeGo Hertz

## 1. 通用约定

| 项 | 约定 |
| --- | --- |
| 本地地址 | `http://127.0.0.1:8888` |
| 官网前台接口前缀 | `/api/v1/frontend/` |
| 管理后台接口前缀 | `/api/v1/admin/` |
| 请求 / 响应格式 | `application/json; charset=utf-8` |
| 时间格式 | Unix 时间戳（秒），JSON number，Go 侧使用 `int64` |
| 语言参数 | `locale=zh-CN` / `locale=en`，默认 `zh-CN` |
| 认证方式 | `Authorization: Bearer <access_token>` |
| ID 传输 | JSON 中统一用字符串，避免 JavaScript 对大整数的精度丢失 |
| 分页参数 | `page` 从 1 开始，`page_size` 默认 20，最大 100 |

所有 `created_at`、`updated_at`、`deleted_at`、`published_at` 等时间字段统一使用 Unix 时间戳（秒），Go 侧为 `int64`，JSON 中为数字；`0` 表示未设置、未发布或未删除。

### 1.1 统一响应

当前 `app/core/response` 已实现：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

- `code = 0` 表示成功。
- 失败时 `code` 为业务错误码，`message` 为可展示文案，`data` 可选。
- 所有响应（包括失败）HTTP 状态统一为 200，成功与失败只通过 `code` 判断，业务错误不使用 HTTP 4xx/5xx。
- 后续建议在响应中增加 `request_id`，用于日志关联和问题排查。

分页响应的 `data`：

```json
{
  "items": [],
  "total": 128,
  "page": 1,
  "page_size": 20
}
```

### 1.2 错误码

当前 `response` 包已定义：

| code | 含义 |
| --- | --- |
| `0` | 成功 |
| `40000` | 参数错误 |
| `40100` | 未登录 / 令牌无效 |
| `40300` | 无权限 |
| `40400` | 资源不存在 |
| `50000` | 服务端错误 |

建议后续按同一规则扩展：

| code | 含义 |
| --- | --- |
| `40001` | 字段校验失败 |
| `40101` | Access Token 过期 |
| `40900` | 资源冲突（路径 / slug 重复） |
| `42900` | 请求过于频繁 |
| `50001` | 下游依赖失败（对象存储等） |

---

## 2. 通用数据结构

建议后续在 `service/app/domain/` 下定义 DTO，handler 只做参数解析和响应组装。

### 2.1 页面

```go
type PageBlock struct {
    ID    string         `json:"id"`
    Type  string         `json:"type"`
    Props map[string]any `json:"props"`
}

type SEO struct {
    Title       string   `json:"title"`
    Description string   `json:"description"`
    Keywords    []string `json:"keywords"`
}

// 页面列表项
type PageSummary struct {
    ID           string            `json:"id"`
    GroupID      string            `json:"group_id"`
    Locale       string            `json:"locale"`
    Path         string            `json:"path"`
    Name         string            `json:"name"`
    Status       string            `json:"status"` // draft / published
    Version      int               `json:"version"`
    UpdatedAt    int64             `json:"updated_at"`
    UpdatedBy    string            `json:"updated_by"`
    Translations []PageTranslation `json:"translations"`
}

// 同一翻译组下的其他语言版本，后台“前往翻译版本”使用
type PageTranslation struct {
    PageID string `json:"page_id"`
    Locale string `json:"locale"`
    Path   string `json:"path"`
    Name   string `json:"name"`
    Status string `json:"status"`
}

// 后台页面详情：当前草稿
type PageDraft struct {
    ID        string      `json:"id"`
    GroupID   string      `json:"group_id"`
    Locale    string      `json:"locale"`
    Path      string      `json:"path"`
    Name      string      `json:"name"`
    Status    string      `json:"status"`
    Version   int         `json:"version"`
    SEO       SEO         `json:"seo"`
    Blocks    []PageBlock `json:"blocks"`
    UpdatedAt int64       `json:"updated_at"`
}

// 对外发布的只读页面
type PublishedPage struct {
    ID          string            `json:"id"`
    GroupID     string            `json:"group_id"`
    Locale      string            `json:"locale"`
    Path        string            `json:"path"`
    Name        string            `json:"name"`
    SEO         SEO               `json:"seo"`
    Blocks      []PageBlock       `json:"blocks"`
    Version     int               `json:"version"`
    PublishedAt int64             `json:"published_at"`
    PublishedBy string            `json:"published_by"`
    // 仅包含已发布语言；未发布语言不出现，避免 hreflang 指向 404
    Alternates  map[string]string `json:"alternates"`
}

type ContentDetail struct {
    ContentSummary
    BodyHTML   string            `json:"body_html"`
    BodyJSON   map[string]any    `json:"body_json"`
    SEO        SEO               `json:"seo"`
    Extra      map[string]any    `json:"extra"`
    Status     string            `json:"status"`
    Version    int               `json:"version"`
    Sort       int               `json:"sort"`
    Alternates map[string]string `json:"alternates"`
    CreatedAt  int64             `json:"created_at"`
    UpdatedAt  int64             `json:"updated_at"`
}
```

`extra` 按类型区分：

| type | 字段 |
| --- | --- |
| `news` | `category`、`read_minutes` |
| `product` | `label`、`description`、`bullets` |
| `case` | `client`、`industry`、`metric`、`metric_suffix`、`metric_label` |

内容与页面一样区分草稿与线上内容：`xqyun_contents` 保存当前草稿，`published_content` 保存最近一次发布的内容；发布时直接覆盖，不保留历史版本。

### 2.3 分类与媒体

```go
type Category struct {
    ID          string `json:"id"`
    GroupID     string `json:"group_id"`
    Type        string `json:"type"`
    Locale      string `json:"locale"`
    Slug        string `json:"slug"`
    Name        string `json:"name"`
    Description string `json:"description"`
    ParentID    string `json:"parent_id,omitempty"`
    Sort        int    `json:"sort"`
}

type Media struct {
    ID         string            `json:"id"`
    Filename   string            `json:"filename"`
    URL        string            `json:"url"`
    Mime       string            `json:"mime"`
    Size       int64             `json:"size"`
    Width      int               `json:"width"`
    Height     int               `json:"height"`
    Alt        map[string]string `json:"alt"` // {"zh-CN": "...", "en": "..."}
    CreatedAt int64            `json:"created_at"`
}
```

### 2.4 站点配置与导航

```go
type NavItem struct {
    ID       string `json:"id"`
    Label    string `json:"label"`
    Path     string `json:"path"`
    LinkType string `json:"link_type"` // page / content / external / none
    Target   string `json:"target"`    // _self / _blank
    Visible  bool   `json:"visible"`
    Sort     int    `json:"sort"`
    Children []NavItem `json:"children,omitempty"`
}

type BrandSettings struct {
    Logo           string `json:"logo"`
    Accent         string `json:"accent"`
    Background     string `json:"background"`
    DarkBackground string `json:"dark_background"`
}

type HeaderCta struct {
    Key     string `json:"key"`
    Label   string `json:"label"`
    Href    string `json:"href"`
    Variant string `json:"variant"` // primary / secondary / ghost
    Visible bool   `json:"visible"`
}

type HeaderSettings struct {
    ShowLanguage    bool             `json:"show_language"`
    Sticky          bool             `json:"sticky"`
    ShowAnnouncement bool            `json:"show_announcement"`
    Announcement    map[string]any   `json:"announcement"`
    CTAs            []HeaderCta      `json:"ctas"`
}

type FooterSettings struct {
    ShowNewsletter bool           `json:"show_newsletter"`
    ShowContact    bool           `json:"show_contact"`
    CTA            map[string]any `json:"cta"`
    Columns        []NavItem      `json:"columns"`
    BottomLinks    []NavItem      `json:"bottom_links"`
}

type SiteConfig struct {
    SiteName    map[string]string `json:"site_name"`
    CompanyName map[string]string `json:"company_name"`
    Tagline     map[string]string `json:"tagline"`
    Email       string            `json:"email"`
    Phone       string            `json:"phone"`
    Address     map[string]string `json:"address"`
    ICP         string            `json:"icp"`
    Copyright   map[string]string `json:"copyright"`
    Socials     []map[string]string `json:"socials"`
    Brand       BrandSettings     `json:"brand"`
    Header      HeaderSettings    `json:"header"`
    Footer      FooterSettings    `json:"footer"`
    AllowIndex  bool              `json:"allow_index"`
}
```

### 2.5 页面渲染聚合

```go
type RenderPageData struct {
    Page        PublishedPage `json:"page"`
    Nav         []NavItem     `json:"nav"`
    Site        SiteConfig    `json:"site"`
    Collections Collections   `json:"collections"`
}

type Collections struct {
    News     []ContentSummary `json:"news"`
    Products []ContentSummary `json:"products"`
    Cases    []ContentSummary `json:"cases"`
}
```

### 2.6 认证

```go
type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginResponse struct {
    AccessToken string    `json:"access_token"`
    ExpiresIn   int       `json:"expires_in"` // 秒
    User        AdminUser `json:"user"`
}

type AdminUser struct {
    ID          string   `json:"id"`
    Username    string   `json:"username"`
    Name        string   `json:"name"`
    Email       string   `json:"email"`
    Mobile      string   `json:"mobile"`
    Avatar      string   `json:"avatar"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
}
```

---

## 3. 官网前台接口（公开）

### 3.1 `GET /ping`（已实现）

健康检查，用于本地联调和探活。

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "message": "pong"
  }
}
```

### 3.2 `GET /api/v1/frontend/render/page`

获取页面渲染所需的全部数据，服务端一次聚合，避免前台为每个区块单独请求。

请求参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `path` | string | 是 | 页面路径，如 `/about`、`/en/about-us` |
| `locale` | string | 否 | 默认 `zh-CN` |

响应 `data`：

```json
{
  "page": {
    "id": "1001",
    "group_id": "01J...",
    "locale": "zh-CN",
    "path": "/about",
    "name": "关于我们",
    "seo": {
      "title": "关于我们 - 小鹊云",
      "description": "...",
      "keywords": ["小鹊云"]
    },
    "blocks": [],
    "version": 3,
    "published_at": 1792375200,
    "published_by": "朱家伟",
    "alternates": {
      "zh-CN": "/about",
      "en": "/en/about-us"
    }
  },
  "nav": [],
  "site": {},
  "collections": {
    "news": [],
    "products": [],
    "cases": []
  }
}
```

行为：

- 只返回已发布页面；未发布或不存在返回 `40400`。
- 数据源区块（新闻 / 产品 / 案例）由服务端按 `limit` 预取，前台不再逐块请求。
- `alternates` 由 `translation_group_id` 计算，只包含已发布语言，供前台输出 hreflang 和语言切换；未发布语言不出现在 `alternates` 中。
- 服务端按「页面（locale + path）/ 导航 / 站点配置」三个分片缓存到 Redis，未命中时查库回填；发布、下架、删除页面与保存导航、站点配置时主动失效，`render_cache.ttl`（默认 10 分钟）作为漏删兜底。`render_cache.enabled=false` 时每次都查库。
- 暂未实现 `ETag` / `Cache-Control`。
- 不提供草稿预览：后台预览由 `preview.html` + `postMessage` 在浏览器内完成，不经过该接口。

### 3.3 `GET /api/v1/frontend/render/content`

获取新闻 / 产品 / 案例详情。

请求参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | string | 是 | `news` / `product` / `case` |
| `slug` | string | 是 | 内容 slug |
| `locale` | string | 否 | 默认 `zh-CN` |

响应 `data`：

```json
{
  "content": {
    "id": "2001",
    "type": "news",
    "slug": "product-update",
    "title": "...",
    "version": 2,
    "alternates": {
      "zh-CN": "/news/product-update",
      "en": "/en/news/product-update-en"
    }
  },
  "related": []
}
```

行为：

- 只返回已发布内容，且读取 `published_content`，不直接读 `xqyun_contents` 草稿字段。
- 找不到时返回 `40400`，前台据此输出 404。
- 中英文 slug 独立，`alternates` 只包含已发布译文，前台语言切换直接使用。

### 3.3.1 单语言发布规则

- 每个语言独立发布；发布中文不会级联发布英文，反之亦然。
- 请求未发布语言：`render/page` / `render/content` 返回 `40400`，前台输出 404 并设置 `noindex`。
- `alternates`、hreflang 和 sitemap 只包含已发布语言，不指向未发布地址。
- 语言切换只展示已发布译文；另一语言未发布时隐藏入口或退回该语言首页。
- 后台草稿预览在 `preview.html` iframe 内完成，不受发布状态限制，也不经过前台接口。
- 后续发布另一语言后，该语言立即生效，`alternates` 与 sitemap 同步更新。

### 3.4 `GET /api/v1/frontend/contents`

内容列表，用于新闻 / 产品 / 案例列表页。

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | string | 是 | 内容类型 |
| `locale` | string | 否 | 默认 `zh-CN` |
| `category` | string | 否 | 分类 slug |
| `featured` | bool | 否 | 是否只看推荐 |
| `page` | int | 否 | 默认 1 |
| `page_size` | int | 否 | 默认 20，最大 100 |

响应 `data` 为分页结构，`items` 使用 `ContentSummary`。

### 3.4.1 `GET /api/v1/frontend/categories`

返回某类内容的分类列表，供前台筛选器使用。

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | string | 是 | `news` / `product` / `case` |
| `locale` | string | 否 | 默认 `zh-CN` |

响应 `data`：

```json
{
  "items": []
}
```

`items` 使用 `Category`，只返回已发布内容实际使用的分类或后台启用的分类。

### 3.5 `GET /api/v1/frontend/site/config`

返回站点级配置，脱敏后供前台头部 / 页脚使用。

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `locale` | string | 否 | 默认 `zh-CN` |

返回 `SiteConfig`，不包含统计代码、head 代码等敏感内容；`allow_index` 用于前台输出 robots。

### 3.6 `GET /api/v1/frontend/site/nav`

返回顶部菜单与页脚栏目。

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `locale` | string | 否 | 默认 `zh-CN` |

响应 `data`：

```json
{
  "header": [],
  "footer": [],
  "footer_bottom": []
}
```

菜单项使用 `NavItem`；页脚顶级项是栏目，`children` 是栏目链接。

边界说明：

- 链接结构（顶部菜单、页脚栏目、底部链接）来自 `xqyun_menus` / `xqyun_menu_items`。
- 公告条、顶部按钮、语言入口、吸顶来自 `site/config` 的 `header`。
- 页脚订阅区、联系入口、大字标来自 `site/config` 的 `footer`。
- 前台把两者合并渲染；`render/page` 已经聚合好，不需要分别请求。

### 3.7 `POST /api/v1/frontend/forms/{key}/submissions`

提交表单，例如 `POST /api/v1/frontend/forms/contact/submissions`。

请求体（`email` 选填，`name` / `phone` / `message` 必填）：

```json
{
  "name": "张三",
  "email": "zhangsan@example.com",
  "company": "某某公司",
  "phone": "13800000000",
  "message": "想了解产品",
  "locale": "zh-CN"
}
```

响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": "9001"
  }
}
```

行为：

- 已实现：控制器在 `app/handler/frontend/submission`；用例接口在 `app/services/submission`（`Submitter`），默认实现在 `app/services/submission/impl`，由 handler 包内的 `newService` 按请求装配。
- 服务端校验必填项（称呼 / 联系电话 / 留言）、邮箱格式与各字段长度；邮箱为空时不校验。
- 落库 `xqyun_form_submissions`，`status = new`，`data` 保存完整表单内容，同时记录提交 IP 与 User-Agent。
- 暂未实现同一 IP 频率限制；后续可补，并可触发邮件 / webhook 通知。

### 3.8 `GET /sitemap.xml`、`GET /robots.txt`

不属于 `/api/v1` 前缀。

- `sitemap.xml`：输出已发布页面与内容的双语 URL，带 hreflang alternate。
- `robots.txt`：根据站点 `allow_index` 输出 `index` / `noindex`，并声明 sitemap 地址。

---

## 4. 管理后台接口

后台接口统一要求登录，并在中间件中校验权限码。

### 4.1 认证

| 方法 | 路径 | 作用 | 请求 / 响应 |
| --- | --- | --- | --- |
| POST | `/api/v1/admin/auth/login` | 账号密码登录 | `LoginRequest` → `LoginResponse` |
| POST | `/api/v1/admin/auth/refresh` | 刷新 Access Token | 从 HttpOnly Cookie 读取 Refresh Token → 新令牌与轮换 Cookie |
| POST | `/api/v1/admin/auth/logout` | 吊销刷新令牌 | 从 HttpOnly Cookie 读取 Refresh Token → 空，并清除 Cookie |
| GET | `/api/v1/admin/me` | 当前用户与权限 | → `AdminUser` |

说明：

- Access Token 有效期 15 分钟，刷新令牌 7 天。
- 刷新令牌只存 SHA-256 哈希，写入 Redis（`xqyun:refresh:{token_type}:{hash}`，TTL 与 `refresh_ttl` 一致），刷新时轮换，旧令牌立即失效。
- 登录失败需要限流，避免暴力破解。
- Access Token 为 HS256 JWT，claims 包含 `sub` / `username` / `iss` / `aud` / `token_type` / `iat` / `exp`；现阶段管理后台使用 `aud = admin-api`、`token_type = admin`，前台预留 `frontend-api` / `frontend`。
- JWT 参数按客户端分组：`jwt.clients.admin` / `jwt.clients.frontend`，分别配置 `audience` / `token_type` / `access_ttl` / `refresh_ttl`；`issuer` 与密钥全局共享。
- Refresh Token 不放在响应体，通过 HttpOnly Cookie 下发；前端无法读取，刷新 / 退出时由浏览器自动携带。Cookie 名、Path、Secure、SameSite 由 `jwt.cookie` 配置。
- Redis 会话的 key 与值都带 `token_type`，用于区分管理后台与前台用户。
- 每个用户的刷新令牌哈希同时维护在 `xqyun:refresh:user:{token_type}:{user_id}`（Set，TTL 与刷新令牌一致），用于改密 / 停用账号时批量吊销。

### 4.1.1 当前管理员资料与密码

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| PUT | `/api/v1/admin/user/profile` | 修改显示名称、邮箱、手机号、头像 |
| PUT | `/api/v1/admin/user/password` | 修改密码；成功后吊销该账号全部刷新令牌并清除 Cookie |

说明：

- 用户 ID 从 Access Token 的 `sub` 读取，前端不传，避免越权修改其他账号。
- 改密成功后所有设备的刷新令牌失效，当前浏览器 Cookie 被清除，前端需清除本地会话并跳回登录页；未过期的 Access Token 仍可能有效，最长 15 分钟。
- 资料字段校验：显示名称必填且不超过 100 字；邮箱可空、不超过 255 字且需符合格式；手机号可空、不超过 32 字；头像不超过 2MB。
- OpenAPI 文档：`docs/open-api/backend/admin-user-profile-openapi.yaml`。

### 4.2 页面

| 方法 | 路径 | 作用 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/v1/admin/pages` | 页面列表，支持 `locale` / `status` / `keyword` / 分页 | `page:read` |
| POST | `/api/v1/admin/pages` | 新建页面 | `page:edit` |
| GET | `/api/v1/admin/pages/:id` | 页面详情（草稿） | `page:read` |
| PUT | `/api/v1/admin/pages/:id` | 保存草稿（路径、名称、SEO、区块） | `page:edit` |
| DELETE | `/api/v1/admin/pages/:id` | 软删除页面 | `page:edit` |
| POST | `/api/v1/admin/pages/:id/translate` | 创建 / 获取另一语言配对草稿 | `page:edit` |
| POST | `/api/v1/admin/pages/:id/publish` | 发布当前草稿，覆盖线上内容 | `page:publish` |
| POST | `/api/v1/admin/pages/:id/unpublish` | 下架：状态改为未发布，清空线上内容，草稿保留 | `page:publish` |

保存草稿请求体：

```json
{
  "path": "/about",
  "name": "关于我们",
  "seo": {
    "title": "关于我们 - 小鹊云",
    "description": "...",
    "keywords": ["小鹊云"]
  },
  "blocks": [
    {
      "id": "b_01",
      "type": "hero",
      "props": {}
    }
  ]
}
```

发布请求体：

```json
{
  "note": "更新首屏文案"
}
```

发布响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "version": 4,
    "published_at": 1792375200
  }
}
```

发布与下架的行为：

- 发布：`published_content = draft_content`，`version + 1`，`status = published`。
- 下架：`status = draft`，`published_content = null`，草稿保留，可再次发布。
- 页面状态只有 `draft` / `published`，不再区分“有未发布修改”；列表状态筛选也只有全部 / 已发布 / 未发布。
- 发布只影响当前语言，不会级联发布配对的另一语言；另一语言保持原状态。


翻译请求体：

```json
{
  "from_locale": "zh-CN",
  "to_locale": "en",
  "path": "/en/about-us"
}
```

行为：

- 复制区块结构；可翻译文本清空，图片、链接、样式保留。
- 目标语言已存在配对页面时直接返回，不重复创建。
- 路径冲突返回 `40900`。

### 4.3 内容与分类

| 方法 | 路径 | 作用 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/v1/admin/contents` | 内容列表，支持类型 / 语言 / 状态 / 分类 / 关键词 | `content:read` |
| POST | `/api/v1/admin/contents` | 新建内容 | `content:edit` |
| GET | `/api/v1/admin/contents/:id` | 内容详情 | `content:read` |
| PUT | `/api/v1/admin/contents/:id` | 保存内容 | `content:edit` |
| DELETE | `/api/v1/admin/contents/:id` | 软删除 | `content:edit` |
| POST | `/api/v1/admin/contents/:id/publish` | 发布内容，覆盖线上内容 | `content:publish` |
| POST | `/api/v1/admin/contents/:id/unpublish` | 下架内容：状态改为未发布，清空线上内容 | `content:publish` |
| GET | `/api/v1/admin/categories` | 分类列表 | `content:read` |
| POST | `/api/v1/admin/categories` | 新建分类 | `content:edit` |
| PUT | `/api/v1/admin/categories/:id` | 修改分类 | `content:edit` |
| DELETE | `/api/v1/admin/categories/:id` | 删除分类 | `content:edit` |

内容保存请求体：

```json
{
  "type": "product",
  "locale": "zh-CN",
  "slug": "ai-service",
  "title": "AI 客服",
  "summary": "面向售前咨询的智能问答",
  "cover_media_id": "501",
  "category_id": "21",
  "body_html": "<p>...</p>",
  "body_json": {},
  "seo": {},
  "extra": {
    "label": "AI 客服",
    "description": "...",
    "bullets": ["多轮对话", "知识库接入"]
  },
  "featured": true,
  "sort": 10
}
```

说明：

- 保存接口只写 `xqyun_contents` 草稿字段，不影响线上。
- 发布接口把当前草稿写入 `published_content`，`version + 1`，更新 `published_at`，并写入 `xqyun_publish_records`；旧的线上内容被直接覆盖。
- 前台只读取 `published_content`，因此编辑已发布内容后需要重新发布才生效；不支持回滚。
- 内容详情接口同时返回当前草稿和线上内容，后台可显示“有未发布修改”。

发布响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "version": 2,
    "published_at": 1792375200
  }
}
```

### 4.4 媒体

已实现（视频 / 图片）：

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| POST | `/api/v1/admin/media/upload` | 上传素材，`multipart/form-data`，字段名 `file`，可选 `kind` |
| GET | `/api/v1/admin/media` | 素材列表，支持 `kind`（`video` / `image`，留空为全部）/ `keyword` / `page` / `page_size` |
| GET | `/api/v1/admin/media/:id` | 素材详情 |
| PUT | `/api/v1/admin/media/:id` | 修改素材名称（`{ "original_name": "..." }`） |
| DELETE | `/api/v1/admin/media/:id` | 软删除素材 |
| DELETE | `/api/v1/admin/media?ids=1,2,3` | 批量软删除素材 |

修改多语言 alt 尚未实现。`kind` 作为分类字段保留，后续可扩展其他类型。
删除会先同步删除物理文件，再把数据库记录标记 `deleted_at`；已引用该 URL 的页面将无法再显示该素材，后台删除确认框有提示。

上传 / 列表响应字段：

```json
{
  "id": "501",
  "kind": "video",
  "url": "http://127.0.0.1:8888/uploads/2026/10/xxx.mp4",
  "filename": "2026/10/xxx.mp4",
  "original_name": "产品演示.mp4",
  "mime": "video/mp4",
  "size": 182300,
  "created_at": 1792647037
}
```

限制与存储：

- 视频支持 MP4 / WebM / OGV / OGG / MOV / M4V，单文件默认最大 200MB（`upload.max_video_mb`）。
- 图片支持 JPG / PNG / WebP / GIF / AVIF，单张默认最大 10MB（`upload.max_image_mb`）；暂不接收 SVG。
- 只有上传文件与落库两种逻辑保留接口：`media.FileUploader`（上传文件）与 `media.Uploader`（上传并落库），默认实现在 `services/media/impl`；列表 / 详情 / 重命名 / 删除是直接业务逻辑，不抽接口，由 `impl.Service` 直接提供。更换上传逻辑（转码、病毒扫描、云存储直传等）时新增一个 `FileUploader` 实现并替换装配处即可。
- 分层为：接口 `services/media`、实现 `services/media/impl`、消费者 `handler/backend/media`。装配放在 `handler/backend/media/common.go` 的 `newService`（handler 包里唯一 import 实现的位置）；控制器不保存 service 字段，在每个处理方法内调用 `newService` 创建用例。上传解析仍在 `handler/backend/media/upload.go`，HTTP 路径与响应不变。
- 上传与落库分开：`media.FileUploader` 只做校验并写入存储（依赖 `repository/media.Storage` 与 `media.Limits`，不访问数据库），`newService` 里直接 `NewFileUploader(storage, limits)` 并嵌入 `impl.Service`；`Service.Upload` 调用嵌入的上传能力、拿到文件信息后写素材记录（落库失败删除已写入的文件）。前台以后只需要上传文件、不需要素材库记录时，自己用 `mediaimpl.NewFileUploader` 装配（Storage + Limits）即可；需要不同规则（游客限流、转码、对象存储直传等）时替换 `media.FileUploader` 实现即可。
- 存储实现由 `upload.provider` 决定，当前仅 `local`；`repository/media.Storage` 是存储接口，后续可增加 OSS / COS / S3 实现，业务层不变。
- 本地实现保存到 `upload.dir`（默认 `uploads`），按 `yyyy/mm` 分目录，静态访问路径为 `upload.url_prefix`（默认 `/uploads`），支持 Range。
- 响应中的 `url` 由 `upload.public_base_url` 拼接完整地址；生产环境填官网域名 / CDN。
- 页面里的图片统一从「内置图形 / 图片素材库」选择，页面区块只保存素材 URL；富文本插图也先上传素材库再插入 URL。
- 删除会同步删除物理文件并软删除记录；文件删除失败时不删记录，可直接重试（不存在的文件视为删除成功）。当前未检查页面引用，后台确认框提示引用页面会失效。

### 4.5 菜单与站点设置

| 方法 | 路径 | 作用 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/v1/admin/menus` | 菜单列表 | `setting:edit` |
| GET | `/api/v1/admin/menus/:code` | 菜单详情与菜单项 | `setting:edit` |
| PUT | `/api/v1/admin/menus/:code` | 整体保存菜单项（排序、显隐、文案、链接） | `setting:edit` |
| GET | `/api/v1/admin/settings` | 站点设置 | `setting:edit` |
| PUT | `/api/v1/admin/settings` | 保存站点设置 | `setting:edit` |

菜单保存请求体：

```json
{
  "name": "顶部导航",
  "items": [
    {
      "label": { "zh-CN": "首页", "en": "Home" },
      "path": "/",
      "link_type": "page",
      "target": "_self",
      "visible": true,
      "children": []
    }
  ]
}
```

- `items` 的数组顺序即展示顺序，`sort` 可省略。
- 后台菜单项用 `label` 同时保存中英文文案；前台 `site/nav` 按 `locale` 取对应语言。
- 顶部菜单只有一级；页脚顶级项是栏目，`children` 是栏目内链接，最多两级。
- 菜单容器（`header` / `footer` / `footer_bottom`）不存在时保存会自动创建。

站点设置请求体：

```json
{
  "locale": "zh-CN",
  "site_name": "小鹊云",
  "company_name": "小鹊云科技",
  "tagline": "...",
  "contact": {
    "email": "...",
    "phone": "...",
    "address": "..."
  },
  "brand": {
    "logo": "...",
    "accent": "#E8503A",
    "background": "#F6F2EA",
    "dark_background": "#141518"
  },
  "header": {},
  "footer": {},
  "allow_index": true
}
```

说明：

- 站点设置按 `key + locale` 存储，中文 / 英文可分别提交。
- 管理后台的站点配置 key（`site_name` / `company_name` / `tagline` / `contact` / `brand` / `header` / `footer` / `copyright` / `icp` / `socials` / `default_locale` / `allow_index` / `seo`）作为全局项保存，值本身可携带中英文文案；其余 key 按 `locale` 保存。
- `GET /api/v1/admin/settings?locale=zh-CN` 返回与请求体相同的扁平结构，并附带 `locale`；全局项与语言级设置合并返回。
- 保存后发布缓存失效。

### 4.6 表单留言

| 方法 | 路径 | 作用 | 权限 |
| --- | --- | --- | --- |
| GET | `/api/v1/admin/submissions` | 留言列表，支持状态 / 表单 / 时间筛选 | `form:read` |
| GET | `/api/v1/admin/submissions/:id` | 留言详情 | `form:read` |
| PUT | `/api/v1/admin/submissions/:id/status` | 标记处理状态 | `form:edit` |
| GET | `/api/v1/admin/submissions/export` | 导出 CSV | `form:read` |
| GET | `/api/v1/admin/submissions/unread-count` | 待处理留言数量（后台侧边栏红点轮询） | `form:read` |

列表查询参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `status` | string | `new` / `processing` / `done` / `spam` |
| `form_key` | string | 表单标识，如 `contact` |
| `locale` | string | `zh-CN` / `en` |
| `keyword` | string | 匹配姓名 / 邮箱 / 电话 / 公司 / 留言 |
| `created_from` / `created_to` | int64 | 提交时间范围（Unix 秒） |
| `page` / `page_size` | int | 分页，`page_size` 最大 100 |

列表响应 `data`：

```json
{
  "items": [
    {
      "id": "1",
      "form_key": "contact",
      "locale": "zh-CN",
      "name": "张三",
      "email": "zhangsan@example.com",
      "phone": "13800000000",
      "company": "某某公司",
      "message": "想了解产品",
      "data": {},
      "status": "new",
      "ip": "203.0.113.10",
      "user_agent": "Mozilla/5.0 ...",
      "created_at": 1792375200,
      "updated_at": 1792375200
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20,
  "stats": {
    "total": 12,
    "new": 5,
    "processing": 3,
    "done": 4,
    "spam": 0
  }
}
```

说明：

- `stats` 统计不受 `status` 影响，其余筛选条件生效，用于后台顶部统计卡片。
- `name` / `email` / `phone` / `company` / `message` 从 `data` 提取的常用字段，完整内容以 `data` 为准。
- 来源只记录语言（`zh-CN` / `en`），不记录具体来源页面。
- 标记状态：`PUT /api/v1/admin/submissions/:id/status`，请求体 `{"status":"processing"}`，返回更新后的留言详情。
- 导出：`text/csv; charset=utf-8`，带 UTF-8 BOM，`Content-Disposition: attachment`；筛选参数与列表一致，最多导出 10000 条。
- 待处理数量：`GET /api/v1/admin/submissions/unread-count` 返回 `data: { "count": 5 }`，只统计 `status = new`，不带列表与其余状态；后台侧边栏红点每 30 秒轮询一次（标签页不可见时暂停，切回立即刷新）。

### 4.7 账号、角色与审计

| 方法 | 路径 | 作用 | 权限 |
| --- | --- | --- | --- |
| GET / POST | `/api/v1/admin/users` | 账号列表 / 新建 | `user:manage` |
| PUT / DELETE | `/api/v1/admin/users/:id` | 修改 / 停用账号 | `user:manage` |
| GET / POST | `/api/v1/admin/roles` | 角色列表 / 新建 | `user:manage` |
| PUT | `/api/v1/admin/roles/:id/permissions` | 更新角色权限 | `user:manage` |
| GET | `/api/v1/admin/audit-logs` | 审计日志，支持操作人 / 动作 / 时间 | `audit:read` |

---

## 5. 缓存与失效

| 数据 | 缓存建议 | 失效时机 |
| --- | --- | --- |
| 页面渲染结果 | 进程内 LRU + Redis，key 含 `path + locale + version` | 页面发布 |
| 内容列表与详情 | Redis，key 含 `type + locale + filters` | 内容发布 / 修改 / 删除 |
| 站点设置与导航 | 进程内缓存 + Redis | 设置保存 |
| 媒体 URL | CDN 缓存 | 文件替换时使用新 key |

多副本部署时必须使用 Redis 共享缓存，避免各实例数据不一致。

---

## 6. 实施状态与落地顺序

已实现：

- `GET /ping`
- `app/core/response` 统一响应
- `app/core/config` YAML + env 配置（数据库用户名 / 密码、`JWT_SECRET` 来自 `.env`）
- `app/core/application` MySQL / Redis 初始化与关闭
- `app/core/controller` 公共控制器
- `app/core/bootstrap` 启动装配
- `main.go` 启动与优雅退出
- `POST /api/v1/admin/auth/login`：bcrypt 校验、HS256 JWT Access Token、刷新令牌写入 Redis
- `app/services/auth`：认证接口、密码哈希、JWT / 刷新令牌工具
- `app/services/auth/impl`：`AdminLogin` 数据库登录实现
- `app/repository/user`：后台账号查询（`FindByUsername` / `FindByID`）
- `app/repository/session`：Redis 登录会话（保存、按哈希查询、Lua 原子轮换、吊销）
- `middleware/admin_auth.go`：管理后台 Bearer JWT 认证中间件
- `hertz-contrib/logger/accesslog`：HTTP 访问日志，经 `hlog.CtxInfof` 输出

业务接口（2026-10-22）：

- 后台认证：`POST /admin/auth/refresh`、`POST /admin/auth/logout`
- 管理员资料与改密：`PUT /admin/user/profile`、`PUT /admin/user/password`（改密后吊销该账号全部刷新令牌）
- 站点编排：页面列表 / 新建 / 详情 / 保存草稿 / 删除 / 翻译配对 / 发布 / 下架，菜单列表 / 详情 / 整体保存，站点设置读取 / 保存
- 表单留言：列表 / 详情 / 状态标记 / CSV 导出
- 媒体素材：上传 / 列表 / 详情 / 重命名 / 单个删除 / 批量软删除（`xqyun_media`，本地磁盘 + Range 静态访问）
- 前台：`GET /frontend/render/page`、`GET /frontend/site/config`、`GET /frontend/site/nav`、`POST /frontend/forms/:key/submissions`

尚未实现：

- `GET /admin/me`
- 内容库与分类（`render/content`、`contents`、`categories`）
- RBAC 角色权限与审计日志
- 服务端图形验证码、登录限流、表单提交 IP 频率限制

建议实现顺序：

1. 认证与用户 / 角色中间件
2. 页面 CRUD、草稿保存、直接发布
3. `render/page` 聚合接口（前台最先依赖）
4. 内容与分类 CRUD、`contents` / `render/content`
5. 媒体上传
6. 菜单与站点设置
7. 表单提交与后台留言
8. 审计日志、sitemap / robots、缓存

---

## 7. 代码落点

```
service/
├── main.go                        # 启动入口
├── router.go                      # 自定义路由
├── router_gen.go                  # hz 生成路由
└── app/
    ├── core/
    │   ├── application/           # DB / Redis 接口与实现
    │   ├── bootstrap/             # 启动装配
    │   ├── config/                # YAML + env 配置
    │   ├── controller/            # BaseController
    │   └── response/              # 统一响应
    ├── handler/                   # 控制器与 handler
    └── router/                    # hz 生成的路由注册
```

新增接口时：

1. 在 `app/handler/` 新建控制器，嵌入 `*controller.BaseController`。
2. 在 `router.go` 的 `customizedRegister` 注册路由。
3. 用 `response.Success` / `response.Fail` 输出统一结构。
4. 业务逻辑放到 service 层，handler 只做参数解析与响应组装。
