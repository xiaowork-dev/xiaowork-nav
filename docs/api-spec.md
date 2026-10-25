# XiaoWork Nav V2.0：API 设计规范

> 文档类型：接口规范
> 版本：api-v1-baseline
> 更新日期：2026-10-24
> 机器可读契约：[`openapi/openapi.yaml`](openapi/openapi.yaml)
> 关联：`architecture.md`、`database-design.md`、`frontend-agent-guide.md`

---

## 1. 基本约定

| 项 | 约定 |
| --- | --- |
| 基础路径 | `/api/v1` |
| 传输 | HTTP/1.1（生产由反代终止 TLS） |
| 编码 | UTF-8 |
| 请求体 | `application/json`（文件上传除外，使用 `multipart/form-data`） |
| 响应体 | `application/json` |
| 时间格式 | RFC 3339 UTC，如 `2026-10-24T08:31:48Z` |
| ID 类型 | 整数（SQLite 自增主键），长期稳定 |
| 命名风格 | JSON 字段使用 `snake_case` |
| 版本策略 | 破坏性变更升级到 `/api/v2`；`/api/v1` 内只做向后兼容的字段新增 |

### 1.1 健康检查（不带版本前缀）

- `GET /healthz` → `200 {"status":"ok","version":"2.0.0"}`，用于 Docker/反代健康检查。
- 该端点不属于业务 API，不纳入 `/api/v1`。

---

## 2. 认证与鉴权

### 2.1 机制

- 管理接口基于**服务端会话 + HttpOnly Cookie**。
- Cookie 属性：`HttpOnly`、`SameSite=Lax`、生产 HTTPS 下 `Secure`。
- 浏览器前端无需手动携带令牌；跨源开发时使用 `credentials: 'include'`。
- 所有**状态变更请求**（POST/PUT/PATCH/DELETE）必须通过 `Origin`/`Referer` 校验；前端应携带 `X-Requested-With: XMLHttpRequest` 作为附加 CSRF 信号。

### 2.2 接口分级

| 级别 | 前缀 | 说明 |
| --- | --- | --- |
| 公开 | `/api/v1/public/*` | 匿名可读，仅返回 `is_visible=true` 的数据 |
| 认证 | `/api/v1/auth/*` | 登录、登出、会话状态、初始化 |
| 管理 | `/api/v1/admin/*` | 需要有效会话，全部写操作在此 |

> 公开接口与鉴权接口物理隔离，便于其他 Agent 快速判断权限要求。

### 2.3 认证相关端点

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| GET | `/api/v1/auth/status` | 返回 `{initialized, authenticated, user?}` | 公开 |
| POST | `/api/v1/setup` | 首次初始化管理员（仅在未初始化时可用） | 公开（一次性） |
| POST | `/api/v1/auth/login` | 登录 | 公开（限流） |
| POST | `/api/v1/auth/logout` | 登出 | 认证 |
| GET | `/api/v1/auth/me` | 当前用户信息 | 认证 |
| PUT | `/api/v1/auth/password` | 修改密码 | 认证 |

---

## 3. 成功响应格式

采用 RESTful 风格：**成功响应直接返回资源对象，HTTP 状态码表达结果。**

- 单对象：直接返回对象。
  ```json
  { "id": 1, "name": "AI & 智能", "sort_order": 10, "is_visible": true }
  ```
- 集合：返回带分页元信息的包装对象。
  ```json
  {
    "items": [ /* ... */ ],
    "total": 42,
    "page": 1,
    "page_size": 20
  }
  ```
- 无返回体操作（删除、排序提交）：`204 No Content` 或 `200 {"ok": true}`（见各端点说明）。

### 3.1 状态码约定

| 码 | 含义 |
| --- | --- |
| 200 | 成功 |
| 201 | 创建成功（返回新资源） |
| 204 | 成功无内容（删除等） |
| 400 | 参数校验失败 |
| 401 | 未认证 / 会话失效 |
| 403 | 已认证但无权限 / CSRF 校验失败 |
| 404 | 资源不存在 |
| 409 | 冲突（如重复用户名、初始化已完成、导入冲突） |
| 413 | 请求体或上传文件过大 |
| 422 | 语义错误（如非法 URL 协议） |
| 429 | 触发限流 |
| 500 | 服务器内部错误 |

---

## 4. 错误响应格式

统一错误信封：

```json
{
  "error": {
    "code": "validation_failed",
    "message": "请求参数校验失败",
    "details": [
      { "field": "url", "issue": "仅允许 http/https 协议" }
    ],
    "request_id": "b1f2c3d4"
  }
}
```

