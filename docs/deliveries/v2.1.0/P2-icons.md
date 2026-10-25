# V2.1.0 / P2 阶段交付报告：网站官方图标自动获取与缓存

> 本文件是 **V2.1.0 第 P2 阶段的正式交付文档**，可长期归档。
> 内容基于实际代码、真实 Docker/浏览器验证与 Git 状态编写；未验证内容均明确标注。
> 关联：[V2.1.0 实施计划](../../development-plan-v2.1.md)、[ADR-0006](../../decisions/ADR-0006-icon-auto-fetch.md)、[AGENTS.md](../../../AGENTS.md)

---

## 1. 阶段目标

实现"网站官方图标自动获取、缓存与管理"：对**没有手动设置图标**的网站自动检索其官方图标，
缓存到持久化数据目录，并在后台提供自动获取 / 重新获取 / 批量补全 / 来源与失败原因展示。
硬约束：**绝不覆盖用户手动设置的图标**、防 SSRF、不宣称来源不明的图标为官方图标。

## 2. 完成的工作

| 工作 | 结果 |
| --- | --- |
| 数据库迁移 | `0003_icon_metadata`（bookmarks 4 列 + uploads 2 列 + 2 个索引），结构版本 2 → 3 |
| 图标发现与抓取 | 网页声明图标 / Apple Touch / Web App Manifest / favicon.ico / 品牌图标（推测），按质量排序 |
| 图片质量门 | 尺寸 ≥16px、PNG 非全透明、真实 SVG、单文件 ≤256 KiB |
| 缓存与去重 | 按 `source_url` 复用；`origin=manual/auto` 区分；仅"无引用"时清理 auto 文件 |
| 异步批量任务 | 单任务互斥、3 并发、150ms 限频、进度/失败明细、可取消、上限 500 |
| 管理接口 | 5 个新接口 + 1 个既有接口扩展，全部同步 OpenAPI 契约 |
| SSRF 加固 | `ValidateFetchURL`（拒绝 URL 内嵌凭据）+ 单候选超时/总预算/体积上限/Content-Type 提前放弃 |
| 后台 UI | 图标统计、批量补全（进度条 + 失败列表）、单站自动获取/重新获取、来源与失败原因展示 |
| 测试 | 后端 115 个测试全部通过；真实站点冒烟 6/6；浏览器实测批量补全 **19/19 成功** |
| 文档 | ADR-0006、CHANGELOG、api-spec、database-design、PROGRESS、README（V2.1 小节后续补齐） |

## 3. 新增和修改的文件

### 3.1 新增

| 文件 | 作用 |
| --- | --- |
| `backend/internal/migrations/sql/0003_icon_metadata.up.sql` / `.down.sql` | 图标元数据迁移 |
| `backend/internal/service/iconfetch.go` | 图标发现、候选排序、质量判定、WebP/ICO 尺寸解析 |
| `backend/internal/service/iconjob.go` | 单站填充服务与异步批量任务管理器 |
| `backend/internal/api/admin_icons.go` | 5 个新接口的处理函数 |
| `backend/internal/service/iconfetch_test.go` | 解析/质量/排序/缓存/失败/安全测试（14 个用例） |
| `backend/internal/service/iconjob_test.go` | 手动图标保护、成功与失败记录、批量任务、清理测试（7 个用例） |
| `backend/internal/service/settingslayout_test.go` | 布局取值归一化与 `auto` 图标取值校验（3 个用例） |
| `backend/internal/store/iconstore_test.go` | 候选查询过滤、统计、引用计数、缓存去重（4 个用例） |
| `backend/internal/migrations/iconmetadata_test.go` | 迁移列与向后兼容默认值（2 个用例） |
| `backend/internal/security/fetchurl_test.go` | `ValidateFetchURL` 规则（2 个用例） |
| `backend/internal/api/iconadmin_test.go` | 新接口鉴权/行为/生命周期（7 个用例） |
| `frontend/src/utils/icons.ts` | 图标类型与来源的中文标签、手动类型判定 |
| `docs/decisions/ADR-0006-icon-auto-fetch.md` | 决策记录 |

### 3.2 修改

