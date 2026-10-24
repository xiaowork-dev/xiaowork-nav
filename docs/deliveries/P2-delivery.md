# P2 阶段交付报告：Go 后端核心业务开发

> 本报告是 **P2 阶段的正式、可长期归档交付文档**。
> 内容基于实际代码、测试记录与 Git 状态编写；未实现或未验证的内容均明确标注，未标记为完成。

---

## 1. 阶段基本信息

| 项 | 内容 |
| --- | --- |
| 项目名称 | XiaoWork Nav |
| 项目版本 | V2.0（后端 `0.1.0-dev`，数据库结构版本 `1`） |
| 阶段编号与名称 | **P2 Go 后端核心业务开发** |
| 阶段目标 | 按需求文档与 OpenAPI 契约实现完整可用的 Go 后端：认证与会话、分类与导航项 CRUD、排序、搜索、系统设置、图标、导入导出、备份恢复、统一校验与错误处理，并配套自动化测试（含权限边界） |
| 阶段开始日期 | 2026-10-24（P1 验收后启动） |
| 阶段完成日期 | 2026-10-24 |
| 当前完成状态 | **已完成**（其中「Docker 镜像构建实测」「旧站真实数据导入验证」「OpenAPI 规范级 lint」为 **未验证/未执行**，不计入已完成） |
| 上位依据 | `docs/development-plan-v2.md`、`docs/requirements.md`、`docs/openapi/openapi.yaml`、`docs/AGENTS.md`（交付规范） |
| 关联交付记录 | [P1 阶段交付报告](P1-delivery.md) |

---

## 2. 本阶段工作总结

### 2.1 实际完成的开发任务

1. **认证与会话**：一次性管理员初始化（Argon2id）、登录/登出、服务端会话 + HttpOnly/SameSite Cookie、会话滑动续期、改密并失效全部会话、登录限流。
2. **分类管理**：CRUD、排序、公开/私有可见性、级联删除导航项。
3. **导航项管理**：CRUD、排序、跨分类移动、可见性、图标（none/url/upload/auto）、关键词、打开方式。
4. **导航站搜索**：公开搜索（仅可见数据）与管理端过滤搜索。
5. **系统设置**：公开视图与管理视图分离、部分更新、取值校验、默认值。
6. **数据校验与统一错误**：字段级校验、稳定错误码、统一错误信封、HTTP 状态映射。
7. **安全机制**：Origin/Referer 校验（CSRF）、请求体限制、URL 协议白名单、SSRF 防护的图标抓取、上传类型/体积校验。
8. **图标**：本地上传与公开读取、服务端抓取站点图标。
9. **导入导出**：JSON 导出（含格式/版本/校验和）、导入预览、导入执行（冲突策略、单事务、默认不覆盖）、旧站 `legacy-v1` 宽松映射。
10. **备份恢复**：`VACUUM INTO` 快照（含图标目录）、快照列表、恢复（结构版本校验 + 自动安全快照）。
11. **更新检查**：可选 GitHub Release 只读检查。
12. **测试**：51 个后端测试，覆盖权限边界、可见性、校验、导入导出与备份恢复。

### 2.2 已实现的功能（已验证）

| 功能 | 验证方式 |
| --- | --- |
| 初始化/登录/登出/会话/改密 | 接口测试 + 真实二进制冒烟 |
| 未授权无法访问管理接口 | 接口测试（16 个管理端点逐一断言 401） |
| 分类 CRUD / 排序 / 可见性 | 接口测试 |
| 导航项 CRUD / 排序 / 移动 / URL 校验 | 接口测试 |
| 公开导航不泄漏私有数据 | 接口测试（导航树与搜索结果） |
| 系统设置默认值/校验/公开过滤 | 接口测试 + 服务测试 |
| 图标上传校验与公开读取 | 接口测试 |
| 图标抓取拒绝内网/元数据地址 | 接口测试 |
| 导出/导入（原生与 legacy） | 接口测试 |
| 备份创建/列表/恢复 | 接口测试 |
| Origin/CSRF 校验 | 接口测试 + 真实二进制冒烟 |

