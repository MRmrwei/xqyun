# 小鹊云 MySQL 数据库表设计

> 数据库：`xqyun`（与 `service/config.yaml` 一致）
> 状态：已落地。业务表由 `migration/` 下的 golang-migrate 脚本创建，建库脚本见 `deploy/init.sql`；表结构以迁移文件为准。

## 1. 基本约定

| 项 | 约定 |
| --- | --- |
| 数据库版本 | MySQL 8.0+ |
| 存储引擎 | InnoDB |
| 字符集 / 排序规则 | `utf8mb4` / `utf8mb4_unicode_ci` |
| 主键 | `BIGINT UNSIGNED AUTO_INCREMENT` |
| 翻译组 ID | `CHAR(26)`，应用侧生成 ULID，同一组的中英文记录共用 |
| 时间类型 | `BIGINT UNSIGNED`，Unix 时间戳（秒） |
| 布尔值 | `TINYINT`，1 = 是，0 = 否 |
| 命名 | 表名、字段名统一 `snake_case`；表名统一加 `xqyun_` 前缀，例如 `xqyun_users` |
| 软删除 | `deleted_at = 0` 表示未删除 |
| 语言 | `locale` 取 `zh-CN` / `en`；前台内部的 `zh` 在接口层映射为 `zh-CN` |
| 结构化数据 | 区块、SEO、设置值等使用 `JSON` 列，避免为每种区块建表 |

### 1.1 公共字段

除关联表和日志表外，业务表统一包含：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 主键，自增 |
| `created_at` | `BIGINT UNSIGNED` | 创建时间（Unix 秒），由应用写入 |
| `updated_at` | `BIGINT UNSIGNED` | 更新时间（Unix 秒），由应用写入 |
| `deleted_at` | `BIGINT UNSIGNED` | 软删除时间（Unix 秒）；0 表示未删除 |

时间字段统一由应用写入秒级时间戳（Go 中 `time.Now().Unix()`），不使用 `DATETIME` 和数据库默认时间。查询展示时可用 `FROM_UNIXTIME(created_at)` 转成可读时间；`0` 表示未设置、未发布或未删除。

> 软删除与唯一索引的注意点：如果唯一键只包含业务字段（例如 `locale + path`），删除后的路径不能立即被新页面复用。需要复用时，先恢复原记录，或把原记录的真实路径改名后再创建。若后续确实需要“删除后复用路径”，再引入 `deleted_at` 哨兵值或生成列参与唯一索引。

---

## 2. 表总览

| 表名 | 作用 | 关键关系 |
| --- | --- | --- |
| `xqyun_users` | 后台账号 | 通过 `xqyun_casbin_rule` 的 `g` 规则关联角色 |
| `xqyun_roles` | 角色定义（展示信息） | 权限关系在 `xqyun_casbin_rule` 的 `p` 规则中 |
| `xqyun_casbin_rule` | Casbin 策略表 | `p`：角色 → URL 模板 → HTTP 方法；`g`：用户 → 角色 |
| `xqyun_pages` | 页面主表，保存草稿与当前线上内容 | `translation_group_id` 配对中英文；`published_content` 为线上内容 |
| `xqyun_contents` | 新闻 / 产品 / 案例统一内容表 | 保存草稿与当前线上内容；`published_content` 为线上快照 |
| `xqyun_categories` | 内容分类 | `translation_group_id`、`parent_id` 自关联 |
| `xqyun_media` | 媒体库 | 被 `xqyun_contents.cover_media_id` 等引用 |
| `xqyun_menus` | 菜单容器（顶部 / 底部） | 通过 `xqyun_menu_items.menu_id` 关联 |
| `xqyun_menu_items` | 菜单项与页脚链接 | `parent_id` 自关联，支持栏目 + 链接 |
| `xqyun_settings` | 全站设置，按 key + locale 存储 | `value` 为 JSON |
| `xqyun_form_submissions` | 表单留言 | `form_key` / `locale` |
| `xqyun_redirects` | 301 / 302 跳转规则 | 无 |
| `xqyun_audit_logs` | 后台操作审计 | `user_id` 可选 |
| `xqyun_publish_records` | 页面 / 内容发布记录 | `target_type` + `target_id` |

---

## 3. 详细表结构

