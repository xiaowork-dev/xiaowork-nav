# P5 阶段交付报告：Docker 部署、GitHub 开源与版本发布

> 本报告是 **P5 阶段的正式、可长期归档交付文档**。
> 内容基于实际代码、真实 Docker 环境测试与 Git 状态编写；未完成或未验证的内容均明确标注。

---

## 1. 阶段基本信息

| 项 | 内容 |
| --- | --- |
| 项目名称 | XiaoWork Nav |
| 项目版本 | **V2.0.0**（后端 `v2.0.0`，前端 `0.1.0`，数据库结构版本 `2`） |
| 阶段编号与名称 | **P5 Docker 部署、GitHub 开源与版本发布** |
| 阶段目标 | 让其他用户可以方便地安装、升级与维护：开源准备、一键部署、版本发布与更新、真实环境部署验证 |
| 阶段开始日期 | 2026-10-25 |
| 阶段完成日期 | 2026-10-25（v2.0.1 补充修复：2026-10-25） |
| 当前完成状态 | **全部完成**：代码与部署工件已在真实 Docker 环境验证；GitHub 仓库已创建并推送；CI 通过；多架构镜像已发布到 GHCR（匿名可拉取）；GitHub Release 已创建。**验收复审后新增 v2.0.1 补充修复**（一键更新与数据库回滚，见 §12） |
| 上位依据 | `AGENTS.md`、`docs/installation.md`、`docs/updating.md` |

---

## 2. 本阶段工作总结

### 2.1 开源准备

- 重写 `README.md`：定位、架构图、特性、一键安装、更新、配置表、开发、文档索引。
- 新增 `SECURITY.md`：支持版本、漏洞上报流程、部署加固清单、内置安全措施。
- 重写 `CHANGELOG.md`：整理为 `[2.0.0] - 2026-10-25` 正式发布条目（Added/Changed/Fixed/Security/Notes）。
- 新增 `docs/installation.md`（安装指南）与 `docs/updating.md`（更新/备份/回滚指南）。
- 重写 `docs/deployment.md` 为部署总览与关键设计约束。
- 仓库安全检查：`.gitignore` 覆盖 `.env`、`data.json`、`*.db`、`docs/screenshots/`、
  `docs/screenshots*.zip`；CI 增加 `repository-safety` 任务做强制校验。
- **实际清理**：删除误生成的 `docs/screenshots.zip`（20.7 MB，含真实站点数据）。

### 2.2 Docker 一键部署

| 交付物 | 说明 |
| --- | --- |
| `Dockerfile` | 三阶段构建：Node 构建前端 → Go 1.26 构建（embed）→ Alpine 非 root 运行；镜像 **36.2 MB** |
| `deploy/compose.yaml` | 独立模式，默认绑定 `127.0.0.1:8787`，`/data` 持久化，健康检查，`no-new-privileges` |
| `deploy/compose.npm.yaml` | NPM 共享网络模式，**不发布宿主机端口** |
| `deploy/.env.example` | 全部环境变量与说明（含 `TZ`） |
| `scripts/install.sh` | 一键安装：检查依赖 → 生成 `.env` → 校验 → 拉取 → 启动 → 健康检查 → 输出后续步骤 |
| `scripts/lib.sh` | 脚本公共函数（日志、compose 封装、健康等待、快照、环境变量读写） |
| `scripts/deploy.sh` | 重新部署 / 重启 / 查看日志，支持 `--build` |
| `scripts/backup.sh` / `scripts/restore.sh` | 数据目录快照与恢复（含安全快照） |

- 持久化：`app.db`、`uploads/`、`backups/` 全部位于挂载目录，镜像更新不影响。
- 配置：所有部署参数（镜像、版本、容器名、数据目录、绑定地址、端口、代理网络、时区等）
  均通过环境变量注入，未写死域名或网络名。

### 2.3 版本发布与更新

| 交付物 | 说明 |
| --- | --- |
| `.github/workflows/ci.yml` | 后端构建/vet/测试、前端构建、镜像构建 + 容器冒烟、仓库安全检查 + shellcheck |
| `.github/workflows/release.yml` | 标签触发：验证 → buildx 多架构（amd64/arm64）→ 推送 GHCR（版本/次版本/latest）→ 创建 GitHub Release |
| `scripts/update.sh` | 记录版本与结构版本 → 快照 → 原子切换版本 → 启动 → 健康检查与结构版本核对 → 失败自动回滚 |