### 2.3 核心技术方案与架构决策

- **分层**：`api`（HTTP）→ `service`（业务）→ `store`（数据），单向依赖；`service` 不感知 HTTP，`api` 负责把 `service.Error` 映射为状态码与错误码。
- **认证**：Argon2id（PHC 格式、随机盐）+ 服务端会话表 + HttpOnly Cookie；改密后删除该用户全部会话。
- **CSRF**：对状态变更请求校验 `Origin`/`Referer` 与请求 Host 是否一致；无 Origin/Referer 的非浏览器客户端放行。
- **SSRF**：在 `net.Dialer.Control` 中校验**已解析的目标 IP**（可防 DNS 重绑定），禁止环回/私有/链路本地/云元数据等网段，并限制重定向。
- **原子导入**：在 `store` 层用单个 `*sql.Tx` 完成分类/导航项/设置写入，保证全有或全无。
- **备份**：`VACUUM INTO` 生成一致性快照（不停止服务），快照旁保存图标目录副本；恢复前自动生成安全快照，并校验快照结构版本与当前一致。
- **可见性模型**：`is_visible = false` 即"私有，仅管理员可见"；公开接口仅返回可见分类与可见导航项。

### 2.4 与原定计划的差异及原因

| 差异 | 原因 |
| --- | --- |
| 新增 `GET /api/v1/public/uploads/{filename}` | 上传图标需要公开读取路径，否则前端无法显示本地图标；已同步更新 OpenAPI 与 API 规范 |
| `internal/icons`、`internal/importexport`、`internal/backup` 合并进 `internal/service` | 减少包数量、避免过早拆分；已在 `architecture.md` 记录实际结构 |
| 备份快照包含图标目录副本（`<id>.files/`） | 满足"数据库与上传图标一起备份"的要求 |
| 限流同时按 IP 与用户名 | 兼顾分布式爆破防护；副作用是可能被用于短时锁定管理员账号（见风险） |
| `PATCH` 部分更新采用"仅写入提交字段"语义 | 与契约一致，避免全量覆盖 |

---

## 3. 代码和文件变更

### 3.1 新增的重要文件

**新增包**

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `backend/internal/models/models.go` | 141 | 领域模型（与契约字段对齐） |
| `backend/internal/auth/password.go` | 77 | Argon2id 哈希与校验 |
| `backend/internal/auth/session.go` | 98 | 会话标识与登录限流器 |
| `backend/internal/security/url.go` | 39 | URL 协议白名单校验 |
| `backend/internal/security/ssrf.go` | 116 | SSRF 防护 HTTP 客户端 |
| `backend/internal/service/errors.go` | 56 | 服务层类型化错误 |
| `backend/internal/service/auth.go` | 222 | 认证与会话业务 |
| `backend/internal/service/catalog.go` | 503 | 分类/导航项/搜索业务 |
| `backend/internal/service/settings.go` | 210 | 系统设置业务 |
| `backend/internal/service/icons.go` | 219 | 图标上传与抓取 |
| `backend/internal/service/transfer.go` | 250 | 导入导出 |
| `backend/internal/service/backup.go` | 253 | 备份与恢复 |

**存储层（新增仓储）**

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `backend/internal/store/repo.go` | 38 | Store 类型与工具 |
| `backend/internal/store/users.go` | 72 | 用户仓储 |
| `backend/internal/store/sessions.go` | 66 | 会话仓储 |
| `backend/internal/store/categories.go` | 155 | 分类仓储 |
| `backend/internal/store/bookmarks.go` | 220 | 导航项仓储与导航聚合 |
| `backend/internal/store/settings.go` | 48 | 设置仓储 |
| `backend/internal/store/uploads.go` | 68 | 上传元数据仓储 |
| `backend/internal/store/import.go` | 226 | 事务化导入与预览 |

