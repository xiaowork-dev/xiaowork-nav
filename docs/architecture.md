# XiaoWork Nav V2.0：总体架构与前后端模块边界

> 文档类型：架构基线
> 更新日期：2026-10-24
> 上位依据：`../XiaoWork_Nav_V2_重构方案与实施计划.md`
> 关联：`api-spec.md`、`database-design.md`、`frontend-agent-guide.md`、`decisions/`

---

## 1. 架构总原则

1. **后端优先、API 驱动**：先冻结 API 契约，再实现后端，最后实现前端。
2. **严格前后端分离**：前端与后端是独立目录、独立构建、独立调试的两个应用，仅通过 REST API 通信。
3. **契约唯一**：`docs/openapi/openapi.yaml` 是前后端唯一事实来源，后端实现与前端调用都必须与之一致。
4. **前端可整体替换**：前端不依赖后端内部代码；其他 AI Agent 可只改 `frontend/` 完成重构，数据与后端保持兼容。
5. **单容器交付**：生产环境由 Go 服务统一托管前端静态资源与 API，SQLite 持久化。

---

## 2. 运行时架构

```text
Browser (desktop / mobile)
          |
        HTTPS (由用户反代终止)
          |
Reverse proxy (NPM / Nginx / Caddy)
          |
   private Docker network
          |
  xiaowork-nav container  (Go 单进程)
      |                |
  Vue 静态资源      REST API /api/v1
  (Go embed)           |
                   SQLite (/data/app.db, WAL)
                       |
                   /data/uploads  (图标)
                   /data/backups  (快照)
```

- 容器内只有一个 Go 进程；开发环境下前端由 Vite 独立启动（见 `frontend-agent-guide.md`）。
- 静态资源与 API 同源，避免生产环境跨域与 Cookie 问题。
- `/healthz` 用于部署与健康检查；非 `/api/*` 且非静态文件的路径回退到 `index.html`（SPA fallback）。

---

## 3. 目录结构

```text
xiaowork-nav/
├── frontend/                    # 可独立启动、可被整体替换
│   ├── src/
│   │   ├── api/                 # 唯一 HTTP 出口（API 客户端）
│   │   ├── pages/               # 路由页面
│   │   ├── components/          # 通用组件
│   │   ├── composables/         # 组合式逻辑
│   │   ├── stores/              # 状态管理
│   │   ├── router/
│   │   ├── types/               # 与 OpenAPI 对齐的类型
│   │   └── styles/
│   ├── index.html
│   ├── vite.config.ts
│   ├── package.json
│   └── README.md                # 前端独立开发说明
├── backend/                     # Go 服务，不引用任何前端代码
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/                 # 路由、handler、DTO、中间件
│   │   ├── auth/                # 密码哈希、会话标识、登录限流
│   │   ├── config/
│   │   ├── store/               # 存储层（仓储接口 + SQLite 实现）
│   │   ├── service/             # 领域服务（认证、目录、设置、图标、导入导出、备份）
│   │   ├── models/              # 领域模型（与契约字段对齐）
│   │   ├── security/            # URL 校验与 SSRF 防护
│   │   ├── migrations/          # 版本化 SQL（embed）
│   │   ├── static/              # 前端产物 embed + SPA fallback
│   │   └── version/
│   ├── go.mod
│   └── openapi/                 # 由 docs/openapi 同步/校验
├── deploy/
│   ├── compose.yaml
│   ├── compose.npm.yaml
│   └── .env.example
├── scripts/
│   ├── install.sh
│   ├── update.sh
│   ├── backup.sh
│   └── restore.sh
├── docs/
│   ├── PROGRESS.md
│   ├── development-plan-v2.md
│   ├── architecture.md
│   ├── api-spec.md
│   ├── database-design.md
│   ├── frontend-agent-guide.md
│   ├── openapi/openapi.yaml
│   ├── decisions/
│   └── deliveries/
├── .github/workflows/
├── Dockerfile
├── README.md
├── CHANGELOG.md
└── LICENSE
```

---

## 4. 后端模块边界

后端遵循 **handler → service → store** 单向依赖，禁止反向引用。