| 文件 | 变更 |
| --- | --- |
| `backend/internal/models/models.go` | Bookmark 增加 4 个图标元数据字段；Upload 增加 `source_url`/`origin`；新增来源/状态常量 |
| `backend/internal/store/bookmarks.go` | 读取新列；新增 `SetBookmarkIcon`、`ListIconFillCandidates`、`CountIconSummary`、`CountBookmarksUsingUpload` |
| `backend/internal/store/uploads.go` | 读写 `source_url`/`origin`；新增 `FindUploadBySourceURL` |
| `backend/internal/service/icons.go` | `saveUpload` 支持来源与 origin；`download` 支持截断与 Content-Type 门；`FetchIcon` 管线；`DeleteAutoUploadIfUnreferenced` |
| `backend/internal/service/catalog.go` | `auto` 类型允许站内缓存路径；`none`/`favicon` 行为不变 |
| `backend/internal/service/settings.go` | `homepage_layout` 归一化为 `scroll`/`tabs`（兼容旧值 `grid`/`list`） |
| `backend/internal/security/ssrf.go` | 新增 `ValidateFetchURL`；重定向逐跳改用更严格校验；出站请求设置 `Accept` |
| `backend/internal/api/routes.go`、`server.go` | 注册 5 个新路由；注入 `IconFillService` |
| `backend/internal/api/helpers.go` | 新增 `decodeOptionalJSON` |
| `docs/openapi/openapi.yaml` | 5 个新路径 + `BulkIconRequest`/`IconFillOutcome`/`IconJobSnapshot`/`IconJobItem`/`IconSummary`；Bookmark 新字段与 icon_type 枚举补全 |
| `frontend/src/types/api.ts` | 新增图标元数据、来源/状态、批量任务类型；`HomepageLayout` 改为 `scroll`/`tabs` |
| `frontend/src/api/admin.ts` | 新增 5 个图标相关请求函数 |
| `frontend/src/pages/admin/BookmarksPage.vue` | 图标统计栏、批量补全（进度/取消/失败明细）、单站自动获取/重新获取、来源与失败原因 |
| `frontend/src/components/BookmarkFormModal.vue` | 图标类型新增"自动获取"；`auto` 不再清空已缓存路径 |
| `frontend/src/pages/admin/SettingsPage.vue` | 默认导航布局改为 `scroll`/`tabs` |
| `frontend/src/styles/main.css` | 图标工具条与进度条样式 |
| `docs/api-spec.md`、`docs/database-design.md`、`CHANGELOG.md`、`docs/PROGRESS.md` | 接口清单、表结构、变更日志、进度同步 |

## 4. 实现的具体功能

### 4.1 图标来源与优先级（严格实现需求 2.2 / 2.3）

| 优先级 | 来源 | 是否被自动流程覆盖 |
| --- | --- | --- |
| 1–3 | `url` / `upload` / `simple` / `text`（用户手动） | **否**（SQL 层即排除，接口层返回 `skipped`） |
| 4 | `auto`（自动获取） | 是（自动流程的目标） |
| 5 | `favicon`（用户选择的站点图标策略） | 可被自动流程缓存化，显示结果不变 |
| 6 | `none`（无图标） | 是 |
| 7 | 首字母占位 | 仅前端显示回退，永不写库 |

候选顺序（按声明尺寸 → 格式 → URL 排序）：
Web App Manifest / Apple Touch / 网页声明图标（`rel` 按 token 精确匹配，忽略 `mask-icon`、`fluid-icon`）
→ 站点 `favicon.ico` → **品牌图标（推测，UI 明确标注）**。

> 刻意**不实现**搜索引擎爬虫抓"官方 Logo"，符合需求"不要求…不宣称官方图标"的免责说明。

### 4.2 缓存与更新策略（需求 2.4）

- 缓存写入 `/data/uploads`，随 Docker 更新保留（备份/恢复已覆盖该目录）。
- 按 `source_url` 去重：同一来源只下载一次，后续直接复用本地文件（不请求第三方）。
- `icon_checked_at` 记录最近抓取时间；批量补全默认跳过 30 天内成功的网站。
- `重新获取`（`force=true`）强制刷新。
- 仅在替换自动图标且**无任何网站引用**时删除旧文件；`origin=manual` 的文件永不自动删除。

### 4.3 后台图标管理（需求 2.5）

- 图标统计：站点总数 / 手动 / 自动 / 站点图标 / 缺失 / 上次失败。
- 单站：`自动获取`（无图标时）与 `重新获取`（已有图标时）；手动图标网站按钮禁用并给出说明。
- 批量补全：异步任务、进度条、可取消、完成后提示成功/失败/跳过数量，并列出失败原因（最多 5 条）。
- 列表展示 `图标类型（来源：xxx）`，失败时直接显示原因。

### 4.4 安全（需求 2.6）

| 措施 | 实现 |
| --- | --- |
| 防 SSRF | dialer `Control` 钩子校验**解析后 IP**（同时防 DNS 重绑定） |
| 禁内网/回环/链路本地/云元数据 | 现有 CIDR 黑名单 + `net.IP` 判定（含 `169.254.169.254`） |
| 检查重定向目标 | 每跳重新校验（scheme + 凭据） |
| 拒绝 URL 内嵌凭据 | 新增 `security.ValidateFetchURL` |
| 体积与超时 | 单候选 10s、单次抓取总预算 25s、响应体 256 KiB（HTML 发现阶段 512 KiB 截断） |
| 校验实际格式 | 内容嗅探（不信任扩展名/Content-Type）+ 真实 SVG 判定 + WebP/ICO 头部解析 |
| SVG 安全 | 必须含 `<svg` 且不含 HTML 文档标记；存储与下发沿用沙箱 CSP + `nosniff` |
| 不执行第三方脚本 | 仅下载图片字节，从不解析执行 HTML/JS |
| 并发与频率 | 单任务 + 3 并发 + 每请求 150ms 间隔 + 单次上限 500 |