### 2.4 数据库降级保护（本阶段新增的关键安全机制）

`backend/internal/migrations` 新增 `CheckCompatible`：当数据库结构版本**高于**当前镜像内置的
迁移版本时，应用**拒绝启动**并输出明确指引。这直接满足"不得直接使用旧镜像打开不兼容的新数据库"
的要求，并让"回滚"成为可安全判断的操作（见 `docs/updating.md` §4）。

---

## 3. 代码和文件变更

### 3.1 新增

| 文件 | 作用 |
| --- | --- |
| `.github/workflows/ci.yml` | 持续集成 |
| `.github/workflows/release.yml` | 发布流水线 |
| `scripts/lib.sh` | 部署脚本公共库 |
| `scripts/install.sh` | 一键安装 |
| `scripts/backup.sh` | 数据备份 |
| `scripts/restore.sh` | 数据恢复 |
| `docs/installation.md` | 安装指南 |
| `docs/updating.md` | 更新/回滚指南 |
| `SECURITY.md` | 安全策略 |
| `docs/deliveries/P5-delivery.md` | 本报告 |

### 3.2 修改

| 文件 | 变更 |
| --- | --- |
| `README.md` | 重写为公开发布版本 |
| `CHANGELOG.md` | 整理为 `[2.0.0]` 发布条目 |
| `Dockerfile` | Go 镜像 `1.24` → `1.26`（与 `go.mod` 对齐）；新增可覆盖的 `GOPROXY` 构建参数 |
| `deploy/compose.yaml`、`deploy/compose.npm.yaml` | 增加 `TZ` 透传与 `no-new-privileges` |
| `deploy/.env.example` | 增加 `TZ` |
| `.gitignore` | 增加 `docs/screenshots*.zip` |
| `backend/internal/migrations/migrations.go` | 新增 `ErrSchemaTooNew` 与 `CheckCompatible` |
| `backend/cmd/server/main.go` | 启动前校验结构兼容性，不兼容则拒绝启动 |
| `backend/internal/migrations/migrations_test.go` | 新增降级保护测试 |
| `scripts/deploy.sh`、`scripts/update.sh` | 重构为基于 `lib.sh`；更新脚本加入结构版本核对与快照回滚 |
| `docs/deployment.md`、`docs/README.md` | 部署总览与索引更新 |

---

## 4. 测试与验证结果

### 4.1 自动化测试

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 后端构建 | `go build ./...` | ✅ 通过 |
| 后端静态检查 | `go vet ./...` | ✅ 通过 |
| 后端测试 | `go test -count=1 ./...` | ✅ **71 通过 / 0 失败** |
| 前端类型检查 + 构建 | `npm run build` | ✅ 通过 |
| 脚本语法 | `bash -n scripts/*.sh` | ✅ 全部通过 |
| Compose 配置 | `docker compose config` | ✅ 两套配置均通过 |

### 4.2 真实 Docker 环境部署测试（本阶段重点）

环境：Windows 11 + Docker Desktop，Docker **29.8.2**（linux/amd64），Compose **v5.6.0**。
全部使用**本地构建的真实镜像** `xiaowork-nav:2.0.0`，数据目录位于宿主机临时路径。

