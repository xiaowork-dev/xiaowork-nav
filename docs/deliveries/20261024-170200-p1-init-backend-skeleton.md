# 开发交付：P1 项目初始化与后端基础架构

- 时间：2026-10-24 17:02（Asia/Shanghai）
- 任务编号/关联需求：P1（需求梳理、契约冻结与项目初始化）
- 开发分支：`master`（本地 Git 仓库，尚无提交）
- 交付状态：已完成（Docker 镜像构建一项为「未验证」）

## 1. 目标与验收标准

- 目标：补齐 P1 剩余文档（需求、DDL、交付），初始化 Git 仓库与项目骨架，交付可运行的 Go 后端基础架构与可构建的 Vue 前端骨架，并提供 Docker 一键部署工件。
- 验收标准：
  1. `docs/requirements.md`、`docs/database-design.md` DDL、契约一致性检查齐备。
  2. `go build`/`go vet`/`go test ./...` 通过。
  3. 后端可启动，`/healthz` 正常。
  4. 前端 `npm run build`（含类型检查）通过。
  5. 前后端可独立启动；Go 可托管前端产物。
  6. Compose 配置与部署脚本有效。
  7. 每阶段交付文档与进度总览更新。

## 2. 实际完成内容

- **文档**：需求清单（含游客/私有可见性模型）、旧站视觉与交互参考分析、API 契约一致性检查、部署指南；更新开发计划与文档索引。
- **数据库**：`0001_init` 迁移（users/sessions/categories/bookmarks/site_settings/uploads）+ down 迁移；版本化迁移执行器（embed + `schema_migrations`，失败即停）。
- **后端**：配置加载（环境变量）、SQLite 连接（外键 + WAL + busy_timeout）、HTTP 路由与中间件（请求 ID、日志、恢复、安全头、体积限制、CORS）、系统接口（`/healthz`、`/api/v1/version`、`/api/v1/auth/status`）、SPA 静态托管（embed + 回退）、优雅退出。
- **前端**：Vue 3 + TS + Vite 骨架，含 API 客户端层（`src/api`）、类型定义、路由、首页与 404 页、基础主题样式、开发代理。
- **Git**：`git init`、`.gitignore`、`.gitattributes`、`.dockerignore`、README、LICENSE(MIT)、CHANGELOG、CONTRIBUTING。
- **部署**：多阶段 `Dockerfile`、`deploy/compose.yaml`、`deploy/compose.npm.yaml`、`deploy/.env.example`、`scripts/deploy.sh`、`scripts/update.sh`。

## 3. 关键文件与技术实现

| 文件/目录 | 修改类型 | 原因与实现摘要 |
| --- | --- | --- |
| `docs/requirements.md` | 新增 | P0/P1 需求与访问模型 |
| `docs/legacy-analysis.md` | 新增 | 视觉/交互参考与迁移假设（不复用旧源码） |
| `docs/api-contract-review.md` | 新增 | 需求→接口覆盖矩阵，无缺口 |
| `docs/deployment.md` | 新增 | 部署、更新、排障与环境变量 |
| `docs/openapi/openapi.yaml` | 修改 | 单资源更新改 PATCH；补 `default` 错误响应、`X-Requested-With`、`operationId`、具名 Schema |
| `docs/PROGRESS.md` | 修改 | 同步最新进度 |
| `docs/README.md`、`docs/development-plan-v2.md` | 修改 | 索引与 P1 任务拆分更新 |
| `backend/go.mod`、`go.sum` | 新增 | 模块 `github.com/xiaowork-dev/xiaowork-nav/backend`，依赖 `modernc.org/sqlite`（纯 Go） |
| `backend/cmd/server/main.go` | 新增 | 启动编排：配置→建目录→开库→迁移→HTTP→优雅退出 |
| `backend/internal/config/config.go` | 新增 | 环境变量配置（`XN_*`），含校验与目录创建 |
| `backend/internal/store/store.go` | 新增 | SQLite 打开与 PRAGMA（foreign_keys/WAL/busy_timeout） |
| `backend/internal/migrations/migrations.go` | 新增 | 版本化迁移执行器（embed、事务、幂等、失败即停） |
| `backend/internal/migrations/sql/0001_init.up.sql` / `.down.sql` | 新增 | 初始表结构与回滚 |
| `backend/internal/api/{server,middleware,response}.go` | 新增 | 路由、中间件链、统一错误信封 |
| `backend/internal/static/static.go` + `dist/index.html` | 新增 | 前端 embed 与 SPA 回退（占位文件保证 `go build` 通过） |
| `backend/internal/**/*_test.go` | 新增 | config/store/migrations/api 单元与接口测试 |
| `frontend/**` | 新增 | Vue 3 + TS + Vite 骨架与 API 客户端层 |
| `Dockerfile`、`deploy/*`、`scripts/deploy.sh`、`scripts/update.sh` | 新增 | 单容器构建与部署/更新 |
| `.gitignore`、`.gitattributes`、`.dockerignore`、`README.md`、`LICENSE`、`CHANGELOG.md`、`CONTRIBUTING.md` | 新增 | 仓库基础文件 |