**API 层（新增 handler）**

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `backend/internal/api/auth_handlers.go` | 151 | 认证接口 |
| `backend/internal/api/authmw.go` | 173 | 认证/来源校验中间件与 Cookie |
| `backend/internal/api/helpers.go` | 110 | JSON/分页/路径参数辅助 |
| `backend/internal/api/public_handlers.go` | 81 | 公开接口 |
| `backend/internal/api/admin_categories.go` | 86 | 分类管理接口 |
| `backend/internal/api/admin_bookmarks.go` | 112 | 导航项管理接口 |
| `backend/internal/api/admin_settings.go` | 132 | 设置/图标/更新检查接口 |
| `backend/internal/api/admin_transfer.go` | 60 | 导入导出接口 |
| `backend/internal/api/admin_backups.go` | 33 | 备份接口 |
| `backend/internal/api/maintenance.go` | 27 | 会话清理与限流清理 |

**测试**

| 文件 | 行数 | 覆盖 |
| --- | --- | --- |
| `backend/internal/api/p2_test.go` | 696 | 权限边界、认证流程、CRUD、可见性、搜索、设置、图标、导入导出、备份 |
| `backend/internal/service/service_test.go` | 192 | 限流、会话生命周期、设置校验、目录校验 |
| `backend/internal/auth/password_test.go` | 85 | 哈希/校验/会话 ID/限流 |
| `backend/internal/security/security_test.go` | 65 | URL 校验与 IP 黑名单 |

### 3.2 修改的重要文件

| 文件 | 修改内容 |
| --- | --- |
| `backend/internal/config/config.go` | 新增 `XN_SECURE_COOKIE`、`XN_BACKUP_KEEP`、`XN_UPDATE_REPO` |
| `backend/internal/api/server.go` | 注册全部 P2 路由、组装服务、挂载来源校验中间件 |
| `backend/internal/api/middleware.go` | 上传请求放宽体积限制（1 MiB + multipart 开销） |
| `backend/internal/api/response.go` | 增加 `service.Error` → HTTP 状态/错误码映射 |
| `backend/cmd/server/main.go` | 启动维护协程（会话清理）、非 Secure Cookie 告警 |
| `docs/openapi/openapi.yaml` | 新增 `/public/uploads/{filename}` |
| `deploy/.env.example`、`deploy/compose.yaml`、`deploy/compose.npm.yaml` | 新增环境变量 |
| `docs/{api-spec,api-contract-review,architecture,deployment}.md` | 同步接口、模块结构与环境变量 |
| `CHANGELOG.md` | 记录 P2 变更 |

### 3.3 各模块实际完成情况

| 模块 | 完成情况 |
| --- | --- |
| 配置 | ✅ 完成（新增 3 个配置项） |
| 存储（SQLite 仓储） | ✅ 完成 |
| 迁移 | ✅ 完成（P2 未新增迁移；沿用 `0001_init`） |
| HTTP 基础设施 | ✅ 完成 |
| 认证与会话 | ✅ 完成 |
| 分类 / 导航项 / 排序 / 搜索 | ✅ 完成 |
| 系统设置 | ✅ 完成 |
| 图标（上传/抓取） | ✅ 完成 |
| 导入导出（含 legacy） | ✅ 完成（legacy 仅通用映射，未经真实样本验证） |
| 备份恢复 | ✅ 完成 |
| 更新检查 | ✅ 完成（未配置仓库时不请求网络） |
| 前端页面 | ⛔ 未开始（P3） |
| Docker 镜像 / 部署 | ⏳ 未验证（沿用 P1 工件，本阶段未重新构建） |

### 3.4 重要目录结构说明

```text
backend/internal/
├── api/          HTTP 路由、中间件、handler（按资源分文件）
├── auth/         密码哈希、会话标识、登录限流
├── config/       环境变量配置
├── migrations/   版本化 SQL（embed）
├── models/       领域模型（契约字段对齐）
├── security/     URL 校验与 SSRF 防护
├── service/      业务规则（auth/catalog/settings/icons/transfer/backup）
├── static/       前端 embed 与 SPA 回退
├── store/        数据访问（users/sessions/categories/bookmarks/settings/uploads/import）
└── version/      构建版本变量
```

后端 Go 代码合计约 **6474 行**（含 8 个测试文件）。

