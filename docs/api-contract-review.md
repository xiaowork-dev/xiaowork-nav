# XiaoWork Nav V2.0：API 契约一致性检查

> 文档类型：契约评审
> 更新日期：2026-10-24
> 对象：[`openapi/openapi.yaml`](openapi/openapi.yaml)
> 结论：**覆盖 V2.0 全部 P0 功能，可作为基线；待最终确认后冻结。**

---

## 1. 检查目标

确认当前 OpenAPI 契约能够覆盖 V2.0 核心功能，并明确公开/鉴权接口边界，避免前后端契约不一致。

---

## 2. 覆盖度矩阵（需求 → 接口）

| 需求编号 | 功能 | 覆盖接口 | 状态 |
| --- | --- | --- | --- |
| F-01 | 首页分类+导航树 | `GET /public/navigation` | ✅ |
| F-02 | 图标多来源 | `Bookmark.icon_type`（none/url/upload/auto） | ✅ |
| F-03 | 搜索 | `GET /public/search` | ✅ |
| F-04 | 多搜索引擎 | `PublicSettings.search_engines` | ✅ |
| F-05 | 新标签打开 | `Bookmark.open_in_new_tab` | ✅ |
| F-06 | 分类导航 | `GET /public/categories` | ✅ |
| F-09 | 主题切换 | `PublicSettings.theme` | ✅ |
| F-10 | 站点标题/Favicon | `PublicSettings.site_title/site_favicon` | ✅ |
| A-01 | 初始化管理员 | `GET /auth/status`、`POST /setup` | ✅ |
| A-02 | 登录/登出 | `POST /auth/login`、`POST /auth/logout` | ✅ |
| A-03 | 修改密码 | `PUT /auth/password` | ✅ |
| A-04 | 分类 CRUD | `GET/POST /admin/categories`、`GET/PATCH/DELETE /admin/categories/{id}` | ✅ |
| A-05 | 分类排序/显隐 | `PUT /admin/categories/order`、`PATCH /admin/categories/{id}` | ✅ |
| A-06 | 导航项 CRUD | `GET/POST /admin/bookmarks`、`GET/PATCH/DELETE /admin/bookmarks/{id}` | ✅ |
| A-07 | 排序/移动 | `PUT /admin/bookmarks/order`、`POST /admin/bookmarks/{id}/move` | ✅ |
| A-08 | 图标上传/抓取 | `POST /admin/uploads/icon`、`POST /admin/icons/fetch` | ✅ |
| A-09 | 站点设置 | `GET/PUT /admin/settings` | ✅ |
| A-11 | 更新信息 | `GET /admin/update-check` | ✅ |
| D-02 | 导出 | `GET /admin/export` | ✅ |
| D-03 | 导入（预览/执行） | `POST /admin/import/preview`、`POST /admin/import` | ✅ |
| D-04 | 旧站导入 | `ExportBundle.format = legacy-v1` | ✅（预留） |
| D-05 | 备份/恢复 | `GET/POST /admin/backups`、`POST /admin/backups/{id}/restore` | ✅ |
| D-06 | 版本信息 | `GET /version` | ✅ |

**结论：无功能缺口。**

---

## 3. 本次修订内容

### P1 修订

| 修订 | 原因 |
| --- | --- |
| 单资源更新由 `PUT` 改为 `PATCH` | 明确"部分字段更新"语义，避免 PUT 全量覆盖歧义 |
| 每个操作补充 `default` 错误响应（`ErrorResponse`） | 保证所有接口都有统一错误体，便于前端统一处理 |
| 增加 `X-Requested-With` 请求头参数 | 明确 CSRF 附加信号 |
| 新增 `CategoryList` / `BookmarkList` / `SearchEngine` 等具名 Schema | 减少内联重复，便于类型生成 |
| `ImportRequest` 独立为 Schema | 表达 `conflict_strategy` 默认 `skip` |
| 增加 `operationId` | 便于 SDK/类型生成与测试命名 |
| 增加 `NavigationResponse`、`HealthResponse`、`UploadResult`、`IconRef` | 覆盖首页聚合、健康检查与图标流程 |
| 明确 `is_visible = false` 即"私有，仅管理员可见" | 落实访客访问策略 |

### P2 修订

| 修订 | 原因 |
| --- | --- |
| 新增 `GET /api/v1/public/uploads/{filename}` | 上传图标（`icon_type=upload`）需要公开读取路径，否则前端无法显示本地图标 |

---

## 4. 接口分级与安全边界

| 级别 | 前缀 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| 公开 | `/api/v1/public/*` | 匿名 | 仅返回 `is_visible = true` 数据 |
| 认证 | `/api/v1/auth/*` | 部分匿名 | 登录/初始化匿名；其余需会话 |
| 管理 | `/api/v1/admin/*` | 会话必需 | 全部读写操作 |
| 系统 | `/healthz`、`/api/v1/version` | 匿名 | 健康检查与版本 |

- 全局默认 `security: cookieAuth`；公开操作显式 `security: []`。
- 所有写操作要求 `X-Requested-With: XMLHttpRequest`。

---

## 5. 已知待办（不阻塞冻结）

| 项 | 说明 |
| --- | --- |
| 规范级 lint | 尚未接入 `redocly lint`/`spectral`；P2 起点引入到 CI |
| `legacy-v1` 字段映射 | 待旧站真实数据；导入器先按宽松映射实现 |
| 分页上限调优 | `page_size` 上限 100，视实际数据量再定 |
| 限流参数 | 登录限流阈值待 P2 实测后确定 |
| 备份 `id` 格式 | 暂定为快照文件名（含时间戳），实现时确认 |

---

## 6. 契约维护规则（复述）

1. 接口变更**先改契约**，再改后端实现与前端类型。
2. 后端实现完成后运行契约一致性测试（路由/字段与 `openapi.yaml` 对齐）。
3. 破坏性变更升级 API 版本并记录 ADR。
4. 每次契约变更同步更新本文件与 `docs/api-spec.md`。
