# XiaoWork Nav — Frontend

Vue 3 + TypeScript + Vite 前端。**只通过 `/api/v1` 与后端通信**，可独立启动与调试。

> 面向其他 AI Agent 的完整规范见 [`../docs/frontend-agent-guide.md`](../docs/frontend-agent-guide.md)。

## 已实现功能

导航首页（时钟、搜索、分类图标磁贴）、初始化向导、管理员登录/退出、分类管理（含拖拽排序）、
网站管理（含图标上传/抓取、跨分类移动）、系统设置、数据导入导出与备份、浅色/深色主题、响应式布局。

## 独立启动

```bash
npm install
npm run dev        # http://localhost:5173
```

开发服务器会把 `/api` 与 `/healthz` 代理到后端（默认 `http://localhost:8080`）。
可通过 `VITE_DEV_PROXY_TARGET` 覆盖后端地址，通过 `VITE_API_BASE_URL` 覆盖运行时 API 基地址。

## 构建

```bash
npm run build      # 类型检查 + 产物输出到 dist/
```

生产环境由 Go 服务托管 `dist/`，无需 Node 运行时。

## 契约

- 契约文件：`../docs/openapi/openapi.yaml`
- 生成类型：`npm run gen:types`
- Mock 联调：`npm run mock`（Prism，端口 8080）

## 规则

1. 所有 HTTP 请求必须走 `src/api/`。
2. 组件/页面不得直接使用 `fetch`。
3. 不依赖后端内部结构，仅依赖契约。
