# P1 阶段交付报告：需求梳理、契约冻结与项目初始化

> 本报告是 **P1 阶段的正式、可长期归档交付文档**。
> 报告内容基于实际代码、文件与测试记录编写；未实现或未验证的内容均明确标注，未标记为完成。

---

## 1. 阶段基本信息

| 项 | 内容 |
| --- | --- |
| 项目名称 | XiaoWork Nav |
| 项目版本 | V2.0（后端 `0.1.0-dev`，前端 `0.1.0`） |
| 阶段编号与名称 | **P1 需求梳理、契约冻结与项目初始化** |
| 阶段目标 | 在编码前确定需求、数据库与 API 契约；初始化 Git 仓库与项目骨架；交付可运行的 Go 后端基础架构、可构建的 Vue 前端骨架与 Docker 部署工件 |
| 阶段开始日期 | 2026-10-24（架构与计划调整：16:31；项目初始化：17:02，Asia/Shanghai） |
| 阶段完成日期 | 2026-10-24 |
| 当前完成状态 | **已完成**（其中「Docker 镜像构建实测」为 **未验证**，不计入已完成） |
| 上位依据 | `XiaoWork_Nav_V2_重构方案与实施计划.md`、`docs/development-plan-v2.md` |
| 关联交付记录 | [架构与开发计划调整](20261024-163148-v2-planning-adjustment.md)、[项目初始化与后端基础架构](20261024-170200-p1-init-backend-skeleton.md) |

---

## 2. 本阶段工作总结

### 2.1 实际完成的开发任务

1. **开发策略与架构调整**：由"前后端同期推进"改为**后端优先、API 驱动、严格前后端分离**；确立 OpenAPI 3.x 为唯一契约；确认访客/私有可见性模型。
2. **需求梳理**：输出 P0/P1 需求清单与角色访问模型。
3. **旧站参考分析**：确认不复用旧源码，仅提取视觉与交互参考，并预留旧数据导入方案。
4. **数据库设计**：冻结 7 张表结构与索引约束，落地 `0001_init` 迁移与回滚脚本。
5. **API 契约**：编写 API 设计规范与 OpenAPI 3.1 契约，并完成需求→接口一致性检查（无缺口）。
6. **项目初始化**：`git init`、仓库基础文件（`.gitignore`/`.gitattributes`/`.dockerignore`/README/LICENSE/CHANGELOG/CONTRIBUTING）。
7. **后端基础架构**：配置、SQLite、版本化迁移、HTTP 路由与中间件、系统接口、SPA 托管、优雅退出，并配套单元/接口测试。
8. **前端骨架**：Vue 3 + TS + Vite，含 API 客户端层、类型、路由、页面骨架与主题变量。
9. **部署工件**：多阶段 `Dockerfile`、两套 Compose、环境变量模板、部署与更新脚本。
10. **架构决策记录**：ADR-0001 ~ ADR-0004。

### 2.2 已实现的功能（可用且已验证）

| 功能 | 说明 |
| --- | --- |
| 服务启动与优雅退出 | `cmd/server` 完成依赖组装与信号处理 |
| 环境变量配置 | `XN_*` 全部配置项，含校验与默认值 |
| SQLite 持久化 | 纯 Go 驱动、外键开启、WAL、busy_timeout |
| 版本化迁移 | embed + `schema_migrations`，事务执行，失败拒绝启动，幂等 |
| 健康检查 | `GET /healthz` |
| 版本信息 | `GET /api/v1/version`（含 `schema_version`） |
| 初始化状态 | `GET /api/v1/auth/status`（返回 `initialized`/`authenticated`） |
| 统一错误信封 | `{ error: { code, message, details?, request_id? } }` |
| HTTP 中间件 | 请求 ID、结构化日志、panic 恢复、安全响应头、请求体限制、可配置 CORS |
| SPA 静态托管 | Go `embed` 托管前端产物 + 深链接回退 `index.html`，API 路径不拦截 |
| 前端骨架 | 首页可连接后端并展示版本/状态；404 页面；主题基础变量 |
| 部署工件 | Compose 配置校验通过；部署/更新脚本语法校验通过 |

### 2.3 核心技术方案与架构决策