| # | 测试项 | 方法 | 结果 |
| --- | --- | --- | --- |
| 1 | 镜像构建 | `docker build` | ✅ 成功，**36.2 MB** |
| 2 | 全新安装 | `./scripts/install.sh --no-pull` | ✅ 容器 `healthy` |
| 3 | 非 root 运行 | `docker exec … id` | ✅ `uid=100(app)` |
| 4 | 健康检查 | `GET /healthz` | ✅ `{"status":"ok","version":"v2.0.0"}` |
| 5 | 版本接口 | `GET /api/v1/version` | ✅ `schema_version=2`，commit 正确 |
| 6 | 前端托管 | `GET /` | ✅ 返回 SPA（含 `id="app"`） |
| 7 | 宿主机持久化 | 检查数据目录 | ✅ `app.db`、`uploads/`、`backups/` 均在宿主机 |
| 8 | 初始化管理员 | `POST /api/v1/setup` | ✅ 201 |
| 9 | 旧数据导入 | 导入真实 `data.json` | ✅ 10 分类 / 110 站点 / 0 跳过 |
| 10 | 图标迁移 | 检查上传目录 | ✅ 31 个 SVG 写入宿主机 |
| 11 | 数据持久化 | `docker compose down` + `up -d` | ✅ 数据完整（10 / 110 / 31） |
| 12 | 登录可用 | `POST /api/v1/auth/login` | ✅ 成功 |
| 13 | 备份 | `./scripts/backup.sh` | ✅ 生成 31 KB 快照，含 `app.db` 与 `uploads/` |
| 14 | **升级 + 迁移** | `./scripts/update.sh 2.0.1 --local`（镜像含新增迁移） | ✅ 版本 2.0.0→2.0.1，**结构版本 2→3**，健康检查通过，数据完整 |
| 15 | **降级保护** | 切回 2.0.0 镜像启动 | ✅ **拒绝启动**：`database schema is newer than this build; refusing to start (database_schema=3 build_schema=2)`，容器重启失败而非破坏数据 |
| 16 | **正确回滚** | `./scripts/restore.sh <pre-update 快照>` | ✅ 恢复至结构版本 2，启动 2.0.0，`healthy`，数据完整（10 / 110 / 31） |

> 说明：测试 14–16 使用的"2.0.1"镜像是在**仓库临时副本**中额外添加一个迁移文件构建的，
> 仅用于验证升级与回滚路径，**未进入仓库**，测试后已删除该镜像与临时副本。

### 4.3 未执行的测试及原因

| 未执行项 | 原因 |
| --- | --- |
| **NPM 反向代理端到端** | 环境无 Nginx Proxy Manager 实例；仅验证了 `compose.npm.yaml` 配置有效性 |
| **真实 HTTPS / 域名** | 无公网域名与证书 |
| **长期运行与并发压力** | 未做 |
| **Windows 原生（非 Docker）部署** | 目标平台为 Linux 容器 |

> GitHub Actions（CI 与 Release）已在本次发布中**实际运行并通过**，见第 9 节。
> **声明：本报告未将任何未执行的测试标记为已通过。**

---

## 5. 项目总体进度

| 阶段 | 名称 | 状态 |
| --- | --- | --- |
| P1 | 需求梳理、契约冻结与项目初始化 | ✅ 已完成 |
| P2 | Go 后端核心业务开发 | ✅ 已完成 |
| P3 | Vue 3 前端功能开发与联调（含旧站迁移） | ✅ 已完成 |
| P4 | UI、交互与主题优化（含收尾修复） | ✅ 已完成 |
| **P5** | **Docker 部署、GitHub 开源与版本发布** | ✅ **已完成**（仓库、CI、多架构镜像、Release 均已发布并核验） |

---

## 6. Git 版本记录

> 说明：为遵守"不把提交 SHA 硬写进待提交文件"的原则，下列 SHA 为**本报告提交前**的状态。

| 项 | 内容 |
| --- | --- |
| 当前分支 | `main`（原 `master`，推送前重命名） |
| 本报告提交前的最近提交 | `d18539d` — `ci(release): keep the v prefix on image tags` |
| 提交总数 | 11 |
| 提交身份 | 本仓库本地占位：`xiaowork-dev <xiaowork-dev@users.noreply.github.com>` |
| GitHub 远程 | ✅ `origin` → `https://github.com/xiaowork-dev/xiaowork-nav.git` |
| 是否已完成推送 | ✅ `main` 与标签 `v2.0.0` 均已推送；`git ls-remote` 核验远程 HEAD = 本地 HEAD |
| 是否创建 Release | ✅ `v2.0.0`（Latest） |
| 是否发布镜像 | ✅ `ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.0`（amd64 + arm64，匿名可拉取） |

---

## 7. 交付物地址

