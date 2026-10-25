# XiaoWork Nav V2.1.0 开发实施计划

> 版本：V2.1.0
> 基线：v2.0.1（已发布，生产环境已通过 Docker + 1Panel OpenResty HTTPS 部署）
> 本文是 V2.1.0 的**实施计划**；阶段交付报告位于 [`deliveries/v2.1.0/`](deliveries/v2.1.0/)。
> 关联：[架构说明](architecture.md)、[API 规范](api-spec.md)、[数据库设计](database-design.md)、[更新指南](updating.md)、[进度总览](PROGRESS.md)

---

## 1. 目标与范围

V2.1.0 为**增量升级**，包含三个核心功能：

| # | 功能 | 目标 |
| --- | --- | --- |
| 1 | 网站官方图标自动获取、缓存与管理 | 减少"首字母占位"，把图标缓存到 `/data/uploads` 随 Docker 更新保留 |
| 2 | PC 端滚动布局左侧分类导航栏 | 网站较多时可快速定位分类 |
| 3 | 标签页式导航布局 + 两种布局一键切换 | 用户可自主选择浏览方式 |

**不做的事**（避免过度设计）：

- 不重构已稳定的后端业务模块（分类 / 网站 CRUD、导入导出、备份、鉴权保持不动）。
- 不引入前端 UI 框架、不改技术栈。
- 不引入第三方图片资源，默认背景仍为现有渐变。
- 不为"官方图标"引入搜索引擎爬虫，也不把推测来源的图标宣称为官方图标。

---

## 2. 现状分析（V2.1-P1 调研结论）

### 2.1 图标系统现状

| 层 | 现状 |
| --- | --- |
| 数据模型 | `bookmarks.icon_type` / `icon_value`；`icon_type` 取值 `none \| url \| upload \| auto \| favicon \| simple \| text`。`auto` 已被保留但**当前完全无效**（校验时强制清空 value） |
| 服务层 | `service.IconService`：`Fetch()` 已实现"抓取站点图标 → 落盘 → 返回 upload 引用"，含 `discoverIcon`（正则解析 `<link rel=icon>`）与 `/favicon.ico` 回退 |
| 安全 | `security.SafeHTTPClient` 已在 **dialer 层**校验解析后 IP（可防 DNS 重绑定），带 CIDR 黑名单（含 `169.254.169.254`）、重定向跳数与 scheme 校验 |
| 存储 | `uploads` 表（`filename/original_name/mime/size/created_at`）+ `/data/uploads` 目录，由 `/api/v1/public/uploads/{filename}` 提供，带 `immutable` 缓存头与 SVG 沙箱 CSP |
| 缺失 | 无 Web App Manifest 解析、无 apple-touch-icon 优先、无图片质量判定、无按来源去重/缓存复用、无来源与失败原因记录、**无任何异步任务基础设施**、无 upload 引用计数、`fetch` 接口同步且不落库 |
| 迁移 | `//go:embed sql/*.sql`，命名 `NNNN_name.up.sql`，当前 **max = 0002**，下一个为 `0003` |
| 契约 | `api/routes.go` 为唯一路由表，`contract_test.go` 强制与 `docs/openapi/openapi.yaml` 一致 |

### 2.2 前端布局现状

| 层 | 现状 |
| --- | --- |
| 页面 | `pages/HomePage.vue` 单文件：`AppHeader` + 时钟 + `SearchBar` + 分类列表（`CategorySection`）或搜索结果 |
| 数据 | `stores/site.ts` 一次 `GET /api/v1/public/navigation` 拿到 `categories[]`（含嵌套 `bookmarks[]`）与 `settings` |
| 样式 | `styles/main.css` 纯 CSS 变量主题（`data-theme`），磁贴网格 `.tiles`，`--page-width: 1200px`；无 UI 框架 |
| 主题 | `composables/useTheme.ts`：模块级单例 + `localStorage['xn-theme']` + `document.documentElement.dataset.theme` |
| 布局偏好 | `PublicSettings.homepage_layout` 类型为 `'grid' \| 'list'`，**首页从未读取**；前端无任何布局偏好持久化 |
| 组件 | `CategorySection` / `BookmarkCard` / `IconImage`（显示回退链：用户图标 → `origin/favicon.ico` → 首字母） |
| 测试 | **前端无测试运行器**；门禁为 `vue-tsc --noEmit` + `vite build` |