| 决策 | 内容 | 记录 |
| --- | --- | --- |
| 后端优先、契约先行 | 先冻结契约再写实现与前端 | [ADR-0001](../decisions/ADR-0001-backend-first.md) |
| 单容器交付 | Go `embed` 托管前端，SQLite 挂载 `/data` | [ADR-0002](../decisions/ADR-0002-single-container-embed.md) |
| SQLite + 版本化迁移 | 失败即停、升级前备份、不可逆迁移靠快照 | [ADR-0003](../decisions/ADR-0003-sqlite-migrations.md) |
| OpenAPI 唯一契约 | 接口分级 `/public`、`/auth`、`/admin` | [ADR-0004](../decisions/ADR-0004-openapi-contract.md) |

技术选型：Go 1.27 标准库 HTTP（无第三方 Web 框架）+ `modernc.org/sqlite`（纯 Go，无需 cgo）；前端 Vue 3.5 + TypeScript 5.7 + Vite 6。

### 2.4 与原定计划的差异及原因

| 差异 | 原因 |
| --- | --- |
| 原 P1「项目骨架」并入 P1 阶段 | 用户要求"完成 P1 后继续执行 P1 阶段的项目初始化与后端基础架构开发" |
| 旧版源码分析 → 仅视觉/交互参考分析 | 用户确认本次为全新重构，不复用旧源码 |
| 单资源更新由 `PUT` 改为 `PATCH` | 消除"全量覆盖"歧义，明确部分字段更新语义 |
| 契约新增 `default` 错误响应 | 保证所有接口都有统一错误体，便于前端统一处理 |
| `backup.sh` / `restore.sh` 推迟到 P2 | 应用内备份功能尚未实现，避免交付空脚本 |
| 未进行高保真 UI 设计 | 用户明确要求不单独做耗时的高保真设计 |
| 未创建 GitHub 远程仓库 | 用户要求基础架构稳定后再配置 |

---

## 3. 代码和文件变更

### 3.1 新增的重要文件

**后端（Go，共 13 个 `.go` 文件 + 2 个 SQL）**

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `backend/cmd/server/main.go` | 120 | 启动编排、优雅退出、日志 |
| `backend/internal/config/config.go` | 129 | 环境变量配置 |
| `backend/internal/store/store.go` | 60 | SQLite 打开与 PRAGMA |
| `backend/internal/migrations/migrations.go` | 191 | 版本化迁移执行器 |
| `backend/internal/migrations/sql/0001_init.up.sql` | 63 | 初始表结构 |
| `backend/internal/migrations/sql/0001_init.down.sql` | 7 | 回滚 |
| `backend/internal/api/server.go` | 115 | 路由与系统接口 |
| `backend/internal/api/middleware.go` | 148 | 中间件链 |
| `backend/internal/api/response.go` | 94 | 统一响应与错误 |
| `backend/internal/static/static.go` | 49 | 前端 embed 与 SPA 回退 |
| `backend/internal/version/version.go` | 19 | 构建版本变量 |
| `backend/internal/**/*_test.go` | 361 | config/store/migrations/api 测试 |

**前端（Vue 3 + TS，共 9 个源文件）**

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `frontend/src/api/client.ts` | 83 | 唯一 HTTP 出口、错误归一化、CSRF 头 |
| `frontend/src/api/system.ts` | 12 | 系统接口封装 |
| `frontend/src/types/api.ts` | 91 | 与 OpenAPI 对齐的类型 |
| `frontend/src/router/index.ts` | 20 | 路由 |
| `frontend/src/pages/HomePage.vue` | 132 | 首页骨架（连通性展示） |
| `frontend/src/pages/NotFoundPage.vue` | 25 | 404 页面 |
| `frontend/src/styles/main.css` | 84 | 主题变量与基础样式 |
| `frontend/src/App.vue` / `src/main.ts` | 6 / 6 | 应用入口 |

**部署与仓库文件**

`Dockerfile`、`deploy/compose.yaml`、`deploy/compose.npm.yaml`、`deploy/.env.example`、`scripts/deploy.sh`、`scripts/update.sh`、`.gitignore`、`.gitattributes`、`.dockerignore`、`README.md`、`LICENSE`（MIT）、`CHANGELOG.md`、`CONTRIBUTING.md`。

**文档（`docs/`）**

`requirements.md`、`legacy-analysis.md`、`api-contract-review.md`、`deployment.md`、`architecture.md`、`api-spec.md`、`database-design.md`、`frontend-agent-guide.md`、`development-plan-v2.md`、`PROGRESS.md`、`README.md`、`openapi/openapi.yaml`（1013 行）、`decisions/ADR-0001~0004`、`deliveries/`（本报告与两条过程记录）。