---

## 4. 测试与验证结果

执行环境：Windows 11，Go `go1.27.3 windows/amd64`。执行时间：2026-10-24。

### 4.1 已执行的测试及实际结果

| # | 检查项 | 实际命令/操作 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| 1 | 后端构建 | `go build ./...` | ✅ 通过 | 退出码 0 |
| 2 | 后端静态检查 | `go vet ./...` | ✅ 通过 | 退出码 0 |
| 3 | 后端测试（全量） | `go test -count=1 ./...` | ✅ 通过 | api/auth/config/migrations/security/service/store 全部 ok |
| 4 | 测试数量统计 | `go test -count=1 -v ./...` | ✅ 51 通过 / 0 失败 | 统计 `--- PASS` / `--- FAIL` |
| 5 | 权限边界 | `internal/api` 测试 | ✅ 通过 | 16 个管理端点未登录均返回 401 且错误码为 `unauthorized` |
| 6 | 可见性隔离 | `internal/api` 测试 | ✅ 通过 | 公开导航/搜索不返回私有分类与私有导航项 |
| 7 | URL 协议白名单 | 接口测试 + 安全测试 | ✅ 通过 | `javascript:`/`file:`/`ftp:` 被拒绝（422 `invalid_url_scheme`） |
| 8 | SSRF 防护 | 接口测试 + 安全测试 | ✅ 通过 | `127.0.0.1`、`169.254.169.254`、`10.0.0.5`、`file:` 全部被拒 |
| 9 | CSRF/Origin 校验 | 接口测试 + 真实二进制冒烟 | ✅ 通过 | 跨域写请求 403 `csrf_failed` |
| 10 | 登录限流 | 服务测试 | ✅ 通过 | 超过阈值返回 `rate_limited`；新限流器可正常登录 |
| 11 | 导入导出往返 | 接口测试 | ✅ 通过 | 导出含 `format`/`checksum`；导入 1 分类 1 导航项；重复导入 `skipped=2` |
| 12 | legacy 导入 | 接口测试 | ✅ 通过 | `links` + `name/link` 别名映射成功 |
| 13 | 备份创建/列表/恢复 | 接口测试 | ✅ 通过 | 恢复后数据回到快照状态（2 → 1 分类） |
| 14 | 真实二进制端到端冒烟 | 启动二进制 + HTTP 调用 | ✅ 通过 | 见 4.1.1 |
| 15 | OpenAPI YAML 解析 | `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过 | 退出码 0 |
| 16 | Compose 配置校验 | `docker compose config --quiet` | ✅ 通过 | `compose.yaml`、`compose.npm.yaml` 均退出码 0 |

#### 4.1.1 真实二进制冒烟结果

以 `XN_SECURE_COOKIE=false` 启动构建产物后，实际观测：

```text
GET /healthz                      -> ok
POST /api/v1/setup                -> user=admin
GET /api/v1/auth/me               -> admin
POST /api/v1/admin/categories     -> id=1
POST /api/v1/admin/bookmarks      -> id=1
GET /api/v1/public/navigation     -> 1 category, first bookmark "ChatGPT"
GET /api/v1/public/settings       -> site_title "XiaoWork Nav"
GET /api/v1/public/search?q=chat  -> total=1
POST /api/v1/admin/backups        -> backup-0.1.0-dev-20261024-094928.db
GET /api/v1/admin/export          -> format=xiaowork-nav-v1, checksum=sha256:67a24bb...
POST /api/v1/auth/logout          -> 204
GET /api/v1/admin/categories      -> HTTP 401
POST cross-origin write           -> HTTP 403
GET /api/v1/public/navigation     -> 1 category (guest still allowed)
```

### 4.2 未通过的测试

无。本阶段执行的全部检查项均通过。

### 4.3 尚未执行的测试及原因

| 未执行项 | 原因 | 影响 |
| --- | --- | --- |
| **Docker 镜像构建与容器启动** | 当前环境 Docker 守护进程不可用（`dockerDesktopLinuxEngine` 管道不存在） | P1 的 Docker 工件未随 P2 重新构建验证；属 P5 必做项 |
| **旧站真实 `data.json` 导入验证** | 用户未提供旧站数据样本 | `legacy-v1` 仅按通用映射实现，字段映射未经真实数据验证 |
| **OpenAPI 规范级 lint**（`redocly`/`spectral`） | 工具未安装 | 契约仅通过 YAML 语法解析，未做规范级校验 |
| **前端对接 P2 接口的端到端测试** | 前端仍为 P1 骨架，页面属 P3 | 未验证 |
| **图标抓取对真实公网站点的成功路径** | 为避免测试依赖外网，仅测试了拒绝内网的路径 | 成功路径未在自动化测试中验证 |
| **备份/恢复在真实 Docker 卷上的行为** | 无 Docker 环境 | 未验证 |
| **响应式/浏览器端验证** | 无前端页面 | 未验证 |

### 4.4 当前环境限制

- Docker 守护进程不可用 → 无法构建/运行镜像。
- 无 GitHub 远程仓库 → 无法推送。
- 无旧站数据样本 → 无法验证 legacy 导入的真实字段映射。
- 无外网依赖测试 → 图标抓取成功路径与更新检查未做联网验证。

> **声明：本报告未将任何未执行的测试标记为已通过。** 4.1 中的通过项均为本次实际执行并取得预期结果。

---

## 5. 项目总体进度

### 5.1 整体开发阶段列表

| 阶段 | 名称 | 状态 |
| --- | --- | --- |
| P1 | 需求梳理、契约冻结与项目初始化 | ✅ 已完成 |
| **P2** | Go 后端核心业务开发 | ✅ **已完成** |
| P3 | Vue 3 前端完整实现 | ⛔ 未开始 |
| P4 | UI、交互与主题完善 | ⛔ 未开始 |
| P5 | 部署、更新与发布 | ⛔ 未开始 |

### 5.2 已完成阶段

- **P1**：见 [P1 阶段交付报告](P1-delivery.md)。
- **P2**：见本报告第 2、3、4 节。

### 5.3 当前所处阶段

- **P2 已完成**，处于 **P3 启动前**。

### 5.4 尚未开始的阶段与剩余任务

| 阶段 | 剩余任务概览 |
| --- | --- |
| P3 | 导航首页（分类/图标/搜索/主题）、管理后台（登录/初始化向导/设置）、网站管理（CRUD/拖拽排序）、分类管理、响应式布局；仅依赖 `/api/v1` |
| P4 | 主题系统、视觉打磨、动效、移动端触控与可访问性 |
| P5 | 镜像构建实测、全新环境一键部署、更新/回滚实测、GitHub Actions + GHCR、Release |

---

## 6. Git 版本记录

> 说明：为遵守"不把提交 SHA 硬写进待提交文件"的原则，下列 SHA 为**本报告提交前**的仓库状态；本报告随 P2 代码一并提交，其 SHA 在聊天总结中报告。

| 项 | 内容 |
| --- | --- |
| 当前分支 | `master` |
| 本报告提交前的最近提交 | `1b3c106` — `docs: add P1 phase delivery report and project AGENTS.md` |
| 提交前工作区状态 | 有未提交变更（P2 代码与文档，39 项变更） |
| 提交身份 | 本仓库本地占位：`xiaowork-dev <xiaowork-dev@users.noreply.github.com>` |
| 是否配置 GitHub 远程 | **否**（`git remote -v` 无输出） |
| 是否已完成推送 | **否**（无远程，未推送） |

---

## 7. 遗留问题和风险

### 7.1 未完成的任务

- P3 前端页面、P4 UI 完善、P5 部署与发布。
- OpenAPI 规范级 lint 与契约一致性自动测试（列入 P5/CI）。

### 7.2 已知问题

| 问题 | 现象 | 影响 | 后续处理 |
| --- | --- | --- | --- |
| Docker 守护进程不可用 | `docker info` 失败 | 镜像与部署未验证 | P5 在具备 Docker 的环境构建并实测 |
| 登录限流可被用于短时锁定管理员 | 连续失败登录后按用户名限流 | 单管理员场景下约 5 分钟内无法登录 | 已记录；如需可改为"仅 IP 限流 + 指数退避"，或提供 CLI 重置 |
| legacy 导入字段映射未经真实数据验证 | 仅有通用映射 | 真实旧站数据可能需补充别名 | 取得 `data.json` 后补充映射与测试 |
| 备份快照的 `size` 仅统计数据库文件 | 图标目录副本未计入 `size` | 展示的备份体积偏小 | 后续可在元数据中增加图标体积统计 |
| 恢复会整体替换图标目录 | 快照中不含某个图标时会被删除 | 与快照保持一致，符合预期但需在文档说明 | 已在 `deployment.md`/本报告说明 |
| 提交身份为占位 | 本地 Git 身份非用户真实身份 | 影响提交作者信息 | 用户提供后替换 |

### 7.3 尚未验证的功能

- Docker 镜像构建、容器启动、一键部署、更新与回滚。
- 旧站真实数据导入。
- 图标抓取对真实公网站点的成功路径。
- 更新检查对 GitHub API 的实际调用。
- OpenAPI 规范级合规性。

### 7.4 进入下一阶段的必要条件

| 条件 | 是否阻塞 P3 | 说明 |
| --- | --- | --- |
| 后端 P0 接口可用 | 否（已满足） | 全部 P0 端点已实现并通过测试 |
| 契约与实现一致 | 否（已满足） | 新增上传路径已同步契约 |
| 前端可独立启动 | 否（已满足） | P1 骨架已具备开发代理 |
| 旧站 `data.json` 样本 | 否 | 仅影响 legacy 导入子项 |
| Docker 守护进程 | 否（阻塞 P5） | 影响镜像与部署验证 |
| GitHub 仓库名 | 否（阻塞推送） | 影响远程配置 |

**结论：P3 编码无硬性阻塞条件，可立即开始。**

---

## 8. 下一阶段计划（P3：Vue 3 前端完整实现）

### 8.1 P3 阶段目标

在仅修改 `frontend/` 的前提下，实现功能完整、与后端严格解耦、可独立启动调试的前端，覆盖导航首页与管理后台。

### 8.2 具体开发任务

1. **基础设施**：API 客户端层完善（`src/api`）、类型与 OpenAPI 对齐（`openapi-typescript`）、Pinia 状态、路由与鉴权守卫、统一错误处理。
2. **导航首页**：分类分组、图标磁贴、实时搜索、搜索引擎切换、主题切换、响应式布局。
3. **初始化与登录**：首次初始化向导（`/setup`）、登录/登出、会话状态（`/auth/status`）。
4. **管理后台**：分类管理、网站管理（含拖拽排序与移动端替代操作）、图标设置（上传/URL/抓取）、站点设置。
5. **数据管理**：导出、导入预览与执行、备份列表与恢复。
6. **验收**：对接真实后端完成端到端流程（初始化 → 登录 → 建分类/网站 → 前台可见 → 导出/导入）。

### 8.3 预期交付成果

- 可构建、可独立启动的前端，覆盖 P0 页面与后台功能。
- 与 OpenAPI 契约一致的类型与调用。
- 新增 `docs/deliveries/P3-delivery.md` 阶段交付报告，并更新 `docs/PROGRESS.md`。

### 8.4 验收标准

1. `npm run build`（含类型检查）通过。
2. 仅修改 `frontend/` 即可完成全部 P0 前端功能。
3. 开发模式可独立启动并完成端到端流程。
4. 无任何直接数据库访问或后端源码依赖。
5. 桌面与移动端均可用。
6. 交付报告与进度总览已更新，且与实际代码、测试、Git 状态一致。

---

## 附：本报告的编写依据

- 代码与文件：`backend/` 实际文件与行数统计（2026-10-24）。
- 测试记录：`go build`/`go vet`/`go test -count=1 ./...`、真实二进制冒烟、`js-yaml`、`docker compose config` 的实际输出。
- Git 状态：`git log`/`git status`/`git remote` 实际输出（见第 6 节）。