## 5. 自动化测试结果

环境：Windows 11 + Go（本机）；后端测试通过"先编译再执行"方式运行（本机安全软件会拦截
`go test` 对新生成可执行文件的立即执行，属本地环境限制，CI 在 Linux 上正常执行 `go test ./...`）。

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 后端构建 | `go build ./...` | ✅ 通过 |
| 后端静态检查 | `go vet ./...` | ✅ 通过 |
| 后端测试 | `go test -count=1 ./...` | ✅ **115 通过 / 0 失败**（P1 基线 75 → 新增 40） |
| 前端类型检查 + 构建 | `npm run build` | ✅ 通过 |
| 脚本语法 | `bash -n scripts/*.sh scripts/tests/*.sh` | ✅ 通过 |
| OpenAPI 语法 | `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过 |
| 契约一致性 | `TestRoutesMatchOpenAPIContract` | ✅ 43 个路由与契约完全一致 |
| Compose 配置 | 独立模式 / NPM 模式 | ✅ 通过 |

新增测试覆盖（按需求 7.1）：

| 需求测试点 | 用例 |
| --- | --- |
| 官方图标解析 | `TestParseLinkTags`、`TestParseLinkTagsIgnoresDecorativeLinks`、`TestParseSizes` |
| 不同图标来源优先级 | `TestRankCandidatesOrdersSourcesByQuality`、`TestFetchIconPrefersLargerDeclaredIconAndCaches`、`TestFetchIconSkipsLowQualityAndFallsBackToFavicon` |
| 图标抓取失败 | `TestFetchIconFailsWhenNothingUsableExists`、`TestFillBookmarkRecordsFailureAndKeepsPreviousIcon` |
| 手动图标保护 | `TestFillBookmarkNeverOverwritesManualIcons`、`TestListIconFillCandidatesExcludesManualIcons`、`TestAutoIconSkipsManualBookmarks` |
| 缓存与失效处理 | `TestFetchIconPrefersLargerDeclaredIconAndCaches`（复用 + force 刷新）、`TestFindUploadBySourceURLDeduplicatesCache`、`TestDeleteAutoUploadIfUnreferenced` |
| 批量获取 | `TestBulkJobFillsEligibleBookmarksOnly`、`TestBulkJobRejectsConcurrentRuns`、`TestBulkJobWithNoCandidatesCompletesImmediately`、`TestBulkIconJobLifecycle` |
| SSRF 与重定向安全 | `TestValidateFetchURL*`、`TestFetchIconRejectsUnsafeURLs`、`TestFetchIconIgnoresUnsafeIconReferences`、既有 `security` 黑名单测试 |
| 数据库迁移兼容性 | `TestIconMetadataMigration`、`TestIconMetadataDefaultsAreBackwardsCompatible` |

### 5.1 真实站点冒烟（非单元测试）

对真实网站执行单站自动获取，结果（`XN_DATA_DIR` 为临时目录）：

| 站点 | 来源 | 结果 |
| --- | --- | --- |
| GitHub | 应用清单图标 | ✅ |
| Cloudflare | 网页声明图标 | ✅ |
| Vite | 网页声明图标（SVG） | ✅ |
| Go | Apple Touch 图标 | ✅ |
| MDN | 应用清单图标 | ✅ |
| 不存在域名 | — | ✅ 明确失败原因（DNS 解析失败），且不写入任何文件 |

缓存复用实测：连续两次调用上传目录文件数不变（5 → 5），未重复下载。

## 6. 浏览器验证结果

真实前后端（Go API `127.0.0.1:8090` + Vite 开发服务器 `127.0.0.1:5199`，经代理联调），
Chrome DevTools 实测：

| 项 | 结果 |
| --- | --- |
| 后台图标统计栏 | ✅ 显示 `站点 20 / 手动 1 / 自动 0 / 站点图标 0 / 缺失 19` |
| 批量补全（真实网站） | ✅ 提示"已开始批量补全（共 19 个网站）"，进度 `0/19`，按钮禁用并出现"取消" |
| 批量补全完成 | ✅ **成功 19 / 失败 0 / 跳过 0**；统计更新为 `自动 19 / 缺失 0` |
| 来源展示 | ✅ 逐条显示"（来源：应用清单图标 / Apple Touch 图标 / 网页声明图标 / 站点 favicon.ico / 品牌图标（推测））" |
| 手动图标保护 | ✅ 文字图标网站显示禁用的"重新获取"，提示"该网站已手动设置图标，自动获取不会覆盖"，图标未被改动 |
| 首页图标效果 | ✅ 19 个站点全部由首字母占位变为真实图标（含 GitHub 猫头、Go 吉祥物、Cloudflare、Figma 等）；文字图标 `1P` 保持不变 |
| 控制台错误 | ✅ 无 JS 错误（仅有 Vite 连接日志与浏览器提示的表单 `id` 建议） |

截图（真实运行页面，非概念图）：

| 文件 | 内容 |
| --- | --- |
| `p2-admin-icons-before.png` | 后台图标管理（补全前：19 个缺失） |
| `p2-admin-icons-running.png` | 批量补全进行中（进度与取消按钮） |
| `p2-admin-icons-after.png` | 后台图标管理（补全后：自动 19、缺失 0、来源可见） |
| `p2-home-after-icons.png` | 首页图标效果（全部真实图标） |

> 截图保存在本机 `%TEMP%\xn-browser\`，**未纳入版本库**（含临时测试数据）。
> 完整的多视口截图集在 P5 统一产出。

## 7. 未执行或未通过的测试

| 项 | 原因 |
| --- | --- |
| 前端单元测试（Vitest） | 计划在 P4 随布局逻辑一并引入（本阶段前端改动以类型检查 + 构建 + 浏览器实测验证） |
| 批量补全在 100+ 网站规模下的表现 | 计划在 P5 用真实数据集（110 个站点）验证 |
| 真实 Docker 容器内的图标抓取 | 计划在 P5 与升级/回滚验证一并执行 |
| 长时间运行与并发压力 | 未做 |
| 非 Windows 环境的本地测试脚本 | 本机为 Windows；CI 在 Linux 运行标准 `go test ./...` |

## 8. 当前项目进度

| 阶段 | 状态 |
| --- | --- |
| v2.0.0 / v2.0.1 发布 | ✅ 已完成 |
| V2.1.0 / P1 调研与架构 | ✅ 已完成 |
| **V2.1.0 / P2 官方图标自动获取** | ✅ **本阶段完成** |
| V2.1.0 / P3 滚动布局左侧分类栏 | ⏳ 待开始 |
| V2.1.0 / P4 标签布局与一键切换 | ⏳ 待开始 |
| V2.1.0 / P5 集成测试与发布 | ⏳ 待开始 |

## 9. Git Commit Hash 与推送状态

| 项 | 内容 |
| --- | --- |
| 本阶段起点 | `b4a1c0f`（`docs(v2.1): plan the V2.1.0 icon and layout work`） |
| 本阶段提交 | 见聊天总结与 `git log`（本文件不硬写自身提交 SHA） |
| 推送 | 本阶段代码、测试与文档一并推送 `origin/main` |
| 标签 | 未创建新标签（v2.1.0 在 P5 发布） |

## 10. 遗留问题

| 问题 | 影响 | 计划 |
| --- | --- | --- |
| 品牌图标为"推测"来源 | 域名与 Simple Icons slug 巧合时可能显示不相关品牌 | 已显式标注"（推测）"；如需可后续改为默认关闭 |
| 自动缓存不做后台 GC | 长期可能积累孤儿文件 | 已按引用计数在替换时清理；如需可后续增加"清理未被引用的自动图标"按钮 |
| 批量任务状态不落库 | 重启后进度丢失 | 明确取舍（ADR-0006）；可重新发起 |
| 部分站点 favicon 体积较大（如 MDN） | 缓存文件偏大 | 上限 256 KiB 已约束；后续可考虑缩放 |
| `auto` 图标在导入/导出中的往返 | 导入时会保留站内路径，但目标实例可能没有该文件 | 与既有 `upload` 行为一致；后续可在导入时校验文件存在性 |

## 11. 下一阶段工作（V2.1-P3）

1. 新增 `components/CategorySidebar.vue`：分类图标/emoji、名称、网站数量、当前高亮、折叠、独立滚动。
2. `ScrollLayout.vue` 采用两列网格 + `position: sticky` 侧栏；分类锚点与 `scroll-margin-top` 抵消顶部固定区。
3. 点击侧栏平滑滚动定位；滚动时用 `IntersectionObserver`（+ `scroll` 兜底）自动高亮，并保持高亮项可见。
4. 响应式：≥1080px 完整侧栏；768–1079px 可折叠；<768px 隐藏侧栏并改用吸顶横向分类胶囊条；无横向滚动。
5. 抽取 `BookmarkGrid.vue` 复用磁贴网格，为 P4 的标签布局做准备。
6. 真实浏览器验证（1440 / 834 / 390 / 375）并截图；阶段交付文档 `P3-scroll-layout.md`；提交并推送。