- `code`：稳定的机器可读错误码（见附录 A）。
- `message`：面向人的简要说明（可本地化，前端不应依赖其内容做逻辑）。
- `details`：可选，字段级错误列表。
- `request_id`：可选，用于日志关联。

---

## 5. 分页、排序与过滤

- 列表接口通用查询参数：
  - `page`（默认 1）、`page_size`（默认 20，最大 100）。
  - `sort`（如 `sort_order`、`created_at`、`title`）、`order`（`asc`/`desc`）。
  - `q`：模糊搜索关键字。
  - `category_id`、`is_visible`：过滤条件。
- 排序类写操作使用**显式排序提交**而非逐条移动：
  - `PUT /api/v1/admin/categories/order`，body：`{"items":[{"id":1,"sort_order":10}, ...]}`
  - `PUT /api/v1/admin/bookmarks/order`，同上。
  - 服务端在单事务内应用，保证一致性。

---

## 6. 端点清单（P0）

### 6.1 公开接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/public/navigation` | 首页一次性获取分类 + 导航项树（仅可见） |
| GET | `/api/v1/public/categories` | 可见分类列表 |
| GET | `/api/v1/public/categories/{id}/bookmarks` | 某分类下可见导航项 |
| GET | `/api/v1/public/search?q=` | 搜索可见导航项 |
| GET | `/api/v1/public/settings` | 公开站点设置（标题、Favicon、主题、搜索引擎等） |
| GET | `/api/v1/public/uploads/{filename}` | 获取上传的图标文件（`icon_type=upload`） |
| GET | `/api/v1/version` | 当前版本与构建信息 |

### 6.2 管理接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/categories` | 全部分类（含隐藏） |
| POST | `/api/v1/admin/categories` | 新建分类 |
| GET | `/api/v1/admin/categories/{id}` | 分类详情 |
| PATCH | `/api/v1/admin/categories/{id}` | 更新分类（部分字段） |
| DELETE | `/api/v1/admin/categories/{id}` | 删除分类（含引用处理策略） |
| PUT | `/api/v1/admin/categories/order` | 提交分类排序 |
| GET | `/api/v1/admin/bookmarks` | 导航项列表（分页/过滤） |
| POST | `/api/v1/admin/bookmarks` | 新建导航项 |
| GET | `/api/v1/admin/bookmarks/{id}` | 导航项详情 |
| PATCH | `/api/v1/admin/bookmarks/{id}` | 更新导航项（部分字段） |
| DELETE | `/api/v1/admin/bookmarks/{id}` | 删除导航项 |
| PUT | `/api/v1/admin/bookmarks/order` | 提交导航项排序 |
| POST | `/api/v1/admin/bookmarks/{id}/move` | 移动到其它分类 |
| GET | `/api/v1/admin/settings` | 读取全部设置 |
| PUT | `/api/v1/admin/settings` | 批量更新设置 |
| POST | `/api/v1/admin/uploads/icon` | 上传图标文件 |
| POST | `/api/v1/admin/icons/fetch` | 服务端抓取站点图标（防 SSRF，返回引用不落库） |
| GET | `/api/v1/admin/icons/summary` | 图标覆盖率统计 |
| POST | `/api/v1/admin/icons/bulk` | 启动批量补全缺失图标（异步、单任务、限频） |
| GET | `/api/v1/admin/icons/bulk` | 当前或最近一次批量任务状态（含进度与失败明细） |
| POST | `/api/v1/admin/icons/bulk/cancel` | 取消正在运行的批量任务 |
| POST | `/api/v1/admin/bookmarks/{id}/icon/auto` | 为单个网站自动获取并保存图标（`force` 可强制刷新） |
| GET | `/api/v1/admin/export` | 导出 JSON（含版本号、校验） |
| POST | `/api/v1/admin/import/preview` | 导入预览（冲突检测） |
| POST | `/api/v1/admin/import` | 执行导入（冲突策略） |
| GET | `/api/v1/admin/backups` | 备份快照列表 |
| POST | `/api/v1/admin/backups` | 立即创建备份 |
| POST | `/api/v1/admin/backups/{id}/restore` | 从快照恢复 |
| GET | `/api/v1/admin/update-check` | 检查最新版本信息（只读） |

> 旧站 JSON 导入器复用 `/admin/import/preview` 与 `/admin/import`，通过 `format` 字段区分（`xiaowork-nav-v1` / `legacy-v1`）。