### 3.2 修改的重要文件

| 文件 | 修改内容 |
| --- | --- |
| `XiaoWork_Nav_V2_重构方案与实施计划.md` | 顶部新增"计划调整说明"，指向 `docs/`，保留原文为总体依据 |
| `docs/api-spec.md` | 单资源更新改为 `PATCH`；排序/设置幂等语义更新 |
| `docs/openapi/openapi.yaml` | 补 `default` 错误响应、`X-Requested-With`、`operationId`、具名 Schema；`PUT`→`PATCH` |
| `docs/development-plan-v2.md` | P1 任务拆分扩展至 P1-8（含项目初始化） |
| `docs/PROGRESS.md`、`docs/README.md` | 同步进度与文档索引 |

### 3.3 各模块实际完成情况

| 模块 | 完成情况 |
| --- | --- |
| 配置 | ✅ 完成 |
| 存储（SQLite） | ✅ 完成（连接层；仓储 CRUD 属 P2） |
| 迁移 | ✅ 完成（执行器 + 初始迁移） |
| HTTP 基础设施 | ✅ 完成（路由/中间件/错误信封） |
| 认证 | ⛔ 未开始（仅 `auth/status` 返回未初始化） |
| 分类/导航项/搜索/设置 | ⛔ 未开始（P2） |
| 图标/导入导出/备份 | ⛔ 未开始（P2） |
| 静态托管 | ✅ 完成 |
| 前端页面与后台 | ⛔ 未开始（P3，仅骨架） |
| 部署工件 | 🟡 配置完成，镜像构建未验证 |

### 3.4 重要目录结构说明

```text
xiaowork-nav/
├── backend/          Go 服务（不依赖前端源码；仅 embed 前端构建产物）
│   ├── cmd/server/   进程入口
│   └── internal/     config / store / migrations / api / static / version
├── frontend/         Vue 3 前端（可独立启动，可被整体替换）
│   └── src/api/      唯一 HTTP 出口
├── deploy/           Compose 与环境变量模板
├── scripts/          部署与更新脚本
├── docs/             架构 / API / 数据库 / 部署 / 决策 / 进度 / 交付
└── Dockerfile        多阶段构建
```

---

## 4. 测试与验证结果

### 4.1 已执行的测试及实际结果

执行环境：Windows 11，Go `go1.27.3 windows/amd64`，Node `v24.22.1`，npm `11.19.1`。执行时间：2026-10-24。

