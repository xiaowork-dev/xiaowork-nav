# ADR-0006：服务端自动获取并缓存网站图标

- 状态：已接受
- 日期：2026-10-25
- 关联：[V2.1.0 实施计划](../development-plan-v2.1.md)、[更新指南](../updating.md)、[数据库设计](../database-design.md)

## 背景

导航站中大量网站没有手动指定图标，只能长期显示首字母占位。V2.1.0 需要"自动获取
网站官方图标、缓存并管理"，同时必须遵守两条硬约束：

1. **绝不覆盖用户手动设置的图标**（上传 / 外链 / Simple Icons / 文字图标）；
2. 抓取属于服务端访问外部 URL，必须防 SSRF，且**不得宣称来源不明的图标是官方图标**。

现有实现已具备可复用的基础：`IconService.Fetch` 能解析 `<link rel=icon>` 并回退
`/favicon.ico`，`security.SafeHTTPClient` 已在 dialer 层校验**解析后 IP**（可防 DNS
重绑定）。缺口在于：无 Web App Manifest 解析、无 apple-touch 优先、无图片质量判定、
无按来源去重缓存、无来源/失败原因记录、无异步批量任务，且 `icon_type=auto` 虽被保留
却完全无效。

## 决策

### 1. `icon_type=auto` 承载"自动获取的图标"

`auto` 允许存储一个站内上传路径（`/api/v1/public/uploads/...`）；为空表示"尚未获取"。
这样前端 `IconImage` 无需改动即可显示自动图标，而 `none`/`favicon` 仍不携带值。

### 2. 手动类型永不被自动流程触碰

`manualIconTypes = {url, upload, simple, text}`。批量补全的候选查询在 SQL 层就排除
这四类（`ListIconFillCandidates`），单个网站的自动获取也会返回 `skipped=true`。

### 3. 候选来源与排序

按"声明尺寸 → 格式 → URL"排序，真实声明永远优先于站点 favicon，favicon 优先于品牌推测：

| 来源 | `icon_source` | 说明 |
| --- | --- | --- |
| `<link rel="icon|shortcut icon|alternate icon">` | `link-icon` | 按 `sizes` 从大到小 |
| `<link rel="apple-touch-icon*">` | `apple-touch` | 通常 180×180 |
| Web App Manifest 的 `icons[]` | `manifest` | 通常 192/512 |
| `scheme://host/favicon.ico` | `favicon-ico` | 固定回退 |
| 域名在 Simple Icons 校验列表中命中 | `brand-guess` | **最后手段，UI 明确标注"（推测）"** |

`mask-icon` / `fluid-icon` 等装饰性 `rel` 会被忽略（按 token 精确匹配，而不是子串匹配）。

### 4. 质量判定

自动获取的图标必须通过质量门：PNG/JPEG/GIF/WebP/ICO 尺寸 ≥ 16px、PNG 不得整图全透明、
SVG 必须含 `<svg` 且不含 HTML 文档标记、单文件 ≤ 256 KiB。质量不合格的候选会被跳过并
继续尝试下一个来源。

### 5. 缓存与去重

`uploads` 表新增 `source_url`（抓取来源）与 `origin`（`manual` / `auto`）。抓取前先按
`source_url` 查缓存，命中且文件仍在就直接复用（不再访问第三方）。`force=true` 强制刷新。

**不做后台 GC**。仅在替换某网站的自动图标时，若旧文件 `origin=auto` 且**无任何网站引用**
才删除；用户上传的文件永不自动删除。这满足"不允许自动清理仍被其他网站引用的共享图标"。

### 6. 异步批量任务

`IconJobManager`：内存态、**同时只允许一个任务**、工作池 3 并发、每个请求间隔 150ms、
单次上限 500 个网站。任务状态不落库（重启即丢），前端通过 `GET /admin/icons/bulk` 恢复显示。

### 7. SSRF 加固

沿用现有能力（dialer 层解析后 IP 校验、CIDR 黑名单含云元数据、重定向逐跳校验），
并新增 `security.ValidateFetchURL`：拒绝 URL 内嵌凭据（`user:pass@host`）。
每个候选独立超时 10s、单次抓取总预算 25s、响应体上限 256 KiB、下载前按 Content-Type
提前放弃 HTML。

**未限制端口**：需求针对的是地址段（回环/内网/链路本地/云元数据），这些已在 dialer 层
按解析后 IP 拦截；限制端口会误伤自建站点且不带来额外安全收益。

### 8. 首页文档体积

真实首页经常超过 512 KiB（github.com ≈ 580 KB、cloudflare.com ≈ 1.3 MB）。图标声明几乎
总在文档头部，因此发现阶段采用"读取前 512 KiB 后截断"而不是整体失败。

## 理由

- **复用**：`auto` 类型、`IconService`、`security` 与 `IconImage` 回退链都已存在，改动集中且向后兼容。
- **安全**：所有出站请求都经过同一套 dialer 校验；质量门避免把空白/极小图片写入缓存。
- **诚实**：来源被记录并展示；推测来源显式标注，不宣称官方。
- **可回滚**：迁移只增列并带默认值，旧镜像仍可读（不删列、不改列）。

## 影响

- 数据库结构版本 2 → **3**；`schema_version` 变化，降级需按 `docs/updating.md` 恢复快照。
- 新增 5 个管理接口（详见 `docs/openapi/openapi.yaml`），全部需要管理员鉴权。
- 自动获取会在 `/data/uploads` 产生缓存文件；备份与恢复已覆盖该目录。
- `settings.homepage_layout` 的旧值 `grid`/`list` 在读取时归一化为 `scroll`/`tabs`（不重写历史数据）。

## 备选方案

| 方案 | 未采用的原因 |
| --- | --- |
| 用搜索引擎/爬虫找"官方 Logo" | 需求明确不要求，且无法保证来源真实性 |
| 每次都实时请求第三方图标服务 | 慢、易被限流，且违反"缓存并随 Docker 更新保留" |
| 后台定时任务清理孤儿图标 | 误删共享图标风险高；改为只在替换且无引用时清理 |
| 限制只允许 80/443 端口 | 会误伤自建站点，而地址段限制已由 dialer 覆盖 |
| 用 `golang.org/x/image` 解析 WebP | 为单一格式引入新依赖；改为解析 VP8X/VP8L/VP8 头部 |
| 任务状态落库 | 需要额外的任务表与恢复逻辑，收益有限（管理员可重新发起） |