### 3.1 `xqyun_users` — 后台账号

保存后台登录账号。密码只存 bcrypt 哈希，不存明文。

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `username` | `VARCHAR(64)` | 否 | — | 登录名，全局唯一 |
| `password_hash` | `VARCHAR(255)` | 否 | — | bcrypt 哈希 |
| `name` | `VARCHAR(100)` | 否 | — | 显示名称 |
| `email` | `VARCHAR(255)` | 是 | `NULL` | 邮箱，用于通知 |
| `mobile` | `VARCHAR(32)` | 是 | `NULL` | 手机号 |
| `avatar` | `MEDIUMTEXT` | 是 | `NULL` | 头像：data URL 或外部地址 |
| `status` | `TINYINT` | 否 | `1` | 1 启用，0 禁用 |
| `last_login_at` | `BIGINT UNSIGNED` | 否 | `0` | 最近登录时间 |
| `last_login_ip` | `VARCHAR(45)` | 是 | `NULL` | 最近登录 IP，兼容 IPv6 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |
| `deleted_at` | `BIGINT UNSIGNED` | 否 | `0` | 软删除 |

索引：

- `UNIQUE KEY uk_username (username)`
- `KEY idx_mobile (mobile)`
- `KEY idx_status (status)`

---

### 3.2 `xqyun_roles` — 角色

只保存角色的展示信息。用户与角色、角色与权限的**关系不存本表**，由 Casbin 写入 `xqyun_casbin_rule`（见 3.3）。`code` 创建后不可修改（策略按 `code` 关联）；`is_system = 1` 的内置角色不允许删除。

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `code` | `VARCHAR(50)` | 否 | — | 角色编码，唯一，如 `editor`、`publisher`；写入后不可改 |
| `name` | `VARCHAR(100)` | 否 | — | 角色名称 |
| `description` | `VARCHAR(255)` | 是 | `NULL` | 说明 |
| `is_system` | `TINYINT` | 否 | `0` | 是否内置角色，内置角色不允许删除 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |

索引：

- `UNIQUE KEY uk_code (code)`

内置角色：

| code | 名称 | 权限 |
| --- | --- | --- |
| `super_admin` | 超级管理员 | 全部权限（`/api/v1/admin/*` + `*`，一条通配） |
| `publisher` | 发布管理员 | 编辑 + 发布 / 下架 |
| `editor` | 内容编辑 | 编辑，不能发布 |
| `viewer` | 只读 | 仅查看 |

完整角色策略清单见 [权限控制方案](./权限控制方案.md) 5.2 与第 8 节。

---

### 3.3 `xqyun_casbin_rule` — Casbin 策略

后端权限判定使用 Casbin（`casbin/v3` + `gorm-adapter/v3`），策略存本表。**业务代码不要直接 SQL 增删规则**，统一走 Casbin API，保证进程内策略与数据库一致。

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `ptype` | `VARCHAR(100)` | 否 | `''` | 规则类型：`p` 角色→URL+方法，`g` 用户→角色 |
| `v0` | `VARCHAR(100)` | 否 | `''` | `p`：角色 code；`g`：用户 ID |
| `v1` | `VARCHAR(100)` | 否 | `''` | `p`：URL 路由模板；`g`：角色 code |
| `v2` | `VARCHAR(100)` | 否 | `''` | `p`：HTTP 方法（`GET` / `POST` / `PUT` / `DELETE` / `*`） |
| `v3` | `VARCHAR(100)` | 否 | `''` | 预留（如 domain） |
| `v4` | `VARCHAR(100)` | 否 | `''` | 预留 |
| `v5` | `VARCHAR(100)` | 否 | `''` | 预留 |

索引：

- `PRIMARY KEY (id)`
- `UNIQUE KEY idx_xqyun_casbin_rule (ptype, v0, v1, v2, v3, v4, v5)`

示例：

| ptype | v0 | v1 | v2 | 含义 |
| --- | --- | --- | --- | --- |
| `p` | `super_admin` | `/api/v1/admin/*` | `*` | 超管拥有全部后台接口 |
| `p` | `publisher` | `/api/v1/admin/pages/:id` | `PUT` | 发布管理员可保存页面草稿 |
| `p` | `editor` | `/api/v1/admin/pages/:id` | `GET` | 内容编辑可查看页面详情 |
| `g` | `1` | `super_admin` | | 用户 1（初始 admin）属于超管 |