| 项 | 值 | 状态 |
| --- | --- | --- |
| GitHub 仓库 | `https://github.com/xiaowork-dev/xiaowork-nav` | ✅ **已创建并推送**（公开） |
| GitHub Release | `https://github.com/xiaowork-dev/xiaowork-nav/releases/tag/v2.0.0` | ✅ 已发布（Latest） |
| Docker 镜像 | `ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.0` | ✅ **已发布且匿名可拉取** |
| 镜像标签 | `v2.0.0`、`v2.0`、`latest` | ✅ 多架构 `linux/amd64` + `linux/arm64` |
| 镜像摘要 | `sha256:c8e2151f22e368d28e0016742f2aabeb8d15418fce2ec4b115485813a2ae24ee` | ✅ 已实测拉取并运行 |

### 一键安装命令

```bash
git clone https://github.com/xiaowork-dev/xiaowork-nav.git
cd xiaowork-nav
./scripts/install.sh
```

### 一键更新命令

```bash
./scripts/update.sh v2.0.1            # 或 latest
```

### 版本回滚方法

| 场景 | 方法 |
| --- | --- |
| 未发生数据库迁移 | `sed -i 's/^APP_VERSION=.*/APP_VERSION=v2.0.0/' deploy/.env && docker compose -f deploy/compose.yaml up -d` |
| **已发生迁移** | `./scripts/restore.sh <pre-update 快照>` 然后切回旧版本（`update.sh` 失败时会自动执行） |

判断依据：`curl /api/v1/version` 的 `schema_version` 是否变大。

---

## 8. 遗留问题与风险

### 8.1 未完成的任务

| 任务 | 说明 |
| --- | --- |
| 独立测试服务器上的真实部署 + NPM 反代 HTTPS | 环境无 NPM 实例与公网域名 |
| 清理 GHCR 遗留标签 `2.0.0` / `2.0` | 见 §9.3，可选且非阻塞 |
| 跨浏览器 / 真机触控 / 无障碍审计 | 见 P4 报告 |
| 替换占位提交身份 | 用户提供真实 Git 身份后更新 |

### 8.2 已知问题

| 问题 | 影响 | 处理建议 |
| --- | --- | --- |
| GHCR 遗留标签 `2.0.0` / `2.0` | 与规范标签并存，可能造成混淆 | 见 §9.3，可在包管理页一键删除 |
| `golang` 模块代理在受限网络下不可达 | 本地/离线构建镜像失败 | 构建时传 `--build-arg GOPROXY=<镜像源>,direct`（已在 Dockerfile 注释说明） |
| 提交身份为占位 | 提交作者信息不真实 | 用户提供后替换并 rebase，或后续提交使用真实身份 |
| 部分站点图标依赖第三方 | 23 个站点显示首字母、4 个第三方 Favicon 内容透明 | 见迁移报告 §13.5 |
| 登录限流可能短时锁定管理员 | 连续失败后约 5 分钟不可登录 | 已知取舍，可按需调整为仅 IP 限流 |

### 8.3 尚未验证的功能

- Nginx Proxy Manager 真实反代与 HTTPS。
- 生产规模下的长期稳定性与并发。
- 真机触控与跨浏览器渲染（见 P4 报告）。

---

## 9. GitHub 发布执行记录（已完成）

仓库由用户在浏览器中登录后创建，随后由我完成推送与发布。

### 9.1 实际执行结果

| 步骤 | 结果 |
| --- | --- |
| 创建仓库 | ✅ `xiaowork-dev/xiaowork-nav`（Public，未初始化任何文件） |
| 关联远程 | ✅ `origin` → `https://github.com/xiaowork-dev/xiaowork-nav.git` |
| 重命名分支 | ✅ `master` → `main` |
| 推送主分支 | ✅ `main` → `origin/main`（远程与本地 HEAD 一致） |
| 打标签并推送 | ✅ `v2.0.0`（附注标签） |
| CI（main） | ✅ 成功（2m 2s） |
| CI（tag） | ✅ 成功（1m 21s） |
| Release 流水线 | ✅ 成功，构建 `linux/amd64` + `linux/arm64` 并推送 GHCR |
| GitHub Release | ✅ `v2.0.0` 已创建（标记 Latest） |
| 匿名拉取镜像 | ✅ `docker pull ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.0`（36.2 MB） |
| 运行已发布镜像 | ✅ `/healthz` = ok，`/api/v1/version` = `v2.0.0` / `schema_version=2` |

