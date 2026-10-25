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
| 阶段完成日期 | 2026-10-25 |
| 当前完成状态 | **代码与部署工件已完成并在真实 Docker 环境验证**；**GitHub 仓库创建与推送、GHCR 镜像发布未完成**（缺少远程与凭据，需用户执行，详见第 9 节） |
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
| **GitHub Actions 实际运行** | 本地无法运行 Actions；未创建远程仓库。工作流仅通过 YAML 语法与步骤审查 |
| **GHCR 镜像发布与拉取** | 未配置 GitHub 远程与凭据，未推送 |
| **多架构（arm64）镜像** | 本机为 amd64，buildx 多架构推送依赖 CI |
| **NPM 反向代理端到端** | 环境无 Nginx Proxy Manager 实例；仅验证了 `compose.npm.yaml` 配置有效性 |
| **真实 HTTPS / 域名** | 无公网域名与证书 |
| **长期运行与并发压力** | 未做 |
| **Windows 原生（非 Docker）部署** | 目标平台为 Linux 容器 |

> **声明：本报告未将任何未执行的测试标记为已通过。**

---

## 5. 项目总体进度

| 阶段 | 名称 | 状态 |
| --- | --- | --- |
| P1 | 需求梳理、契约冻结与项目初始化 | ✅ 已完成 |
| P2 | Go 后端核心业务开发 | ✅ 已完成 |
| P3 | Vue 3 前端功能开发与联调（含旧站迁移） | ✅ 已完成 |
| P4 | UI、交互与主题优化（含收尾修复） | ✅ 已完成 |
| **P5** | **Docker 部署、GitHub 开源与版本发布** | 🟡 **代码与部署工件完成并实测；仓库创建/推送/镜像发布待用户执行** |

---

## 6. Git 版本记录

> 说明：为遵守"不把提交 SHA 硬写进待提交文件"的原则，下列 SHA 为**本报告提交前**的状态。

| 项 | 内容 |
| --- | --- |
| 当前分支 | `master` |
| 本报告提交前的最近提交 | `ee7e7c8` — `fix(frontend): P4 wrap-up - mobile admin layout, paging and sort mode` |
| 提交总数 | 8 |
| 提交身份 | 本仓库本地占位：`xiaowork-dev <xiaowork-dev@users.noreply.github.com>` |
| 是否配置 GitHub 远程 | **否**（`git remote -v` 无输出） |
| 是否已完成推送 | **否** |
| 是否创建 Release / 打标签 | **否** |

---

## 7. 交付物地址

| 项 | 值 | 状态 |
| --- | --- | --- |
| GitHub 仓库 | `https://github.com/xiaowork-dev/xiaowork-nav`（**尚未创建**） | ⏳ 待用户创建 |
| Docker 镜像 | `ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.0`（**尚未发布**） | ⏳ 待 CI 发布 |
| 本地镜像 | `xiaowork-nav:2.0.0`（36.2 MB，已实测） | ✅ 可用 |

### 一键安装命令

```bash
git clone https://github.com/xiaowork-dev/xiaowork-nav.git
cd xiaowork-nav
./scripts/install.sh
```

在远程仓库与镜像发布前，可先用本地镜像验证：

```bash
git clone <仓库或本地路径> && cd xiaowork-nav
./scripts/install.sh --build          # 本地构建并安装
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

### 8.1 未完成的任务（需用户执行）

| 任务 | 说明 |
| --- | --- |
| 创建 GitHub 仓库 | `xiaowork-dev/xiaowork-nav`（公开或私有） |
| 配置远程并推送 | `git remote add origin …` + `git push -u origin master` |
| 打标签并发布 | `git tag v2.0.0 && git push origin v2.0.0`，由 Actions 发布镜像与 Release |
| 确认 GHCR 包可见性 | 首次推送后需在 GitHub 设置中确认包为 public（若需匿名拉取） |
| 在服务器上正式部署 | 未操作任何现有服务器 |

### 8.2 已知问题

| 问题 | 影响 | 处理建议 |
| --- | --- | --- |
| 未创建远程仓库 | 无法推送、无法运行 CI、无镜像 | 由用户创建后执行第 9 节步骤 |
| `golang` 模块代理在受限网络下不可达 | 本地/离线构建镜像失败 | 构建时传 `--build-arg GOPROXY=<镜像源>,direct`（已在 Dockerfile 注释说明） |
| 提交身份为占位 | 提交作者信息不真实 | 用户提供后替换并 rebase，或后续提交使用真实身份 |
| 部分站点图标依赖第三方 | 23 个站点显示首字母、4 个第三方 Favicon 内容透明 | 见迁移报告 §13.5 |
| 登录限流可能短时锁定管理员 | 连续失败后约 5 分钟不可登录 | 已知取舍，可按需调整为仅 IP 限流 |

### 8.3 尚未验证的功能

- GitHub Actions 实际执行（CI 与 Release）。
- GHCR 镜像推送、拉取与多架构。
- Nginx Proxy Manager 真实反代与 HTTPS。
- 生产规模下的长期稳定性与并发。

---

## 9. 需要你完成的操作（未完成部分）

在远程仓库与凭据就绪前，我**不会也不能**声称推送成功。请执行：

```bash
# 1) 在 GitHub 创建仓库 xiaowork-dev/xiaowork-nav（不要初始化 README）
# 2) 关联远程并推送（本地已有完整提交历史）
cd xiaowork-nav
git remote add origin https://github.com/xiaowork-dev/xiaowork-nav.git
git branch -M main
git push -u origin main

# 3) 触发首次发布（Actions 会构建多架构镜像、推送 GHCR 并创建 Release）
git tag v2.0.0
git push origin v2.0.0

# 4) 推送后确认
#    - Actions 中 CI 与 Release 均通过
#    - GHCR 出现 ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.0 与 latest
#    - Release 页面出现 v2.0.0 及发行说明
```

若需要我改为使用不同的组织/仓库名，请告知；`deploy/.env.example` 与文档中的镜像地址需同步更新。

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

1. 你创建远程仓库并推送（第 9 节）。
2. CI/Release 首次运行后确认镜像与 Release 可用。
3. 在一台**独立测试服务器**上按 `docs/installation.md` 完成一次真实部署与 NPM 反代验证。
4. 反馈后进入后续版本（P1 增强项：多主题、书签导入、PWA 等，见 `docs/requirements.md` §2）。

---

## 附：本报告的编写依据

- 代码与文件：仓库实际文件（2026-10-25）。
- Docker 测试：`docker build`、`docker compose up/down`、`scripts/*.sh`、HTTP 接口的实际输出（见 4.2）。
- 测试记录：`go test -count=1 ./...`、`npm run build`、`bash -n`、`docker compose config` 的实际输出。
- Git 状态：`git log` / `git status` / `git remote` 实际输出（见第 6 节）。