策略按「角色 + URL 路由模板 + HTTP 方法」存储，`v1` 必须与 `c.FullPath()` 完全一致（参数占位符写法也要一致）；超管用 `v1 = /api/v1/admin/*`、`v2 = *` 一条通配覆盖全部后台接口。完整模型、角色策略清单与前端对接方式见 [权限控制方案](./权限控制方案.md)。

> 建表走 golang-migrate（表结构以迁移文件为准）。`gorm-adapter v3.41.0` 构造时默认 AutoMigrate：初始化前调用 `gormadapter.TurnOffAutoMigrate(db)` 关闭，再用 `gormadapter.NewAdapterByDBUseTableName(db, "", "xqyun_casbin_rule")`。adapter 默认索引名为 `idx_` + 表名，与上面的 `idx_xqyun_casbin_rule` 一致。

---

### 3.4 登录会话（Redis，不建表）

管理后台登录会话由 `hertz-contrib/sessions` 的 Redis store 保存，不落 MySQL。Redis key 形如：

```
session_{sessionID}
```

value 为编码后的会话数据（`admin_user_id` / `admin_username` 等），TTL 默认 12 小时；浏览器 Cookie `xqyun_admin_session` 只保存加密签名的会话 ID。退出登录或改密时删除对应 key。

> JWT 刷新令牌（后续前台用户登录预留）：如需启用，使用 `xqyun:refresh:{token_type}:{sha256}`，同样不建表；TTL 与 `refresh_ttl` 对齐，刷新时用 Lua 脚本原子轮换，退出时删除 key。

- 管理后台 Session 的 Redis key 由 hertz-contrib/sessions 管理，默认 `session_{sessionID}`，过期自动删除；
- Redis 必须启用（`redis.enabled=true`），否则服务启动失败。

---

### 3.5 `xqyun_pages` — 页面主表

一个页面记录 = 一种语言 + 一个路径 + 当前草稿与当前线上内容。中英文页面通过 `translation_group_id` 配对，各自独立发布。

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `translation_group_id` | `CHAR(26)` | 否 | — | 翻译组 ID；中英文记录共用 |
| `locale` | `VARCHAR(10)` | 否 | — | `zh-CN` / `en` |
| `path` | `VARCHAR(255)` | 否 | — | 页面路径，如 `/about`、`/en/about-us` |
| `name` | `VARCHAR(200)` | 否 | — | 后台显示名称 |
| `status` | `VARCHAR(20)` | 否 | `draft` | `draft` / `published` |
| `draft_content` | `JSON` | 否 | — | 当前草稿：区块列表 + SEO |
| `published_content` | `JSON` | 是 | `NULL` | 最近一次发布的线上内容：区块 + SEO；未发布为空 |
| `version` | `INT UNSIGNED` | 否 | `0` | 发布序号，每次发布 +1，用于缓存 / ETag |
| `created_by` | `BIGINT UNSIGNED` | 是 | `NULL` | 创建人 |
| `updated_by` | `BIGINT UNSIGNED` | 是 | `NULL` | 最后修改人 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |
| `deleted_at` | `BIGINT UNSIGNED` | 否 | `0` | 软删除 |

索引：

- `UNIQUE KEY uk_locale_path (locale, path)`
- `KEY idx_group (translation_group_id)`

`draft_content` 结构：

```json
{
  "seo": {
    "title": "关于我们 - 小鹊云",
    "description": "...",
    "keywords": ["..."]
  },
  "blocks": [
    {
      "id": "b_01",
      "type": "hero",
      "props": {
        "title": "...",
        "subtitle": "...",
        "image": "..."
      }
    }
  ]
}
```

说明：

- 区块只存 `type` 与 `props`，不存渲染后的 HTML。
- 未发布的页面只有 `draft_content`，`published_content` 为空。
- 发布时用 `draft_content` 覆盖 `published_content`，`version + 1`，并更新 `status` / `published_at`。
- **不保留历史版本**：旧的 `published_content` 被直接覆盖，不支持回滚。
- 下架 = `status` 改为 `draft`、`published_content` 置空，草稿保留，可再次发布。
- 同一 `translation_group_id` 下，各语言独立发布：中文发布不会改变英文记录；英文未发布时 `published_content` 为空、`status = draft`，前台按 `locale` 查询时返回 404。