Release 流水线各任务耗时：

| 任务 | 耗时 |
| --- | --- |
| Verify before publishing | 24s |
| Build amd64 | 1m 44s |
| **Build arm64（原生 runner）** | **1m 11s** |
| Publish multi-arch manifest | 22s |
| Create GitHub Release | 5s |

### 9.2 发布过程中发现并修复的两个问题

| 问题 | 现象 | 修复 |
| --- | --- | --- |
| **QEMU 构建 arm64 极慢** | 首次 Release 运行 48 分钟仍未完成（纯 Go SQLite 驱动代码量巨大） | 改为矩阵构建：amd64 用 `ubuntu-latest`、arm64 用 `ubuntu-24.04-arm` **原生 runner**，按 digest 推送后合并为多架构清单。arm64 从 >48 分钟降到 **1m 11s** |
| **镜像标签缺少 `v` 前缀** | `docker/metadata-action` 的 semver 模式会去掉 `v`，实际标签为 `2.0.0`/`2.0`，而 git 标签、README、Release 说明、`deploy/.env.example` 都写 `v2.0.0`，导致 `docker pull ...:v2.0.0` 与 `APP_VERSION=v2.0.0` 失败 | 标签模式改为 `v{{version}}` / `v{{major}}.{{minor}}`，使镜像标签与所有文档一致 |

### 9.3 遗留的清理项（可选，非阻塞）

首次发布尝试产生了 `2.0.0` 与 `2.0` 两个标签（指向更早一次构建的同一版本代码，仅内嵌 commit 不同）。
规范标签为 `v2.0.0` / `v2.0` / `latest`。后续发布不会再产生无 `v` 前缀的标签。

如需清理：打开 `https://github.com/users/xiaowork-dev/packages/container/xiaowork-nav/versions`，
删除摘要以 `cf6040d5` 开头的那一条即可。

### 9.4 本次新增提交

| 提交 | 说明 |
| --- | --- |
| `a4e773c` | 每个架构使用原生 runner 构建 |
| `d18539d` | 镜像标签保留 `v` 前缀 |

> 推送与发布均已通过 `git ls-remote`、GitHub Actions 运行记录与匿名 `docker pull` 三重核验，**未虚构任何结果**。

---

## 10. 发布前安全检查结论

| 检查项 | 结果 |
| --- | --- |
| 真实导航数据是否入库 | ✅ 否（`data.json` 已忽略，仓库内无真实数据文件） |
| 数据库/备份文件是否入库 | ✅ 否（`*.db` 已忽略） |
| `.env` 是否入库 | ✅ 否（仅 `.env.example` 提交） |
| 截图（含真实数据）是否入库 | ✅ 否（`docs/screenshots/` 与 `*.zip` 已忽略；并已删除误生成的 20.7 MB 压缩包） |
| 私钥/令牌 | ✅ 未发现（CI 亦有强制检查） |
| 默认管理员密码 | ✅ 无默认密码，首次运行向导创建 |
| 容器权限 | ✅ 非 root + `no-new-privileges` |
| 降级保护 | ✅ 旧镜像拒绝打开新结构数据库 |

---

## 11. 下一步

1. 在一台**独立测试服务器**上按 `docs/installation.md` 完成一次真实部署与 NPM 反代 HTTPS 验证。
2. 可选：清理 GHCR 遗留标签（§9.3）。
3. 可选：提供真实 Git 提交身份，替换本地占位身份。
4. 进入后续版本（P1 增强项：多主题、书签导入、PWA 等，见 `docs/requirements.md` §2）。

---

## 12. P5 补充交付（v2.0.1）：一键更新与数据库回滚缺陷修复

P5 验收复审确认 Docker 部署、CI/CD、GHCR 与 Release 通过，但**一键更新与数据库回滚**
存在缺陷。修复已在 2026-10-25 完成并通过真实 Docker 异常注入验证。

