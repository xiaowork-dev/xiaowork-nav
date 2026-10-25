# 安装指南

> 适用版本：XiaoWork Nav 2.0.x
> 关联：[更新指南](updating.md)、[部署总览](deployment.md)、[架构说明](architecture.md)

---

## 1. 前置条件

| 项 | 要求 |
| --- | --- |
| 操作系统 | 任意可运行 Docker 的 Linux 发行版（amd64 / arm64） |
| Docker | Docker Engine 24+ |
| Compose | Docker Compose 插件（`docker compose`，非旧的 `docker-compose`） |
| 磁盘 | 建议 ≥ 1 GB 可用（镜像约 30 MB，数据取决于图标数量） |
| 网络 | 能访问 GHCR（或使用本地构建的镜像） |

无需在宿主机安装 Node、Go 或数据库。

---

## 2. 一键安装（推荐）

```bash
git clone https://github.com/xiaowork-dev/xiaowork-nav.git
cd xiaowork-nav
./scripts/install.sh
```

脚本会依次：检查 Docker 与 Compose → 从 `deploy/.env.example` 生成 `deploy/.env` →
校验 Compose 配置 → 拉取镜像 → 启动 → 等待健康检查 → 输出数据目录与后续步骤。

常用参数：

```bash
./scripts/install.sh --port 9000            # 修改宿主机端口
./scripts/install.sh --bind 0.0.0.0         # 直接对公网开放（不推荐，需自行配置 TLS）
./scripts/install.sh --build                # 本地构建镜像（不使用 GHCR）
./scripts/install.sh --compose compose.npm.yaml   # NPM 共享网络模式
```

安装完成后打开站点，按向导创建管理员账号（初始化只能执行一次）。

---

## 3. 安装模式

### A. 独立模式（默认）

- 应用监听宿主机 `127.0.0.1:8787`（由 `XN_BIND_ADDR` / `XN_PUBLISH_PORT` 决定）。
- 只有本机可访问，适合在同机反向代理后面运行。
- 若使用 SSH 隧道：`ssh -L 8787:127.0.0.1:8787 you@server`，然后访问 `http://127.0.0.1:8787`。

### B. Nginx Proxy Manager 共享网络模式

```bash
cd deploy
cp .env.example .env
# 编辑 .env，至少设置：
#   XN_PROXY_NETWORK=<你的 NPM Docker 网络名>
#   XN_CONTAINER_NAME=xiaowork-nav
cd ..
./scripts/install.sh --compose compose.npm.yaml
```

该模式**不发布任何宿主机端口**。在 NPM 中新建 Proxy Host：

| 字段 | 值 |
| --- | --- |
| Domain | 你的域名 |
| Scheme | `http` |
| Forward Hostname | `xiaowork-nav`（即 `XN_CONTAINER_NAME`） |
| Forward Port | `8080` |
| SSL | 申请并启用 Let's Encrypt |

> 网络名与域名一律通过 `.env` 配置，未写死在 Compose 文件中。

---

## 4. 配置项

所有配置都通过环境变量提供，复制 `deploy/.env.example` 后按需修改。

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `XN_IMAGE` | `ghcr.io/xiaowork-dev/xiaowork-nav` | 镜像仓库 |
| `APP_VERSION` | `latest` | 运行的镜像标签（生产建议固定语义化版本） |
| `XN_CONTAINER_NAME` | `xiaowork-nav` | 容器名 / NPM 转发主机名 |
| `XN_DATA_HOST_DIR` | `./data` | 宿主机数据目录（挂载到容器 `/data`） |
| `XN_BIND_ADDR` | `127.0.0.1` | 独立模式监听地址 |
| `XN_PUBLISH_PORT` | `8787` | 独立模式宿主机端口 |
| `XN_PROXY_NETWORK` | – | NPM 模式使用的外部 Docker 网络名 |
| `XN_LOG_LEVEL` | `info` | 日志级别 |
| `XN_ALLOW_GUEST_ACCESS` | `true` | 是否允许游客浏览公开导航 |
| `XN_SECURE_COOKIE` | `true` | 会话 Cookie 是否标记 `Secure`（HTTPS 下保持 true） |
| `XN_BACKUP_KEEP` | `5` | 应用内保留的备份快照数量 |
| `XN_UID` / `XN_GID` | `100` / `101` | 容器运行使用的 uid/gid（**非 root**）。数据目录属主不同时可改为该属主的 id |
| `XN_UPDATE_REPO` | – | 更新检查使用的 `owner/name` |
| `XN_TRUSTED_PROXIES` | – | 受信任代理 CIDR（用于 `X-Forwarded-*`） |
| `XN_ALLOWED_ORIGINS` | – | 额外允许的 CORS 来源（仅开发用） |
| `TZ` | `UTC` | 日志时区 |