---

### 3.6 `xqyun_contents` — 新闻 / 产品 / 案例

统一内容表，用 `type` 区分类型，用 `locale` 区分语言。

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `translation_group_id` | `CHAR(26)` | 否 | — | 翻译组 ID |
| `type` | `VARCHAR(20)` | 否 | — | `news` / `product` / `case` |
| `locale` | `VARCHAR(10)` | 否 | — | `zh-CN` / `en` |
| `slug` | `VARCHAR(255)` | 否 | — | URL 标识，同语言同类型内唯一 |
| `title` | `VARCHAR(255)` | 否 | — | 标题 |
| `summary` | `VARCHAR(500)` | 是 | `NULL` | 摘要 / 列表页文案 |
| `cover_media_id` | `BIGINT UNSIGNED` | 是 | `NULL` | 封面媒体 |
| `category_id` | `BIGINT UNSIGNED` | 是 | `NULL` | 分类 |
| `body_html` | `MEDIUMTEXT` | 是 | `NULL` | 富文本渲染结果（已清洗） |
| `body_json` | `JSON` | 是 | `NULL` | 富文本编辑器结构，便于再次编辑 |
| `seo` | `JSON` | 是 | `NULL` | `{title, description, keywords}` |
| `extra` | `JSON` | 是 | `NULL` | 类型专属字段，见下表 |
| `status` | `VARCHAR(20)` | 否 | `draft` | `draft` / `published` |
| `featured` | `TINYINT` | 否 | `0` | 是否置顶 / 推荐 |
| `read_minutes` | `INT UNSIGNED` | 否 | `0` | 新闻预计阅读分钟数 |
| `sort` | `INT` | 否 | `0` | 列表排序，越小越靠前 |
| `published_at` | `BIGINT UNSIGNED` | 否 | `0` | 最近发布时间；0 表示从未发布 |
| `published_content` | `JSON` | 是 | `NULL` | 最近一次发布的线上内容快照；未发布为空 |
| `version` | `INT UNSIGNED` | 否 | `0` | 发布序号，每次发布 +1 |
| `created_by` | `BIGINT UNSIGNED` | 是 | `NULL` | 创建人 |
| `updated_by` | `BIGINT UNSIGNED` | 是 | `NULL` | 最后修改人 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |
| `deleted_at` | `BIGINT UNSIGNED` | 否 | `0` | 软删除 |

索引：

- `UNIQUE KEY uk_locale_type_slug (locale, type, slug)`
- `KEY idx_group (translation_group_id)`
- `KEY idx_type_status (type, status, published_at)`
- `KEY idx_category (category_id)`
- `KEY idx_featured_sort (type, featured, sort)`

说明：`xqyun_contents` 行保存当前草稿；`published_content` 保存最近一次发布的内容。发布时直接覆盖，不保留历史版本。前台只读取 `published_content`，因此编辑草稿不会影响线上，必须重新发布才生效。下架时把 `status` 改为 `draft` 并清空 `published_content`，草稿保留。

`extra` 按 `type` 取值：

| type | extra 字段 | 说明 |
| --- | --- | --- |
| `news` | `category` | 分类名称快照，便于列表直接展示 |
| `product` | `label`、`description`、`bullets` | 产品标签、详细描述、要点数组 |
| `case` | `client`、`industry`、`metric`、`metric_suffix`、`metric_label` | 客户案例指标 |

示例：

```json
{
  "label": "AI 客服",
  "description": "面向售前咨询的智能问答",
  "bullets": ["多轮对话", "知识库接入", "人工接管"]
}
```

---

### 3.7 `xqyun_categories` — 内容分类

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `translation_group_id` | `CHAR(26)` | 否 | — | 翻译组 ID |
| `type` | `VARCHAR(20)` | 否 | — | 对应内容类型 |
| `locale` | `VARCHAR(10)` | 否 | — | 语言 |
| `slug` | `VARCHAR(255)` | 否 | — | 分类标识 |
| `name` | `VARCHAR(100)` | 否 | — | 分类名称 |
| `description` | `VARCHAR(255)` | 是 | `NULL` | 分类说明 |
| `parent_id` | `BIGINT UNSIGNED` | 是 | `NULL` | 父分类，支持一层或多层 |
| `sort` | `INT` | 否 | `0` | 排序 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |
| `deleted_at` | `BIGINT UNSIGNED` | 否 | `0` | 软删除 |

