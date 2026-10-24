# XiaoWork Nav 部署指南

> 文档类型：部署说明
> 更新日期：2026-10-24
> 状态：P1（部署工件已就绪，镜像构建/生产验证待 P5）
> 关联：`../deploy/compose.yaml`、`../scripts/deploy.sh`、`../scripts/update.sh`、`decisions/ADR-0002-single-container-embed.md`

---

## 1. 架构与前置条件

单个应用容器：Go 服务同时托管前端静态资源与 REST API，SQLite 与上传目录持久化到挂载卷 `/data`。

前置条件：

- Linux 服务器，已安装 **Docker Engine** 与 **Docker Compose 插件**。
- 无需在宿主机安装 Node、Go 或数据库。
- 一个反向代理（如 Nginx Proxy Manager）用于终止 TLS（推荐）。

---

## 2. 目录与文件

```text
Dockerfile                    多阶段构建（前端 → Go(embed) → 精简运行镜像）
deploy/compose.yaml           独立模式（宿主机端口，默认 127.0.0.1:8787）
deploy/compose.npm.yaml       NPM 共享网络模式（不发布宿主机端口）
deploy/.env.example           环境变量模板（复制为 .env）
scripts/deploy.sh             部署 / 重启
scripts/update.sh             安全更新（备份 + 版本切换 + 健康检查 + 回滚）
```

---

## 3. 快速开始（独立模式）

```bash
git clone <repo-url> xiaowork-nav
cd xiaowork-nav/deploy
cp .env.example .env
# 编辑 .env：至少确认 XN_DATA_HOST_DIR 与 APP_VERSION
../scripts/deploy.sh
```

脚本会：校验 Compose 配置 → 拉取镜像（或 `--build` 本地构建）→ 启动 → 等待健康检查 → 输出状态。

默认监听 `127.0.0.1:8787`（仅本机可访问，适合前置反代）。

---

## 4. NPM 共享网络模式（推荐用于公网）

1. 在 `.env` 中设置：

   ```dotenv
   XN_PROXY_NETWORK=<你的 NPM Docker 网络名>
   XN_CONTAINER_NAME=xiaowork-nav
   ```

2. 部署：

   ```bash
   ../scripts/deploy.sh --compose compose.npm.yaml
   ```

3. 在 NPM 中新建 Proxy Host：
   - Forward Hostname：`xiaowork-nav`（即 `XN_CONTAINER_NAME`）
   - Forward Port：`8080`
   - 申请并启用 HTTPS。

> 该模式**不发布宿主机端口**，因此公网无法通过 IP+端口直接访问应用。
> 网络名与域名一律通过 `.env` 配置，不写死在 Compose 文件中。

---

## 5. 环境变量

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `XN_IMAGE` | `ghcr.io/xiaowork-dev/xiaowork-nav` | 镜像仓库 |
| `APP_VERSION` | `latest` | 运行的镜像标签（建议固定语义化版本） |
| `XN_CONTAINER_NAME` | `xiaowork-nav` | 容器名 / NPM 转发主机名 |
| `XN_DATA_HOST_DIR` | `./data` | 宿主机数据目录（挂载到 `/data`） |
| `XN_BIND_ADDR` | `127.0.0.1` | 独立模式监听地址 |
| `XN_PUBLISH_PORT` | `8787` | 独立模式宿主机端口 |
| `XN_PROXY_NETWORK` | 空 | NPM 模式的外部网络名 |
| `XN_LOG_LEVEL` | `info` | 日志级别 |
| `XN_ALLOW_GUEST_ACCESS` | `true` | 是否允许游客浏览公开导航 |
| `XN_SECURE_COOKIE` | `true` | 会话 Cookie 是否标记 Secure（生产保持 true；本地 HTTP 调试设为 false） |
| `XN_BACKUP_KEEP` | `5` | 保留的备份快照数量（0 = 不限制） |
| `XN_UPDATE_REPO` | 空 | 用于更新检查的 GitHub 仓库 `owner/name`（空则关闭远程检查） |
| `XN_TRUSTED_PROXIES` | 空 | 受信任代理 CIDR（逗号分隔） |
| `XN_ALLOWED_ORIGINS` | 空 | 允许的 CORS 来源（仅开发用） |
| `XN_SESSION_TTL_HOURS` | `168` | 会话有效期（小时） |
| `XN_SESSION_COOKIE` | `xiaowork_session` | 会话 Cookie 名 |

数据目录内容：

```text
/data/app.db         SQLite 数据库
/data/uploads/       上传的图标
/data/backups/       备份快照（P2 起由应用生成）
```

---

## 6. 更新

```bash
cd deploy
../scripts/update.sh v2.0.1      # 或 latest
```

流程：读取当前版本 → 停止应用并**快照数据目录** → 原子切换 `APP_VERSION` → 拉取镜像 → 启动 → 健康检查；失败自动回滚到旧版本并保留快照。

- 数据库迁移在容器启动时自动执行；**迁移失败会拒绝启动**。
- 可逆迁移可直接回退镜像；不可逆迁移需使用快照恢复。
- 更新不会覆盖 `/data` 中的用户数据。

---

## 7. 健康检查与排障

```bash
curl -fsS http://127.0.0.1:8787/healthz
docker compose -f deploy/compose.yaml --env-file deploy/.env logs -f app
docker inspect -f '{{.State.Health.Status}}' xiaowork-nav
```

常见问题：

| 现象 | 排查 |
| --- | --- |
| 容器 unhealthy | 查看日志；确认 `/data` 可写；确认迁移未失败 |
| NPM 502 | 确认应用与 NPM 在同一网络；转发端口为 `8080`；容器名一致 |
| 数据丢失 | 确认 `XN_DATA_HOST_DIR` 未被更改；从备份快照恢复 |

---

## 8. 本地开发（不使用 Docker）

```bash
# 后端
cd backend && go run ./cmd/server        # http://localhost:8080

# 前端（另一终端）
cd frontend && npm install && npm run dev # http://localhost:5173
```

---

## 9. 当前验证状态

| 项 | 状态 |
| --- | --- |
| `compose.yaml` / `compose.npm.yaml` 配置校验 | ✅ `docker compose config` 通过 |
| `deploy.sh` / `update.sh` 语法检查 | ✅ `bash -n` 通过 |
| Docker 镜像实际构建 | ⏳ 未执行（当前环境 Docker 守护进程不可用） |
| 全新服务器部署 / 升级实测 | ⏳ 待 P5 |

> 未执行的验证不得视为已通过；P5 将在具备 Docker 的环境完成构建、部署与升级实测。