---

## 7. 请求/响应示例

### 7.1 登录

```http
POST /api/v1/auth/login
Content-Type: application/json

{ "username": "admin", "password": "********" }
```

```json
200 OK
{ "user": { "id": 1, "username": "admin" } }
```

失败：

```json
401 Unauthorized
{ "error": { "code": "invalid_credentials", "message": "用户名或密码错误" } }
```

### 7.2 获取公开导航

```json
200 OK
{
  "categories": [
    {
      "id": 1, "name": "AI & 智能", "description": "", "sort_order": 10,
      "bookmarks": [
        { "id": 7, "title": "ChatGPT", "url": "https://chat.openai.com",
          "icon_type": "url", "icon_value": "https://.../favicon.ico",
          "open_in_new_tab": true, "sort_order": 10 }
      ]
    }
  ],
  "settings": { "site_title": "XiaoWork Nav", "theme": "auto" }
}
```

### 7.3 提交排序

```http
PUT /api/v1/admin/bookmarks/order
Content-Type: application/json

{ "items": [ { "id": 7, "sort_order": 10 }, { "id": 3, "sort_order": 20 } ] }
```

```json
200 OK
{ "ok": true, "updated": 2 }
```

---

## 8. 数据校验规则

| 字段 | 规则 |
| --- | --- |
| `username` | 3–32 字符，`[A-Za-z0-9_-]` |
| `password` | 最少 8 字符；哈希使用 Argon2id |
| `url` | 必填，仅 `http:`/`https:`，长度 ≤ 2048 |
| `title` | 1–128 字符，去除首尾空白 |
| `name`（分类） | 1–64 字符 |
| `description` | ≤ 512 字符 |
| `keywords` | ≤ 512 字符，逗号分隔 |
| `icon_type` | `none` / `url` / `upload` / `auto` |
| `icon_value` | 依 `icon_type` 而定，长度 ≤ 2048 |
| 上传图标 | ≤ 1 MiB；允许 `image/png`、`image/jpeg`、`image/svg+xml`、`image/webp`、`image/x-icon` |
| 请求体上限 | 1 MiB（上传端点除外） |

---

## 9. 幂等与并发

- `PATCH` 更新为幂等（仅提交的字段被修改）；`POST` 创建非幂等。
- `PUT /admin/settings` 为"部分更新"语义：只写入请求中出现的键。
- 排序提交整体幂等（重复提交相同顺序结果一致）。
- 导入在单事务内执行；预览不写库。
- 不引入乐观锁字段；并发编辑以"最后写入生效"为准（单管理员场景可接受），文档中明确说明。

---

## 10. 契约维护流程

1. 任何接口变更先改 `docs/openapi/openapi.yaml`。
2. 使用 `redocly lint` / `spectral` 校验（P1 引入 CI 检查）。
3. 后端以契约为准实现；前端用 `openapi-typescript` 生成类型。
4. 破坏性变更需在 `docs/decisions/` 记录并升级 API 版本。

---

## 附录 A：错误码表

| code | HTTP | 含义 |
| --- | --- | --- |
| `validation_failed` | 400 | 参数校验失败 |
| `invalid_credentials` | 401 | 用户名或密码错误 |
| `unauthorized` | 401 | 未登录或会话失效 |
| `forbidden` | 403 | 无权限 |
| `csrf_failed` | 403 | Origin/CSRF 校验失败 |
| `not_found` | 404 | 资源不存在 |
| `already_initialized` | 409 | 已完成初始化，拒绝重复初始化 |
| `username_taken` | 409 | 用户名已存在 |
| `import_conflict` | 409 | 导入冲突需确认策略 |
| `payload_too_large` | 413 | 请求体/文件过大 |
| `invalid_url_scheme` | 422 | 非法 URL 协议 |
| `unsupported_media_type` | 422 | 不支持的图标格式 |
| `rate_limited` | 429 | 触发限流 |
| `internal_error` | 500 | 服务器内部错误 |

## 附录 B：公开设置字段（`/public/settings`）

| 字段 | 说明 |
| --- | --- |
| `site_title` | 站点标题 |
| `site_favicon` | Favicon 地址 |
| `theme` | `light` / `dark` / `auto` |
| `background` | 背景配置（颜色/图片/渐变） |
| `search_engines` | 搜索引擎列表 |
| `default_search_engine` | 默认搜索引擎 |
| `open_in_new_tab_default` | 默认打开方式 |
| `homepage_layout` | 首页展示模式 |
