# XiaoWork Nav V2.0：前端交接与开发规范（面向其他 AI Agent）

> 文档类型：交接规范
> 更新日期：2026-10-24
> 适用对象：**只负责 `frontend/` 的前端开发者或 AI Agent**
> 关联：`api-spec.md`、`openapi/openapi.yaml`、`architecture.md`

---

## 0. 给接手者的最重要三句话

1. **你只需要改 `frontend/`，不需要读后端源码。**
2. **你只需要理解 `/api/v1` 和 OpenAPI 契约，不需要理解数据库。**
3. **你重构前端后，后端与数据保持不变，用户数据不会丢失。**

---

## 1. 你拥有的资源

| 资源 | 位置 | 用途 |
| --- | --- | --- |
| 接口契约 | `docs/openapi/openapi.yaml` | 唯一权威接口定义 |
| 接口说明 | `docs/api-spec.md` | 约定、错误码、示例 |
| 前端目录 | `frontend/` | 你的工作区 |
| Mock 服务 | 由 OpenAPI 生成（P1 提供） | 无后端时联调 |
| 真实后端 | `http://localhost:8080` | 端到端联调 |

---

## 2. 你不需要关心的东西

- Go 源码、SQLite 表结构、迁移脚本。
- 部署、Docker、CI/CD。
- 后端如何存储图标、如何抓取 favicon。

> 若某个功能在契约中缺失，**不要自行猜测后端**，应在交付说明中标注"契约缺口"，由后端补充 OpenAPI 后再实现。

---

## 3. 环境与启动

### 3.1 独立启动（推荐开发方式）

```bash
cd frontend
npm install
npm run dev          # Vite 开发服务器，默认 http://localhost:5173
```

开发代理（`vite.config.ts`）将 `/api` 与 `/healthz` 代理到后端：

```ts
server: {
  port: 5173,
  proxy: {
    '/api':     { target: 'http://localhost:8080', changeOrigin: true },
    '/healthz': { target: 'http://localhost:8080', changeOrigin: true },
  },
}
```

- 后端未启动时，可切换到 Mock 服务（见第 5 节）。
- 通过环境变量覆盖 API 地址：`VITE_API_BASE_URL`（默认空，即同源）。

### 3.2 生产

- `npm run build` 产物由 Go 服务托管，前端**不需要**独立部署。
- 运行时不需要 Node。

---

## 4. 技术栈与目录约定

- Vue 3（`<script setup>` + TypeScript）+ Vite + Pinia + Vue Router。
- 样式：CSS 变量优先，避免重型 UI 库。

```text
frontend/src/
├── api/            # 唯一 HTTP 出口
│   ├── client.ts   # fetch 封装：baseURL、credentials、错误归一化
│   ├── auth.ts
│   ├── public.ts
│   └── admin.ts
├── types/          # 与 OpenAPI 对齐（openapi-typescript 生成）
├── stores/         # Pinia
├── router/         # 路由与鉴权守卫
├── pages/          # 路由页面
├── components/     # 通用组件
├── composables/
└── styles/
```

---

## 5. 契约驱动开发流程

1. 从 `docs/openapi/openapi.yaml` 生成类型：
   ```bash
   npx openapi-typescript ../docs/openapi/openapi.yaml -o src/types/api.d.ts
   ```
2. 用类型编写 `src/api/*` 封装，**所有请求必须走 `api/` 层**。
3. 无后端时启动 Mock：
   ```bash
   npx @stoplight/prism-cli mock ../docs/openapi/openapi.yaml -p 8080
   ```
4. 联调完成后，与真实后端做端到端验证。

---

## 6. 必须遵守的规则

| 规则 | 原因 |
| --- | --- |
| 所有 HTTP 请求集中在 `src/api/` | 便于统一鉴权、错误处理、替换后端 |
| 组件/页面不得直接 `fetch` | 同上 |
| 类型来自 OpenAPI 生成物 | 保证与后端一致 |
| 不依赖后端内部字段/表结构 | 保证前端可整体替换 |
| 不新增后端接口需求而不更新契约 | 契约唯一 |
| 使用 Cookie 会话，请求带 `credentials: 'include'` | 鉴权方式由后端定义 |
| 状态变更请求带 `X-Requested-With: XMLHttpRequest` | CSRF 信号 |
| 不硬编码后端 URL | 通过 `VITE_API_BASE_URL` / 代理 |
| 不提交密钥或 `.env` | 安全 |

---

## 7. 错误处理约定

统一处理后端错误信封（见 `api-spec.md` 第 4 节）：

```ts
type ApiError = {
  error: {
    code: string;
    message: string;
    details?: Array<{ field: string; issue: string }>;
    request_id?: string;
  };
};
```

- `401` → 清除会话状态并跳转登录。
- `403 csrf_failed` → 提示刷新页面重试。
- `400/422` → 依据 `details` 做字段级提示。
- `429` → 提示稍后重试。

---

## 8. 数据兼容性保证

- 前端重构**只影响界面**，数据通过同一套 API 读写，SQLite 数据不变。
- 路由与后端接口解耦：前端可自由设计 URL（如 `/admin/*`），只要调用相同 API。
- 主题、设置等通过 `/api/v1/admin/settings` 与 `/api/v1/public/settings` 读写，格式以契约与 `database-design.md` 第 5 节为准。

---

## 9. 前端验收标准

1. 仅修改 `frontend/` 即可完成全部 P0 前端功能。
2. `npm run build` 成功，产物可被 Go 托管。
3. 开发模式可独立启动并完成端到端流程（登录→管理→保存→前台可见）。
4. 无任何直接数据库访问或后端源码依赖。
5. 桌面与移动端均可用。

---

## 10. 交付要求

- 每完成一个前端里程碑，输出交付说明（可复用 `docs/deliveries/` 模板）。
- 若发现契约缺口，单独列出，等待后端补充。
- 不修改 `backend/`、`docs/openapi/openapi.yaml`（除非获得授权并走契约流程）。

---

## 附录：常用命令

```bash
npm install
npm run dev
npm run build
npm run preview
npx openapi-typescript ../docs/openapi/openapi.yaml -o src/types/api.d.ts
npx @stoplight/prism-cli mock ../docs/openapi/openapi.yaml -p 8080
```