### 2.3 结论

- 功能 1 可**复用** `IconService` 与 `security` 现有能力，增量补齐"解析来源 / 质量判定 / 缓存复用 / 异步批量 / 来源与失败记录"。
- 功能 2、3 属于**纯前端新增**，数据层完全复用 `stores/site.ts`，不新增导航数据 API。
- 唯一需要的数据库迁移是功能 1 的图标元数据列（`0003`）。

---

## 3. 功能一设计：官方图标自动获取与缓存

### 3.1 图标类型与优先级（严格遵守）

| 优先级 | 来源 | 是否被自动流程覆盖 |
| --- | --- | --- |
| 1 | 用户上传（`upload`） | **否** |
| 2 | 用户指定的外部 URL（`url`） | **否** |
| 3 | 用户选择的 Simple Icons（`simple`）/ 文字图标（`text`） | **否** |
| 4 | 自动获取（`auto`） | 是（自动流程的目标） |
| 5 | 用户选择的"站点图标"（`favicon`） | 仅**缓存化**（自动流程可把 `origin/favicon.ico` 落盘，显示结果不变） |
| 6 | 无图标（`none`） | 是（自动流程的目标） |
| 7 | 首字母占位 | 前端显示回退，永不写库 |

判定规则（`manualIconTypes = {url, upload, simple, text}`）：**批量补全永远跳过 manual 类型**。

### 3.2 自动获取的候选来源与顺序

1. HTML 中的 `<link rel="icon|shortcut icon|apple-touch-icon|apple-touch-icon-precomposed">`，
   按 `sizes` 从大到小、格式优先级 `svg > png > webp > ico` 排序。
2. HTML 中的 `<link rel="manifest">` → 拉取 manifest JSON → `icons[]`（按 `sizes` 从大到小）。
3. `scheme://host/favicon.ico`。
4. **品牌图标库（推测）**：仅当上面全部失败时，用域名可注册部分（如 `github.com` → `github`）
   在**内置的 Simple Icons 校验列表**中匹配；命中才使用，并标记来源为 `brand-guess`（UI 显示"品牌图标（推测）"）。
   *不宣称官方图标*。

> 刻意不实现"搜索引擎爬虫抓官方 Logo"，符合需求 2.3 的免责说明。

### 3.3 质量判定（拒绝低质量 / 透明空白）

`imageQuality(data, mime)`：

- `png` / `jpeg` / `gif`：`image.DecodeConfig` 取尺寸，宽或高 **< 16px** 判为低质量；
- `png` 额外检测**整图全透明**（解码后扫描 alpha，像素数超阈值则跳过检测）；
- `webp`：解析 `VP8X` / `VP8L` / `VP8 ` 头取尺寸（不引入 `golang.org/x/image` 依赖）；
- `ico`：解析目录，取最大条目的宽高；
- `svg`：必须含 `<svg` 根元素，且体积 ≥ 64 字节；
- 自动获取的图标**单文件上限 256 KiB**（用户上传仍为 1 MiB）。

### 3.4 缓存与去重

- 新增 `uploads.source_url`（抓取来源 URL，手动上传为空）与 `uploads.origin`（`manual` | `auto`）。
- 抓取前先按 `source_url` 查已有缓存：命中且文件仍在 → **直接复用**（不发起网络请求）。
  → 满足"避免每次访问首页都重新请求第三方图标服务"。
- `force=true`（"重新获取"）跳过复用。
- **不实现自动 GC**；仅在替换某个网站图标时，若旧图标是 `origin=auto` 且**无任何其他网站引用**
  （按 `icon_value` 计数）才删除文件，满足"不允许自动清理仍被其他网站引用的共享图标"。
- 更新策略：`bookmarks.icon_checked_at` 记录最近一次自动抓取时间；后台批量补全默认跳过
  `icon_checked_at` 在 TTL（默认 30 天）内且状态为 `ok` 的网站。

### 3.5 数据库迁移 `0003_icon_metadata`

只增列，全部带默认值（旧数据可读、可回滚）：