| # | 检查项 | 实际命令/操作 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| 1 | 后端构建 | `go build ./...` | ✅ 通过 | 退出码 0 |
| 2 | 后端静态检查 | `go vet ./...` | ✅ 通过 | 退出码 0 |
| 3 | 后端测试 | `go test -count=1 ./...` | ✅ 通过 | `api` ok、`config` ok、`migrations` ok、`store` ok |
| 4 | 迁移幂等与级联 | `go test ./internal/migrations` | ✅ 通过 | 重复 `Apply` 无变更；删除分类级联删除导航项；7 张表存在 |
| 5 | 后端运行时冒烟 | 启动二进制 + HTTP 请求 | ✅ 通过 | `/healthz`=200；`/api/v1/version` `schema_version=1`；`/api/v1/auth/status` 正常；`/api/v1/nope`=404 JSON |
| 6 | 前端类型检查 + 构建 | `npm run build` | ✅ 通过 | `vue-tsc --noEmit` + `vite build`，产物 `dist/` |
| 7 | 前后端集成（Go 托管前端） | 复制前端产物到 embed 目录后重启后端 | ✅ 通过 | `/` 返回 SPA；`/assets/index-*.js` 返回 200 且 MIME `text/javascript`；`/admin/deep/route` 回退 `index.html` |
| 8 | OpenAPI YAML 解析 | `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过 | 退出码 0 |
| 9 | OpenAPI 引用完整性 | `js-yaml` 解析 + `$ref` 遍历 | ✅ 通过 | 29 路径、23 Schema、7 响应、33 处 `$ref`，无悬空引用 |
| 10 | Compose 配置校验 | `docker compose config --quiet` | ✅ 通过 | `compose.yaml` 与 `compose.npm.yaml` 均退出码 0 |
| 11 | 部署脚本语法 | `bash -n scripts/deploy.sh`、`update.sh` | ✅ 通过 | 退出码 0 |

### 4.2 未通过的测试

无。本次执行的全部检查项均通过。

### 4.3 尚未执行的测试及原因

| 未执行项 | 原因 | 影响 |
| --- | --- | --- |
| **Docker 镜像构建**（`docker build`） | 当前环境 Docker 守护进程不可用（`dockerDesktopLinuxEngine` 管道不存在，`docker info` 失败） | 镜像能否构建、容器能否启动、健康检查是否生效 **均未验证**；需在具备 Docker 的环境补做 |
| 容器启动与一键部署实测 | 同上 | `deploy.sh` / `update.sh` 的运行时行为未验证（仅语法与配置校验） |
| 升级 / 回滚 / 数据保留实测 | 同上，且备份功能属 P2 | 未验证 |
| OpenAPI 规范级 lint（`redocly` / `spectral`） | 工具未安装 | 契约仅通过语法与引用校验，未做规范级校验；列入 P2 CI |
| 旧站数据导入测试 | 无旧站 `data.json` 样本 | `legacy-v1` 导入器未实现、未测试 |
| 真实浏览器端到端 / 响应式视觉验收 | 前端仅骨架，页面属 P3 | 未验证 |

### 4.4 当前环境限制

- Docker 守护进程不可用 → 无法构建/运行镜像。
- 无 GitHub 远程仓库 → 无法推送。
- 无旧站数据样本 → 无法实现/验证旧站导入器。
- 未部署任何服务器。

> **声明：本报告未将任何未执行的测试标记为已通过。** 4.1 中的通过项均为本次实际执行并取得退出码 0 或预期响应。

---

## 5. 项目总体进度

### 5.1 整体开发阶段列表

| 阶段 | 名称 | 状态 |
| --- | --- | --- |
| **P1** | 需求梳理、契约冻结与项目初始化 | ✅ **已完成** |
| P2 | Go 后端完整实现 | ⛔ 未开始 |
| P3 | Vue 3 前端完整实现 | ⛔ 未开始 |
| P4 | UI、交互与主题完善 | ⛔ 未开始 |
| P5 | 部署、更新与发布 | ⛔ 未开始 |

### 5.2 已完成阶段

- **P1**：见本报告第 2、3 节。

### 5.3 当前所处阶段

- **P1 已完成**，处于 **P2 启动前**（本轮按要求暂停 P2）。

### 5.4 尚未开始的阶段与各阶段剩余任务

| 阶段 | 剩余任务概览 |
| --- | --- |
| P2 | 认证（初始化管理员/登录/会话/改密）、分类与导航项 CRUD、排序、搜索、系统设置、图标上传与抓取（防 SSRF）、导入导出、备份恢复；引入 OpenAPI lint 与契约一致性测试 |
| P3 | 导航首页、管理后台、网站管理、分类管理、响应式布局；仅依赖 `/api/v1` |
| P4 | 主题系统、视觉打磨、动效、移动端触控与可访问性 |
| P5 | 镜像构建实测、全新环境一键部署、更新/回滚实测、GitHub Actions + GHCR、Release |

---

## 6. Git 版本记录

> 说明：为遵守"不把提交 SHA 硬写进待提交文件"的原则，下列 SHA 为**本报告提交前**的仓库状态；本报告随本轮独立提交，其 SHA 在聊天总结中报告。

| 项 | 内容 |
| --- | --- |
| 当前分支 | `master` |
| 本报告提交前的最近提交 | `17323bb` — `chore: initialize XiaoWork Nav V2 project (P1)` |
| 提交前工作区状态 | 干净（`git status --short` 无输出） |
| 提交身份 | 本仓库本地占位：`xiaowork-dev <xiaowork-dev@users.noreply.github.com>` |
| 是否配置 GitHub 远程 | **否**（`git remote -v` 无输出） |
| 是否已完成推送 | **否**（无远程，未推送） |

---

## 7. 遗留问题和风险

### 7.1 未完成的任务

- P2 全部业务功能（认证、分类、导航项、搜索、设置、图标、导入导出、备份）。
- P3 前端页面与后台。
- P5 部署与发布（含镜像构建实测）。

### 7.2 已知问题

| 问题 | 现象 | 影响 | 后续处理 |
| --- | --- | --- | --- |
| Docker 守护进程不可用 | `docker info` 失败 | 镜像与部署未验证 | 在具备 Docker 的环境构建并实测（P5 必做） |
| `auth/status` 恒返回 `authenticated=false` | 认证未实现 | 不影响当前阶段 | P2 实现会话后替换 |
| 提交身份为占位 | 本地 Git 身份非用户真实身份 | 影响提交作者信息 | 用户提供后替换 |

### 7.3 尚未验证的功能

- Docker 镜像构建、容器启动、健康检查、一键部署、一键更新、数据保留与回滚。
- OpenAPI 规范级合规性。
- 旧站数据导入。
- 前端页面功能与响应式视觉（尚未实现）。

### 7.4 进入下一阶段的必要条件

| 条件 | 是否阻塞 P2 | 说明 |
| --- | --- | --- |
| 契约基线可用 | 否（已满足） | `openapi.yaml` 已通过语法与引用校验，P2 可据此实现 |
| 数据库 DDL 可用 | 否（已满足） | `0001_init` 迁移已通过测试 |
| 后端骨架可运行 | 否（已满足） | 已实测 |
| 旧站 `data.json` 样本 | 否（仅阻塞旧站导入器） | 影响 P2 中 `legacy-v1` 导入子项 |
| Docker 守护进程 | 否（阻塞 P5） | 影响镜像与部署验证 |
| GitHub 仓库名 | 否（阻塞推送） | 影响远程配置与推送 |
| 用户对契约冻结的最终确认 | 建议确认 | 不确认亦可按当前基线推进 |

**结论：P2 编码无硬性阻塞条件，可立即开始。**

---

## 8. 下一阶段计划（P2：Go 后端完整实现）

### 8.1 P2 阶段目标

实现完整、稳定、有测试的 Go 后端业务功能，使前端可仅依赖 `/api/v1` 完成全部管理操作。

### 8.2 具体开发任务

1. **认证与安全**：`POST /setup`（一次性初始化管理员，Argon2id）、`POST /auth/login`、`POST /auth/logout`、`GET /auth/me`、`PUT /auth/password`；会话表与 Cookie（HttpOnly/SameSite/Secure）、Origin/CSRF 校验、登录限流。
2. **分类管理**：`GET/POST /admin/categories`、`GET/PATCH/DELETE /admin/categories/{id}`、`PUT /admin/categories/order`。
3. **导航项管理**：`GET/POST /admin/bookmarks`、`GET/PATCH/DELETE /admin/bookmarks/{id}`、`PUT /admin/bookmarks/order`、`POST /admin/bookmarks/{id}/move`。
4. **公开接口**：`GET /public/navigation`、`/public/categories`、`/public/categories/{id}/bookmarks`、`/public/search`、`/public/settings`（仅返回可见数据）。
5. **系统设置**：`GET/PUT /admin/settings` 与公开设置过滤。
6. **图标**：`POST /admin/uploads/icon`（类型/体积校验）、`POST /admin/icons/fetch`（防 SSRF）。
7. **导入导出**：`GET /admin/export`、`POST /admin/import/preview`、`POST /admin/import`（`skip`/`overwrite`/`rename`，单事务，默认不覆盖）。
8. **备份恢复**：`GET/POST /admin/backups`、`POST /admin/backups/{id}/restore`（`VACUUM INTO` 或 Online Backup API）。
9. **契约与 CI**：引入 `redocly lint`/`spectral` 与契约一致性测试。

### 8.3 预期交付成果

- 可运行的完整后端，覆盖 `openapi.yaml` 中全部 P0 端点。
- 单元测试与接口测试；`go test ./...` 通过。
- 新增 `docs/deliveries/P2-delivery.md` 阶段交付报告，并更新 `docs/PROGRESS.md`。
- 相应数据库迁移（如会话/设置/上传所需）与文档同步。

### 8.4 验收标准

1. `go build ./...`、`go vet ./...`、`go test ./...` 全部通过。
2. OpenAPI 中全部 P0 端点可用且返回结构与契约一致。
3. 未授权请求无法修改数据；初始化仅允许一次。
4. 数据库迁移可重复执行且幂等；失败拒绝启动。
5. 公开接口只返回 `is_visible = true` 数据；私有数据仅管理员可见。
6. 契约一致性测试与 lint 在 CI 中通过。
7. 交付报告与进度总览已更新，且与实际代码、测试、Git 状态一致。

---

## 附：本报告的编写依据

- 代码与文件：`backend/`、`frontend/`、`deploy/`、`scripts/`、`docs/`（实际文件与行数统计）。
- 测试记录：2026-10-24 实际执行的命令与输出（见 4.1）。
- Git 状态：`git branch` / `git log` / `git status` / `git remote` 实际输出（见第 6 节）。