应用还支持（通常无需修改）：`XN_DB_PATH`、`XN_UPLOADS_DIR`、`XN_BACKUPS_DIR`、
`XN_SESSION_COOKIE`、`XN_SESSION_TTL_HOURS`、`XN_HOST`、`XN_PORT`。

### 数据目录

```text
<XN_DATA_HOST_DIR>/
├── app.db          SQLite 数据库（WAL）
├── uploads/        上传/迁移的图标文件
└── backups/        应用内备份快照
```

更新镜像不会影响该目录。**请把它纳入你的常规备份。**

#### 目录权限（首次部署最常见的问题）

容器**以非 root 用户运行**（默认 `uid=100 gid=101`），而数据目录是宿主机绑定挂载，
属主由宿主机决定。如果该目录由其他账号（例如 root）创建，应用在写入时会出现：

```text
mkdir /data/uploads: permission denied
```

`./scripts/install.sh` 会自动处理：以 root 运行时把目录 `chown` 给容器用户；
无法 `chown` 时自动把 `XN_UID`/`XN_GID` 设为**当前账号**的 uid/gid，让容器与宿主机属主一致。

手动修复（任选其一，**不要使用 `chmod 777`**）：

```bash
# 1) 把目录交给容器用户
sudo chown -R 100:101 /path/to/data

# 2) 让容器以目录属主身份运行（无需 root）
id -u   # 记下 uid
id -g   # 记下 gid
# 写入 deploy/.env：
#   XN_UID=<uid>
#   XN_GID=<gid>
docker compose -f deploy/compose.yaml up -d
```

容器启动时会校验数据目录可写，失败时直接给出**可复制执行**的修复命令并退出，
不会留下一个看似启动成功、实则无法保存图标的实例。

---

## 5. 验证安装

```bash
# 容器健康状态
docker inspect -f '{{.State.Health.Status}}' xiaowork-nav

# 应用健康检查
curl -fsS http://127.0.0.1:8787/healthz

# 版本与数据库结构版本
curl -fsS http://127.0.0.1:8787/api/v1/version
```

首次访问会被引导到初始化向导；完成后该入口自动关闭。

---

## 6. 首次使用

1. 打开站点 → 创建管理员账号（用户名 3–32 位，密码至少 8 位）。
2. 「分类管理」新建分类；「网站管理」添加站点（可上传图标、填图片地址、
   使用 Simple Icons 标识或文字图标）。
3. 「系统设置」调整站点标题、Favicon、主题与搜索引擎。
4. 需要迁移旧数据时，进入「数据管理」→ 导入数据：支持本系统导出的 JSON 与
   旧站格式，导入前会显示预览，默认不覆盖已有数据。

---

## 7. 常见问题

| 现象 | 排查 |
| --- | --- |
| `install.sh` 报 docker 未运行 | 启动 Docker 守护进程；`docker info` 应能返回信息 |
| 容器 `unhealthy` | `docker compose -f deploy/compose.yaml logs --tail=80 app`；确认数据目录可写 |
| 启动日志出现 `permission denied` / `is not writable` | 数据目录属主与容器用户不一致，见 §4「目录权限」；**不要用 `chmod 777`** |
| NPM 502 | 确认应用与 NPM 在同一网络、转发端口为 `8080`、容器名与 `XN_CONTAINER_NAME` 一致 |
| 页面能开但登录后立刻掉线 | HTTPS 下需 `XN_SECURE_COOKIE=true`；HTTP 调试时设为 `false` |
| 数据看似丢失 | 确认 `XN_DATA_HOST_DIR` 未改动；可用 `./scripts/restore.sh` 恢复快照 |
| `restore.sh` 提示 "refusing to start" | 快照文件名不含版本；加 `--version vX.Y.Z` 指定后重跑（见[更新指南 §3](updating.md)） |
| NPM 模式下想确认结构版本 | 用 `--db-info` 读数据库文件，无需宿主机端口（见[更新指南 §1.2](updating.md)） |
| 端口被占用 | `./scripts/install.sh --port <空闲端口>` |

---

## 8. 卸载

```bash
cd deploy
docker compose down            # 停止并删除容器（保留数据目录）
# 确认不再需要后，再手动删除 XN_DATA_HOST_DIR 指向的目录
```

> 脚本不会自动删除数据目录，避免误删。