```sql
ALTER TABLE bookmarks ADD COLUMN icon_source     TEXT NOT NULL DEFAULT '';
ALTER TABLE bookmarks ADD COLUMN icon_status     TEXT NOT NULL DEFAULT '';   -- ok | failed | ''
ALTER TABLE bookmarks ADD COLUMN icon_error      TEXT NOT NULL DEFAULT '';   -- 最近失败原因（截断）
ALTER TABLE bookmarks ADD COLUMN icon_checked_at TEXT NOT NULL DEFAULT '';   -- 最近自动抓取时间
ALTER TABLE uploads   ADD COLUMN source_url      TEXT NOT NULL DEFAULT '';
ALTER TABLE uploads   ADD COLUMN origin          TEXT NOT NULL DEFAULT 'manual';
CREATE INDEX IF NOT EXISTS idx_uploads_source_url ON uploads(source_url);
```

`icon_source` 取值：`manual` | `link-icon` | `apple-touch` | `manifest` | `favicon-ico` | `brand-guess`。

### 3.6 API（新增 / 变更，需同步 OpenAPI）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/admin/bookmarks/{id}/icon/auto` | 为单个网站抓取并落库；body `{force?:bool}`；返回更新后的 bookmark + `icon_source/icon_status/icon_error` |
| POST | `/api/v1/admin/icons/bulk` | 启动批量补全（**同时只允许一个任务**）；body `{only_missing?:bool, force?:bool, limit?:int}`；返回任务快照 |
| GET | `/api/v1/admin/icons/bulk` | 当前 / 最近一次任务快照（含进度与失败明细），无任务返回 `null` |
| POST | `/api/v1/admin/icons/bulk/cancel` | 取消正在运行的任务 |
| GET | `/api/v1/admin/icons/summary` | 图标统计：总数 / 手动 / 自动 / 缺失 / 失败 |

变更：`POST /api/v1/admin/icons/fetch` 保持兼容（仍返回 `{icon_type, icon_value}`），
新增可选 `bookmark_id` 用于直接落库。

### 3.7 异步任务

- 新增 `service.IconJobManager`：内存态、单任务、工作池 **3** 并发、请求间 `150ms` 间隔（限频）。
- 任务状态：`running | done | canceled | failed`，含 `total/done/succeeded/failed/skipped/started_at/finished_at/errors[]`（最多 50 条明细）。
- 单任务上限默认 500 个网站（`limit` 可调），避免长时间占用。
- 任务状态不落库（重启即丢失），但前端可通过 `GET .../bulk` 恢复显示；文档明确此取舍。

### 3.8 安全加固（在现有 SSRF 防护之上）

- 新增 `security.ValidateFetchURL`：拒绝 userinfo（`user:pass@`）、仅允许 **80/443** 端口、
  必须是绝对 http(s) URL；保留 DNS 解析后 IP 校验（dialer 层，防 DNS 重绑定）与重定向逐跳校验。
- 请求头 `Accept: image/*,image/webp,image/svg+xml;q=0.9,*/*;q=0.1`；
  响应 `Content-Type` 为 `text/html` 时立即放弃该候选。
- 每个候选独立超时 10s，单次抓取（含所有候选）总预算 25s。
- 响应体上限：自动抓取 256 KiB（`io.LimitReader` 强制）。
- 落盘前一律 `detectImageType` 内容嗅探（不信任扩展名与 Content-Type），SVG 必须含 `<svg`；
  继续沿用现有 SVG 沙箱 CSP 与 `nosniff`。

---

## 4. 功能二设计：PC 端固定分类侧边栏

- 新增 `components/CategorySidebar.vue`：分类图标/emoji、名称、网站数量、当前高亮、可折叠、独立滚动。
- `ScrollLayout.vue` 使用 `display: grid; grid-template-columns: 236px 1fr`，侧栏 `position: sticky; top: 84px`，
  与 `.admin__nav` 现有模式一致。
- 定位：点击侧栏 → `scrollIntoView`/`scrollTo` 平滑滚动，分类锚点用 `scroll-margin-top` 抵消顶部固定区高度。
- 高亮：`IntersectionObserver`（rootMargin 顶部负偏移）为主 + `scroll` 兜底（节流），
  当前项通过 `scrollIntoView({block:'nearest'})` 保持在侧栏可视范围内。