| 模块 | 职责 | 允许依赖 | 禁止依赖 |
| --- | --- | --- | --- |
| `cmd/server` | 组装依赖、启动 HTTP、优雅退出 | 所有 internal | — |
| `internal/api` | HTTP 路由、请求解析、DTO 转换、中间件、错误映射 | service、auth、models | store 的具体实现（通过接口） |
| `internal/service` | 业务规则、事务编排、校验 | store 接口、models、auth | api（HTTP 细节） |
| `internal/store` | 持久化、SQL、迁移 | models、migrations | service、api |
| `internal/auth` | 密码哈希、会话、Cookie、CSRF | store 接口、models | api |
| `internal/config` | 环境变量/配置 | — | 其他业务模块 |
| `internal/models` | 领域模型（与契约字段对齐） | — | 其他业务模块 |
| `internal/auth` | 密码哈希（Argon2id）、会话标识、登录限流 | — | api、store |
| `internal/security` | URL 协议校验、SSRF 防护的 HTTP 客户端 | — | api、store |
| `internal/service` | 业务规则：认证、目录（分类/导航项/搜索）、设置、图标、导入导出、备份 | store、models、auth、security、config | api |
| `internal/icons` / `internal/importexport` / `internal/backup` | **实现上归入 `internal/service`**（图标、导入导出、备份） | store、models、security | api |
| `internal/static` | 托管前端产物、SPA fallback | embed 资源 | 业务模块 |

**关键约束：**

- `api` 层只暴露 OpenAPI 定义的字段，不直接返回数据库行结构。
- 领域模型与 API DTO 分离，避免数据库字段名泄漏到接口。
- 所有数据库访问经由 `store`，业务层不写裸 SQL。
- 后端不得引入任何前端框架或页面概念。

---

## 5. 前端模块边界

| 目录 | 职责 | 约束 |
| --- | --- | --- |
| `src/api` | 封装所有 `/api/v1` 调用、错误归一化、鉴权携带 | **唯一允许发 HTTP 请求的层** |
| `src/types` | 与 OpenAPI 对齐的类型定义 | 由 `openapi-typescript` 生成或手工对齐 |
| `src/stores` | 全局状态（会话、设置、导航数据） | 通过 `api` 层取数 |
| `src/pages` | 路由级页面 | 不直接调用 `fetch` |
| `src/components` | 展示与交互组件 | 无副作用、可复用 |
| `src/composables` | 可复用逻辑 | 通过 `api`/`stores` 取数 |
| `src/styles` | 主题与样式（CSS 变量） | 不耦合后端 |

**关键约束：**

- 前端不得直接访问数据库、不得依赖后端源码或内部数据结构。
- 前端只依赖 `/api/v1` 与 OpenAPI 契约。
- 前端可独立启动、独立构建、独立测试。
- 前端重构不得要求后端改动（除非契约本身需要演进，走版本化流程）。

---

## 6. 前后端交互契约

1. **唯一入口**：所有业务交互走 `/api/v1`。
2. **唯一契约**：`docs/openapi/openapi.yaml`。
3. **鉴权**：管理接口基于会话 Cookie（`HttpOnly`），浏览器自动携带；公开接口匿名可读。
4. **错误格式**：统一错误信封（见 `api-spec.md`）。
5. **数据兼容**：前端重构后沿用同一份 SQLite 数据，不需要迁移。

---

## 7. 部署边界

- 生产：单容器，Go 托管前端与 API，SQLite 与上传目录挂载到 `/data`。
- 开发：后端 `go run`（:8080）+ 前端 Vite（:5173，代理 `/api` → 后端）。
- 前端构建产物在 CI/镜像构建阶段生成，由 Go `embed` 打包；**运行时不需要 Node**。
- 部署与更新细节见 `../XiaoWork_Nav_V2_重构方案与实施计划.md` 第 5、6 节与后续 `deployment` 文档。

---

## 8. 关键架构决策（摘要）

详见 `decisions/` 目录：

| ADR | 决策 |
| --- | --- |
| [ADR-0001](decisions/ADR-0001-backend-first.md) | 后端优先、API 驱动、契约先行 |
| [ADR-0002](decisions/ADR-0002-single-container-embed.md) | 单容器交付，Go embed 托管前端 |
| [ADR-0003](decisions/ADR-0003-sqlite-migrations.md) | SQLite + 版本化迁移 + 升级前备份 |
| [ADR-0004](decisions/ADR-0004-openapi-contract.md) | OpenAPI 作为前后端唯一契约与前端解耦基础 |

---

## 9. 安全边界（架构层）

- 认证 Cookie：`HttpOnly` + 生产 `Secure` + `SameSite=Lax`；状态变更接口校验 `Origin`/CSRF。
- 只允许 `http:`/`https:` 书签协议，拒绝 `javascript:` 等。
- 图标抓取必须防 SSRF（拒绝内网/环回/元数据 IP、限制重定向、限制响应大小）。
- `/data`、`.env`、备份文件不得进入静态资源目录。
- 上传文件校验 MIME、扩展名、内容与体积。
- 容器以非 root 运行，最小权限。
