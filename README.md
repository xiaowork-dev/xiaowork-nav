# XiaoWork Nav

一个轻量、可自托管的个人导航站：后端 Go + SQLite，前端 Vue 3 + TypeScript，单容器 Docker 一键部署。

> 状态：**开发中（P1 项目初始化阶段）**。当前已具备可运行的后端骨架与前端骨架，业务功能正在按阶段实现。

## 特性（目标）

- 分类 + 图标卡片式导航首页，搜索框与多搜索引擎切换，响应式布局。
- 管理后台：分类、导航项、排序、图标、站点设置的完整可视化管理。
- 认证：首次初始化管理员、登录/登出、会话管理。
- 数据安全：SQLite 持久化、JSON 导入导出、备份与恢复。
- 游客可浏览公开导航，私有分类/网站仅管理员可见。
- 一键部署（Docker Compose）与一键更新，数据不随更新丢失。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3 + TypeScript + Vite |
| 后端 | Go（标准库 HTTP） |
| 数据库 | SQLite（WAL，纯 Go 驱动） |
| 部署 | Docker / Docker Compose |
| CI/CD | GitHub Actions + GHCR |

## 目录结构

```text
frontend/    Vue 3 前端（可独立启动、可被整体替换）
backend/     Go 后端（不依赖前端）
deploy/      Docker Compose 与部署配置
scripts/     部署/更新脚本
docs/        架构、API、数据库、进度与交付文档
```

## 开发环境

### 后端

```bash
cd backend
go run ./cmd/server          # 默认 http://localhost:8080
```

健康检查：`GET http://localhost:8080/healthz`

### 前端

```bash
cd frontend
npm install
npm run dev                  # http://localhost:5173，自动代理 /api 到 :8080
```

前后端严格分离：前端只通过 `/api/v1` 与后端通信，契约见 [`docs/openapi/openapi.yaml`](docs/openapi/openapi.yaml)。

## 生产部署

```bash
cd deploy
cp .env.example .env
# 编辑 .env：设置 APP_VERSION、数据目录等
../scripts/deploy.sh
```

详见 [docs/deployment.md](docs/deployment.md)（待 P5 完善）。

## 文档

- [文档索引](docs/README.md)
- [开发计划](docs/development-plan-v2.md)
- [总体架构](docs/architecture.md)
- [API 规范](docs/api-spec.md)
- [数据库设计](docs/database-design.md)
- [前端交接规范](docs/frontend-agent-guide.md)
- [项目进度](docs/PROGRESS.md)

## 安全

- 不提交任何密钥或真实 `.env`（见 `.gitignore`）。
- 认证使用 HttpOnly Cookie；书签 URL 限制 `http`/`https` 协议。
- 部署参数全部通过环境变量注入，不写死域名与网络名。

## 许可证

[MIT](LICENSE)