- 响应式：
  - `≥1080px`：完整侧栏；
  - `768–1079px`：默认折叠为图标条（可展开为浮层）；
  - `<768px`：隐藏侧栏，改为顶部**吸顶横向分类胶囊条**（可横向滚动）用于快速跳转；
  - 任何断点都不产生横向滚动。
- 标签模式**不显示**侧栏。

## 5. 功能三设计：双布局与一键切换

| 模式 | 值 | 中文 | 说明 |
| --- | --- | --- | --- |
| 滚动模式 | `scroll` | 滚动模式 | 现有布局增强版 + 左侧分类栏 |
| 标签模式 | `tabs` | 标签模式 | 顶部横向分类标签 + 居中搜索 + 网格磁贴 |

- 新增 `composables/useLayout.ts`：模块级单例 + `localStorage['xn-layout']`，
  与 `useTheme` 完全同构；本地偏好优先于站点默认值；默认 `scroll`。
- 新增 `components/LayoutSwitcher.vue`：分段控件（与主题按钮同风格），放在 `AppHeader` 右侧。
- 站点默认值：`settings.homepage_layout` 取值规范化为 `scroll | tabs`
  （旧值 `grid → scroll`、`list → tabs` 在读取时归一化，保持向后兼容）。
- 主题与布局**完全独立**：切换布局不触碰 `xn-theme` 与 `data-theme`。
- 标签模式提供「首页」标签（无收藏机制时的合理默认 = 展示全部可见网站），
  个性化常用网站留待后续版本（文档标注）。
- 两种布局共享 `stores/site.ts` 的同一份数据，搜索逻辑抽到 `composables/useNavSearch.ts`，
  磁贴网格抽到 `components/BookmarkGrid.vue`。

### 5.1 新增/调整组件清单

```text
frontend/src/components/
  CategorySidebar.vue     (新) PC 左侧分类栏
  LayoutSwitcher.vue      (新) 布局切换控件
  BookmarkGrid.vue        (新) 磁贴网格（两种布局共用）
  ScrollLayout.vue        (新) 滚动模式
  TabsLayout.vue          (新) 标签模式
  CategorySection.vue     (改) 增加锚点 id / scroll-margin
  AppHeader.vue           (改) 右侧加入 LayoutSwitcher（可选开关）
frontend/src/composables/
  useLayout.ts            (新)
  useNavSearch.ts         (新)
frontend/src/pages/
  HomePage.vue            (改) 仅负责编排：header + 按布局渲染
```

---

## 6. 部署与数据安全

### 6.1 首次部署数据目录权限修复（需求 6.4）

问题：宿主机数据目录由 root（或非 100 的 uid）创建，容器内以 `uid=100` 运行，
`mkdir /data/uploads` 失败。

方案（不使用 777，也不长期以 root 运行）：

1. `deploy/compose.yaml` / `compose.npm.yaml` 增加 `user: "${XN_UID:-100}:${XN_GID:-101}"`；
2. `scripts/install.sh`：创建数据目录后，若能 `chown`（root 或可用的 sudo）则 `chown -R 100:101`；
   否则**自动**把 `XN_UID`/`XN_GID` 写入 `.env` 为当前用户 uid/gid，使容器与宿主机属主一致；
3. 镜像内新增 `deploy/docker-entrypoint.sh`：启动时尝试创建/校验数据目录可写，
   失败则打印**可直接复制执行**的修复命令并非 0 退出（快速失败、信息明确）；
4. `deploy.sh` / `update.sh` 复用同一套目录准备逻辑（`lib.sh` 新增 `ensure_data_dirs`）。

### 6.2 升级与回滚验证（需求 6.4）

在真实 Docker 环境验证：V2.0.1 → V2.1.0 升级后数据库完整、分类/网站数量不变、
手动 SVG 图标不丢失、自动缓存图标持久化、登录/搜索/布局切换正常、更新前自动备份正常、
失败可安全回滚。扩展 `scripts/tests/deploy-faults.test.sh` 或新增升级用例。

---

## 7. 阶段划分与交付物