## 4. 验证记录

| 检查项 | 实际命令/操作 | 结果 | 备注 |
| --- | --- | --- | --- |
| 后端构建 | `go build ./...` | 通过 | backend |
| 后端静态检查 | `go vet ./...` | 通过 | backend |
| 后端测试 | `go test ./...` | 通过 | api/config/migrations/store 全部 ok |
| 迁移幂等与级联 | `go test ./internal/migrations` | 通过 | 重复 Apply 无变更；删除分类级联删导航项 |
| 后端运行时冒烟 | 启动二进制 + HTTP 请求 | 通过 | `/healthz`=200、`/api/v1/version` schema_version=1、`/api/v1/auth/status` 正常、`/api/v1/nope`=404 JSON |
| 前端类型检查+构建 | `npm run build` | 通过 | `vue-tsc --noEmit` + `vite build` |
| 前后端集成 | 复制前端产物到 embed 目录后重启后端 | 通过 | `/` 返回 SPA、`/assets/*.js` MIME 正确、深链接回退 `index.html` |
| OpenAPI 解析与引用 | `js-yaml` 解析 + `$ref` 检查 | 通过 | 29 路径、23 Schema、无悬空引用 |
| Compose 配置校验 | `docker compose config --quiet` | 通过 | `compose.yaml` 与 `compose.npm.yaml` |
| 部署脚本语法 | `bash -n scripts/*.sh` | 通过 | deploy.sh / update.sh |
| **Docker 镜像构建** | `docker build ...` | **未运行** | 本机 Docker 守护进程不可用（`dockerDesktopLinuxEngine` 管道不存在） |
| OpenAPI 规范级 lint | `redocly lint`/`spectral` | 未运行 | 未安装；列入 P2 |

> 局限：Docker 镜像构建、容器启动与全新服务器部署未在本环境验证；未执行任何生产部署。

## 5. 部署、数据与兼容性影响

- Docker/环境变量/端口/网络：新增单容器镜像与两套 Compose（独立模式默认 `127.0.0.1:8787`；NPM 模式不发布端口）。全部参数经 `XN_*` 环境变量注入，未写死域名/网络名。
- 数据库迁移及回滚风险：`0001_init` 为可逆迁移；启动时自动执行，失败拒绝启动。备份/恢复的应用内实现（`VACUUM INTO` 等）在 P2 完成；`update.sh` 目前以「停容器后快照数据目录」保证一致性。
- 安全与敏感信息检查：仓库未包含任何密钥；`.env` 已忽略；`.dockerignore` 排除 `.env`/数据/构建产物；容器以非 root 运行。

## 6. 遗留问题与风险

- **Docker 镜像构建未验证**：现象为守护进程不可用；影响为镜像可能需在 P5 首次构建时修正；后续在具备 Docker 的环境构建并实测。
- **旧站 `data.json` 缺失**：`legacy-v1` 导入器无法完成字段映射与测试；待样本。
- **OpenAPI 规范级 lint 未接入**：P2 引入到 CI。
- **Git 提交身份为占位**：`xiaowork-dev <xiaowork-dev@users.noreply.github.com>`，待用户提供真实身份后替换。
- **前端为骨架**：仅验证连通性与构建，业务页面在 P3 实现。

## 7. 下一步

1. 提交本轮成果到本地 Git（首次提交）。
2. 进入 P2：认证（初始化管理员、登录、会话、改密）、分类与导航项 CRUD、排序、搜索、系统设置。
3. 引入 OpenAPI lint 与契约一致性测试到 CI。
4. 实现图标上传/抓取（防 SSRF）与导入导出。
5. 取得旧站数据样本；确认 GitHub 仓库名与 NPM 网络名。

## 8. 关联文档

- [当前项目进度](../PROGRESS.md)
- [开发计划](../development-plan-v2.md)
- [需求清单](../requirements.md)
- [旧站参考分析](../legacy-analysis.md)
- [API 规范](../api-spec.md) · [契约一致性检查](../api-contract-review.md)
- [数据库设计](../database-design.md)
- [部署指南](../deployment.md)
- [前端交接规范](../frontend-agent-guide.md)

> 备注：本文件不静态记录提交 SHA 或推送状态；当前为本地仓库首次提交前，尚未配置远程，相关事实在最终回复中说明。