索引：

- `UNIQUE KEY uk_locale_type_slug (locale, type, slug)`
- `KEY idx_group (translation_group_id)`
- `KEY idx_parent_sort (parent_id, sort)`

---

### 3.8 `xqyun_media` — 媒体库

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `filename` | `VARCHAR(255)` | 否 | — | 原始文件名 |
| `storage_key` | `VARCHAR(500)` | 否 | — | 对象存储 key，唯一 |
| `url` | `VARCHAR(1000)` | 否 | — | 可访问 URL |
| `mime` | `VARCHAR(100)` | 否 | — | MIME 类型 |
| `size_bytes` | `BIGINT UNSIGNED` | 否 | `0` | 文件大小 |
| `width` | `INT UNSIGNED` | 是 | `NULL` | 图片宽度 |
| `height` | `INT UNSIGNED` | 是 | `NULL` | 图片高度 |
| `sha256` | `CHAR(64)` | 是 | `NULL` | 内容哈希，用于去重 |
| `alt` | `JSON` | 是 | `NULL` | 多语言 alt：`{"zh-CN":"...","en":"..."}` |
| `uploaded_by` | `BIGINT UNSIGNED` | 是 | `NULL` | 上传人 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |
| `deleted_at` | `BIGINT UNSIGNED` | 否 | `0` | 软删除 |

索引：

- `UNIQUE KEY uk_storage_key (storage_key)`
- `KEY idx_sha256 (sha256)`
- `KEY idx_mime (mime)`

---

### 3.9 `xqyun_menus` — 菜单容器

一个菜单代表顶部导航或底部导航。

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `code` | `VARCHAR(50)` | 否 | — | 菜单编码，唯一，如 `header`、`footer` |
| `name` | `VARCHAR(100)` | 否 | — | 后台显示名称 |
| `description` | `VARCHAR(255)` | 是 | `NULL` | 说明 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |

索引：

- `UNIQUE KEY uk_code (code)`

---

### 3.10 `xqyun_menu_items` — 菜单项

同一张表同时承载顶部菜单项和页脚栏目 / 链接。

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `menu_id` | `BIGINT UNSIGNED` | 否 | — | 所属菜单 |
| `parent_id` | `BIGINT UNSIGNED` | 是 | `NULL` | 父项；页脚中顶级项是栏目，子项是链接 |
| `label` | `JSON` | 否 | — | 多语言文案：`{"zh-CN":"...","en":"..."}` |
| `link_type` | `VARCHAR(20)` | 否 | `page` | `page` / `content` / `external` / `none` |
| `link_value` | `VARCHAR(500)` | 是 | `NULL` | 页面路径、内容 slug 或外链 |
| `target` | `VARCHAR(20)` | 否 | `_self` | `_self` / `_blank` |
| `sort` | `INT` | 否 | `0` | 同级排序 |
| `visible` | `TINYINT` | 否 | `1` | 是否显示 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |

索引：

- `KEY idx_menu_parent_sort (menu_id, parent_id, sort)`

说明：

- 顶部菜单：`parent_id` 为空；本期只支持一级。
- 页脚：顶级项表示栏目，`label` 是栏目标题；子项是栏目内链接。
- 也可以把页脚底部链接放在 `code = footer_bottom` 的菜单里。
- **职责边界**：`xqyun_menus` / `xqyun_menu_items` 只负责链接结构；公告条、顶部按钮、语言入口、吸顶等视觉与行为配置在 `settings.header`，页脚订阅区、联系入口、大字标等在 `settings.footer`。前台渲染时把两者合并。

---

### 3.11 `xqyun_settings` — 全站设置

按 `key + locale` 存储，值统一用 JSON。

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `key` | `VARCHAR(100)` | 否 | — | 设置项 |
| `locale` | `VARCHAR(10)` | 否 | `''` | 语言；空字符串表示全局配置 |
| `value` | `JSON` | 否 | — | 设置值 |
| `updated_by` | `BIGINT UNSIGNED` | 是 | `NULL` | 最后修改人 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |

索引：

- `UNIQUE KEY uk_key_locale (key, locale)`

> 这里不用 `NULL` 表示全局，因为 MySQL 唯一索引中多个 `NULL` 不算冲突；用空字符串 `''` 才能保证每个 key 只有一条全局记录。

常用 key：

| key | value 结构 | 说明 |
| --- | --- | --- |
| `site_name` | `{"zh-CN":"小鹊云","en":"Magpie Cloud"}` | 站点名称 |
| `company_name` | `{"zh-CN":"...","en":"..."}` | 公司名称 |
| `tagline` | `{"zh-CN":"...","en":"..."}` | 品牌标语 |
| `contact` | `{"email":"...","phone":"...","phone_en":"...","address":{...},"work_hours":{...}}` | 联系方式 |
| `brand` | `{"logo":"...","accent":"#E8503A","background":"#F6F2EA","dark_background":"#141518"}` | 品牌视觉 |
| `header` | `{"show_language":true,"sticky":true,"show_announcement":true,"announcement":{...},"ctas":[...]}` | 顶部导航设置 |
| `footer` | `{"show_newsletter":true,"show_contact":true,"cta":{...},"show_wordmark":true,"status_text":{...}}` | 页脚行为与文案；栏目和链接来自 `xqyun_menu_items` |
| `socials` | `[{"name":"WeChat","handle":"..."}]` | 社交链接 |
| `seo` | `{"allow_index":true,"analytics":"","head_code":""}` | 收录、统计与自定义 head |
| `copyright` | `{"zh-CN":"...","en":"..."}` | 版权文案 |
| `icp` | `"..."` | 备案号 |

> `settings.header` / `settings.footer` 只保存行为与文案；菜单项、页脚栏目和底部链接统一放在 `xqyun_menu_items`，避免两处都能改链接导致数据不一致。

`header` 与 `footer` 的完整结构可参考 `frontend/domain/types.ts` 中的 `HeaderSettings` / `FooterSettings`。

---

### 3.12 `xqyun_form_submissions` — 表单留言

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `form_key` | `VARCHAR(50)` | 否 | — | 表单标识，如 `contact` |
| `locale` | `VARCHAR(10)` | 是 | `NULL` | 提交时的语言（来源只记录中文 / 英文） |
| `name` | `VARCHAR(100)` | 是 | `NULL` | 姓名 |
| `email` | `VARCHAR(255)` | 是 | `NULL` | 邮箱 |
| `phone` | `VARCHAR(50)` | 是 | `NULL` | 电话 |
| `company` | `VARCHAR(200)` | 是 | `NULL` | 公司 |
| `message` | `TEXT` | 是 | `NULL` | 留言正文 |
| `data` | `JSON` | 否 | — | 完整表单数据，兼容后续新增字段 |
| `status` | `VARCHAR(20)` | 否 | `new` | `new` / `processing` / `done` / `spam` |
| `ip` | `VARCHAR(45)` | 是 | `NULL` | 提交 IP |
| `user_agent` | `VARCHAR(500)` | 是 | `NULL` | 浏览器信息 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 提交时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 处理时间 |

说明：`name` / `email` / `phone` / `company` / `message` 是从 `data` 中提取的常用字段，便于列表筛选和导出；完整内容仍以 `data` 为准。

---

### 3.13 `xqyun_redirects` — 跳转规则

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `locale` | `VARCHAR(10)` | 否 | `''` | 语言；空表示全语言生效 |
| `from_path` | `VARCHAR(255)` | 否 | — | 旧路径 |
| `to_path` | `VARCHAR(1000)` | 否 | — | 新路径或完整 URL |
| `status_code` | `SMALLINT` | 否 | `301` | 301 / 302 |
| `enabled` | `TINYINT` | 否 | `1` | 是否启用 |
| `note` | `VARCHAR(255)` | 是 | `NULL` | 说明 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 创建时间 |
| `updated_at` | `BIGINT UNSIGNED` | 否 | `0` | 更新时间 |

索引：

- `UNIQUE KEY uk_locale_from_path (locale, from_path)`

---