| 阶段 | 内容 | 交付文档 |
| --- | --- | --- |
| **V2.1-P1** | 调研、架构与计划（本文） | `deliveries/v2.1.0/P1-architecture.md` |
| **V2.1-P2** | 图标自动获取：迁移 0003、服务层、API、异步批量、后台 UI、测试 | `deliveries/v2.1.0/P2-icons.md` |
| **V2.1-P3** | 滚动布局增强：`CategorySidebar`、定位/高亮/折叠/响应式 | `deliveries/v2.1.0/P3-scroll-layout.md` |
| **V2.1-P4** | 标签布局与切换：`TabsLayout`、`LayoutSwitcher`、偏好持久化、联调 | `deliveries/v2.1.0/P4-tabs-layout.md` |
| **V2.1-P5** | 集成测试、真实数据兼容、Docker 权限修复、升级/回滚、文档、发布 v2.1.0 | `deliveries/v2.1.0/P5-release.md` |

每个阶段结束：运行测试 → 更新交付文档 → 更新 `docs/PROGRESS.md` → 提交并推送 GitHub。

---

## 8. 测试策略

### 8.1 后端（Go）

- 图标解析：`<link rel=icon>` 多候选排序、apple-touch 优先、manifest 解析、favicon.ico 回退、品牌推测回退。
- 优先级：manual 类型永不被批量覆盖。
- 失败处理：超时、404、非图片、HTML 伪装、超大响应、低质量（<16px）、全透明 PNG。
- 缓存：同 `source_url` 复用、`force` 重新抓取、旧图标无引用时才删除、有引用时不删除。
- 批量：进度统计、跳过 manual、单任务互斥、取消。
- 安全：私网/回环/链路本地/云元数据 IP 拦截、userinfo 拒绝、非 80/443 端口拒绝、
  重定向到内网被拒、DNS 重绑定（dialer 层）由现有测试覆盖。
- 迁移：0003 幂等、旧库升级后数据完整、`CheckCompatible` 不变。

### 8.2 前端

- 无测试运行器 → 以 `vue-tsc --noEmit` + `vite build` 为门禁；
- 新增最小 **Vitest** 单元测试（布局偏好持久化、搜索 composable、图标候选链），
  并在 CI 中运行 `npm run test`（若引入成本过高，则改为 Node 脚本断言 + 明确标注未执行）。

> 决策：为满足需求 7.2（"前端测试"），引入 **Vitest**（Vite 官方、零配置、与现有构建同源），
> 仅覆盖纯逻辑（composable / 工具函数），不引入 jsdom 组件快照测试，避免过度设计。

### 8.3 真实浏览器验收（需求 7.3）

使用 chrome-devtools MCP 对**真实运行的前后端**截图，视口 1440 / 834 / 390 / 375，
覆盖 9 类截图（滚动深/浅、标签深/浅、移动端两模式、侧栏展开/折叠、图标前后对比、后台图标管理）。

---

## 9. 风险与对策

| 风险 | 对策 |
| --- | --- |
| 自动抓取误判来源（把非官方图标当官方） | UI 只显示"来源"，品牌推测显式标注"（推测）"；文档免责 |
| 批量抓取拖慢服务器 / 触发对方风控 | 单任务 + 3 并发 + 150ms 间隔 + 单次上限 |
| 上传目录被自动任务撑大 | 按 `source_url` 去重；仅在无引用时清理；不做后台 GC |
| 生产数据被破坏 | 全部开发在本地/隔离环境；不直接操作线上；发布不等于升级 |
| 前端无测试导致回归 | 引入 Vitest 覆盖纯逻辑 + 真实浏览器验收截图 |
| 新迁移影响回滚 | 0003 只增列且带默认值；旧镜像可读（不删列、不改列） |

---

## 10. 版本与发布

- 正式版本 **v2.1.0**，数据库结构版本 **3**。
- 发布前：全量测试 + 真实浏览器验收 + 升级/回滚验证。
- 发布流程：更新版本号/CHANGELOG/README/文档 → 提交推送 → 打标签 `v2.1.0` →
  GitHub Actions 构建 amd64/arm64 → 推送 GHCR → 创建 Release → 匿名拉取并运行验证。
- **生产升级必须等用户确认后执行**，本次开发不触碰 `/opt/xiaowork-nav`。
