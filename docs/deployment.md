# 部署总览

> 本文是部署主题的入口。具体步骤见：
> **[安装指南](installation.md)**、**[更新与回滚指南](updating.md)**。
> 关联：[架构说明](architecture.md)、[数据库设计](database-design.md)。

---

## 1. 部署形态

单容器、单进程：Go 服务同时提供 REST API 与前端静态资源，SQLite 与上传图标
持久化在挂载目录 `/data`。

```text
Browser ──HTTPS──▶ 反向代理 (NPM / Nginx / Caddy)
                        │  (私有 Docker 网络)
                        ▼
              xiaowork-nav 容器
              ├── Go API  (/api/v1, /healthz)
              ├── Vue SPA (embed)
              └── /data  ← 挂载自宿主机
                   ├── app.db
                   ├── uploads/
                   └── backups/
```

运行时**不需要** Node，也不需要外部数据库。

---

## 2. 快速入口

| 目标 | 命令 / 文档 |
| --- | --- |
| 一键安装 | `./scripts/install.sh` → [安装指南](installation.md) |
| 反向代理（NPM） | `./scripts/install.sh --compose compose.npm.yaml` |
| 重新部署 / 重启 | `./scripts/deploy.sh` |
| 本地构建镜像 | `./scripts/deploy.sh --build` |
| 更新版本 | `./scripts/update.sh v2.0.1` → [更新指南](updating.md) |
| 备份 | `./scripts/backup.sh` |
| 恢复 | `./scripts/restore.sh <快照>` |
| 查看日志 | `./scripts/deploy.sh --logs` |

---

## 3. 文件与目录

| 路径 | 作用 |
| --- | --- |
| `Dockerfile` | 多阶段构建：前端 → Go(embed) → 精简运行镜像（非 root） |
| `deploy/compose.yaml` | 独立模式（宿主机端口，默认 `127.0.0.1:8787`） |
| `deploy/compose.npm.yaml` | NPM 共享网络模式（不发布宿主机端口） |
| `deploy/.env.example` | 环境变量模板 |
| `scripts/lib.sh` | 脚本公共函数（含基于 `--db-info` 的结构版本读取） |
| `scripts/install.sh` | 首次安装 |
| `scripts/deploy.sh` | 重新部署 / 重启 / 查看日志 |
| `scripts/update.sh` | 安全更新（检查/拉取 → 快照 → 切换 → 健康检查 → 验证回滚） |
| `scripts/backup.sh` | 数据目录快照 |
| `scripts/restore.sh` | 从快照恢复（可指定/推断版本，启动前切换） |
| `scripts/tests/deploy-faults.test.sh` | 部署脚本异常注入测试（需 Docker） |
| `.github/workflows/ci.yml` | 构建、测试、镜像构建、仓库安全检查 |
| `.github/workflows/release.yml` | 多架构镜像发布到 GHCR + GitHub Release |

---

## 4. 关键设计约束

1. **持久化**：数据库、图标、备份都在 `/data`，镜像更新不影响。
2. **默认不暴露公网**：`XN_BIND_ADDR` 默认 `127.0.0.1`；NPM 模式完全不发布端口。
3. **配置全部来自环境变量**：域名、网络名、端口均不写死。
4. **更新前必快照**：`update.sh` 在改动前生成一致性备份。
5. **结构版本保护**：数据库结构比镜像新时，应用**拒绝启动**，避免旧代码破坏数据。
6. **失败可回滚**：可逆迁移切回旧镜像；不可逆迁移用快照恢复；无法判断兼容性时
   按最保守方式先恢复快照。回滚后会**再次验证**健康与数据，而非只看退出码。
7. **检测不依赖端口**：结构版本从数据库文件读取（`--db-info`），因此在 NPM
   共享网络模式（不发布宿主机端口）与应用停止时同样可用。
8. **非 root 运行**：容器以 `app` 用户运行，并启用 `no-new-privileges`。

---

## 5. 健康检查与冒烟验证

```bash
docker inspect -f '{{.State.Health.Status}}' xiaowork-nav   # healthy
curl -fsS http://127.0.0.1:8787/healthz                     # {"status":"ok",...}
curl -fsS http://127.0.0.1:8787/api/v1/version              # 版本 + schema_version
curl -fsS http://127.0.0.1:8787/ | grep -q 'id="app"'        # SPA 已托管
```

CI 中的 `docker` 任务会执行同样的冒烟检查。

---

## 6. 环境变量

完整列表见 [安装指南 §4](installation.md)。最小可用配置只有：

```dotenv
XN_IMAGE=ghcr.io/xiaowork-dev/xiaowork-nav
APP_VERSION=v2.0.0
XN_DATA_HOST_DIR=./data
```

---

## 7. 相关文档

- [安装指南](installation.md) — 前置条件、两种安装模式、首次使用、排障
- [更新与回滚指南](updating.md) — 更新、备份、恢复、回滚、迁移兼容性
- [数据库设计](database-design.md) — 表结构与迁移策略
- [架构说明](architecture.md) — 模块边界与运行时架构