### 3.14 `xqyun_audit_logs` — 操作审计

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `user_id` | `BIGINT UNSIGNED` | 是 | `NULL` | 操作人；系统任务可为空 |
| `action` | `VARCHAR(100)` | 否 | — | 动作，如 `page.publish` |
| `target_type` | `VARCHAR(50)` | 是 | `NULL` | 对象类型，如 `page`、`content` |
| `target_id` | `VARCHAR(64)` | 是 | `NULL` | 对象 ID |
| `detail` | `JSON` | 是 | `NULL` | 变更详情 |
| `ip` | `VARCHAR(45)` | 是 | `NULL` | 操作 IP |
| `user_agent` | `VARCHAR(500)` | 是 | `NULL` | 浏览器信息 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 操作时间 |

索引：

- `KEY idx_user_created (user_id, created_at)`
- `KEY idx_target (target_type, target_id)`
- `KEY idx_action_created (action, created_at)`

---

### 3.15 `xqyun_publish_records` — 发布记录

| 字段 | 类型 | 空 | 默认 | 说明 |
| --- | --- | --- | --- | --- |
| `id` | `BIGINT UNSIGNED` | 否 | 自增 | 主键 |
| `target_type` | `VARCHAR(20)` | 否 | — | 发布对象：`page` / `content` |
| `target_id` | `BIGINT UNSIGNED` | 否 | — | 对象 ID（页面或内容） |
| `locale` | `VARCHAR(10)` | 否 | — | 发布语言 |
| `status` | `VARCHAR(20)` | 否 | `success` | `success` / `failed` |
| `message` | `VARCHAR(500)` | 是 | `NULL` | 失败原因或备注 |
| `operator_id` | `BIGINT UNSIGNED` | 是 | `NULL` | 操作人 |
| `created_at` | `BIGINT UNSIGNED` | 否 | `0` | 发布时间 |

索引：

- `KEY idx_target_created (target_type, target_id, created_at)`

---

## 4. 关系与数据流

```
xqyun_users ──< xqyun_casbin_rule(g) >── xqyun_roles ──< xqyun_casbin_rule(p)

xqyun_pages(zh-CN) ── translation_group_id ── xqyun_pages(en)

xqyun_contents(zh-CN) ── translation_group_id ── xqyun_contents(en)
xqyun_contents ──> xqyun_categories
xqyun_contents ──> xqyun_media

xqyun_menus ──< xqyun_menu_items（parent_id 自关联）

xqyun_settings：按 key + locale 读取
```

页面发布流程：

1. 编辑写入 `xqyun_pages.draft_content`
2. 发布时用 `draft_content` 覆盖 `published_content`，`version + 1`
3. 更新 `status = published`、`published_at`
4. 写入 `xqyun_publish_records`
5. 前台只读取 `published_content`，不保留历史版本

内容发布流程：

1. 编辑 `xqyun_contents` 草稿字段
2. 发布时用当前字段生成 `published_content`，`version + 1`
3. 更新 `status = published`、`published_at`
4. 写入 `xqyun_publish_records`（`target_type = content`）
5. 前台只读取 `published_content`，且 `deleted_at = 0`

---

## 5. 建表顺序

按依赖关系创建：

1. `xqyun_users`
2. `xqyun_roles`
3. `xqyun_casbin_rule`
4. `xqyun_media`
5. `xqyun_categories`
6. `xqyun_contents`
7. `xqyun_pages`
8. `xqyun_menus`
9. `xqyun_menu_items`
10. `xqyun_settings`
11. `xqyun_form_submissions`
12. `xqyun_redirects`
13. `xqyun_audit_logs`
14. `xqyun_publish_records`

外键不是必须。`published_content` 是 JSON 快照，不引用额外表。

---

## 6. 后续扩展

- 内容审核流：增加 `content_reviews` 或复用 `xqyun_publish_records` 扩展 `target_type`。
- 多站点：增加 `sites` 表，并在页面、内容、设置上加 `site_id`。
- 更多语言：`locale` 已是字符串，新增语言只需加值，不需要改表。
- 媒体引用检查：增加 `media_references` 表，或在删除前扫描 JSON 字段。
- 站内搜索：后续可接 MySQL 全文索引或独立搜索服务，不影响现有表结构。