- **完整增量记录**：[`20261025-170154-p5-update-rollback-fix.md`](20261025-170154-p5-update-rollback-fix.md)
- **版本**：V2.0.1（数据库结构版本仍为 `2`，**无结构变更**，可从 2.0.0 直接升级）
- **历史标签 `v2.0.0` 未移动、未重建**

### 12.1 修复摘要

| # | 缺陷 | 修复 |
| --- | --- | --- |
| 1 | `latest` 与当前版本相同时直接退出，永不重新拉取 | 可变标签（`latest`、`v2.0`）总是查询并拉取；镜像未变且容器健康时报告"无需操作"且不重建容器；固定 `vX.Y.Z` 仍幂等 |
| 2 | `app_schema_version()` 依赖宿主机 `127.0.0.1` 端口，NPM 模式不发布端口 | 新增 `--db-info` 命令行模式，**直接读数据库文件**获得结构版本与行数；不依赖端口、容器停止时也可用、只读 |
| 3 | 回滚判断依赖 HTTP，无法判断时行为不安全 | 结构版本变大 / 无法读取 / 更新前未知 → **先用更新前快照恢复数据**再切回旧镜像；覆盖拉取失败、启动失败、健康检查失败、迁移后失败、版本不可读、NPM 模式失败六类场景 |
| 4 | 回滚只看 `docker compose up -d` 退出码 | 回滚后**再次验证**健康检查、结构版本与书签行数 |
| 5 | `restore.sh` 恢复旧库后又用新镜像启动并重新迁移 | 启动**之前**把 `APP_VERSION` 切到快照版本（`--version` 指定或从文件名推断）；版本未知时只恢复数据、保持停止并打印指引 |

### 12.2 本阶段新增/修改（摘要）

- 后端：`cmd/server/main.go`（`--db-info`）、`internal/migrations`（`ReadVersion`）、
  `internal/store`（`OpenForInspection`）；新增 2 个测试文件。
- 脚本：`scripts/lib.sh`、`scripts/update.sh`、`scripts/restore.sh` 重写关键逻辑。
- 测试：新增 `scripts/tests/deploy-faults.test.sh`（12 场景 / 43 断言）。
- 文档：`docs/updating.md`、`docs/deployment.md`、`docs/installation.md`、`README.md`、`CHANGELOG.md`。
- 决策：新增 [`docs/decisions/ADR-0005-db-info-inspection.md`](../decisions/ADR-0005-db-info-inspection.md)
  （用 `--db-info` 直接读库判断结构版本，替代依赖宿主机端口的 HTTP 查询）。
- CI：shellcheck 覆盖 `scripts/tests/*.sh`。

### 12.3 验证结果（2026-10-25）

| 检查 | 结果 |
| --- | --- |
| 后端 `go build` / `go vet` | ✅ 通过 |
| 后端 `go test -count=1 ./...` | ✅ **75 通过 / 0 失败** |
| 前端 `npm run build` | ✅ 通过 |
| `bash -n` 全部脚本 | ✅ 通过 |
| Compose 配置校验（独立 + NPM） | ✅ 通过 |
| OpenAPI YAML 语法 | ✅ 通过 |
| **异常注入测试（真实 Docker，12 场景）** | ✅ **43 断言全部通过，0 失败** |

未执行项：真实 NPM 反代 + HTTPS 端到端、真实 GHCR 新版本拉取升级（v2.0.1 发布前镜像
尚不存在）、长期稳定性与并发、Linux 原生运行测试脚本（本机为 Windows + Docker Desktop）。

---

## 附：本报告的编写依据

- 代码与文件：仓库实际文件（2026-10-25）。
- Docker 测试：`docker build`、`docker compose up/down`、`scripts/*.sh`、HTTP 接口的实际输出（见 4.2）。
- 测试记录：`go test -count=1 ./...`、`npm run build`、`bash -n`、`docker compose config` 的实际输出。
- GitHub 发布：`git ls-remote`、GitHub Actions 运行记录（`/actions/runs/*`）、匿名 `docker pull` 与已发布镜像的运行时输出（见第 9 节）。
- Git 状态：`git log` / `git status` / `git remote -v` 实际输出（见第 6 节）。
