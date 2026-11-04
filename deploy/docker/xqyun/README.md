# 小鹊云 Docker 部署包

这个目录可以**整个上传**到服务器，不需要源码，也不需要 `docker build`。

## 上传

把本目录内容上传到服务器：

```
/xqyun/
```

最终结构：

```
/xqyun/
├── docker-compose.yml             # 业务 + 监控（监控用 profile jk 控制）
├── monitoring/                    # Prometheus / Grafana 配置（可选）
│   ├── prometheus.yml
│   └── grafana/
│       ├── provisioning/          # 数据源与看板自动装载
│       └── dashboards/            # 看板 JSON
├── .env.example
├── init.sql
├── migration/
├── data/
│   ├── nginx/conf/default.conf
│   ├── release/
│   │   ├── admin/                 # 管理后台静态文件
│   │   ├── frontend/.output/      # 官网 Nuxt SSR
│   │   └── backend/
│   │       ├── xqyun              # Linux Go 二进制
│   │       └── config.yaml
│   ├── uploads/                   # 素材（自动生成）
│   ├── logs/                      # 后端日志（自动生成）
│   └── monitoring/                # Prometheus / Grafana 数据（自动生成）
└── README.md
```

## 启动

```bash
cd /xqyun
chmod +x data/release/backend/xqyun
mkdir -p data/uploads data/logs
cp .env.example .env
# 编辑 .env：MySQL 密码、JWT_SECRET、UPLOAD_PUBLIC_BASE_URL
# 当前 Redis 不设置密码（REDIS_PASSWORD 留空）
docker compose up -d
```

不需要指定 `-f`，默认使用同目录的 `docker-compose.yml`。

## 访问

- 官网：`http://服务器IP/`
- 管理后台：`http://服务器IP/admin/`
- 健康检查：`http://服务器IP/ping`

初始管理员 `admin` / `123456`，登录后请立即修改密码。

## 监控（可选）

Go 后端在 `/metrics` 暴露 Prometheus 指标（`xqyun_http_requests_total`、`xqyun_http_request_duration_seconds`、`xqyun_business_responses_total`）。Prometheus / Grafana 写在主 `docker-compose.yml` 里，用 `profiles: ["jk"]` 标成可选：不加 `--profile jk` 就不会启动，业务照常运行。

### 首次启动

```bash
cd /xqyun

# 1. 运行时数据目录，并交给容器内进程（Prometheus 65534，Grafana 472）
mkdir -p data/monitoring/prometheus data/monitoring/grafana
chown -R 65534:65534 data/monitoring/prometheus
chown -R 472:472     data/monitoring/grafana

# 2. 在 .env 里补 GRAFANA_PASSWORD

# 3. 启动（业务 + 监控）
docker compose --profile jk up -d

# 不带监控、只起业务：
# docker compose up -d
```

### 访问 Grafana

Grafana 只绑定宿主机回环 `127.0.0.1:3001`，公网不可达，需要 SSH 隧道：

```bash
ssh -L 3001:127.0.0.1:3001 用户名@服务器IP
# 浏览器打开 http://127.0.0.1:3001，账号 admin，密码为 .env 里的 GRAFANA_PASSWORD
```

数据源与看板由 `monitoring/grafana/provisioning/` 自动装载。

### 日常操作

```bash
# 只重启监控
docker compose restart prometheus grafana

# 改完 monitoring/prometheus.yml 后
docker compose restart prometheus

# 停止监控（不要用 down：会连同 MySQL / Redis / 业务容器一起停掉）
docker compose stop prometheus grafana
```

### 安全约束

- `/metrics` 不挂在 `/api/v1` 下，Nginx 也不转发，只有 Compose 内网能访问。
- backend 端口已改为 `127.0.0.1:8888:8888`，**不要改回 `8888:8888`**，否则 `/metrics` 与后端接口都会暴露到公网。
- Prometheus / Grafana 的端口都只绑 `127.0.0.1`，不要改成 `0.0.0.0`。

## 连接数据库（图形客户端）

MySQL 只映射到宿主机回环地址，公网无法直接访问，需要通过 SSH 隧道：

```text
数据流：Navicat → SSH 加密 → 服务器 127.0.0.1:3306 → mysql 容器:3306
```

先确认映射已生效（应只显示 `127.0.0.1:3306`）：

```bash
docker compose ps mysql
ss -lnt | grep 3306
```

**方式一：客户端自带 SSH 隧道（Navicat / DBeaver / DataGrip）**

- SSH 页签：主机填服务器 IP，端口 `22`，填 SSH 用户名与密码 / 私钥
- 数据库页签：
  - 主机 `127.0.0.1`
  - 端口 `3306`
  - 用户 `xqyun`
  - 密码为 `.env` 里的 `MYSQL_PASSWORD`
  - 数据库 `xqyun`

**方式二：命令行建隧道**

```bash
ssh -N -L 13306:127.0.0.1:3306 用户名@服务器IP
```

然后客户端连 `127.0.0.1:13306`（不用再配 SSH 页签）。

**说明**

- 不要用 `root` 远程连接：官方镜像的 root 默认只允许本机登录；日常使用 `xqyun` 账号即可。
- 如果宿主机 3306 已被占用，在 `.env` 增加 `MYSQL_HOST_PORT=3307`，隧道与客户端端口同步改成 3307。
- 修改映射后执行 `docker compose up -d mysql` 重建容器生效；数据在 `data/mysql`，不会丢失。
- **不要把端口改成 `3306:3306` 或 `0.0.0.0`**，那会把数据库直接暴露到公网。

## 说明

- 使用官方镜像：`nginx:1.27-alpine`、`node:22-alpine`、`alpine:3.22`、`mysql:8.0`、`redis:7-alpine`、`migrate/migrate:v4.18.3`，首次启动会自动拉取。
- 管理后台、官网、Go 服务都是挂载 `data/release` 下的构建产物，没有构建步骤。
- 数据库首次启动执行 `init.sql` 建库，随后 `migrate` 服务执行 `migration/` 下的迁移。
- 修改 Nginx 配置后：
  ```bash
  docker compose exec gateway nginx -t
  docker compose exec gateway nginx -s reload
  ```
- 更新版本时，替换 `data/release/admin`、`data/release/frontend/.output`、`data/release/backend/xqyun`，然后：
  ```bash
  docker compose restart backend frontend gateway
  ```
- 数据都在 `data/mysql`、`data/redis`、`data/uploads`、`data/logs`，删除容器不会丢数据。
- 开启 HTTPS 后把 `.env` 里的 `JWT_COOKIE_SECURE` 改为 `true`，并把 `UPLOAD_PUBLIC_BASE_URL` 改成 HTTPS 域名。
- 当前 Go 二进制为 **linux/amd64**；ARM 服务器需要重新编译。
