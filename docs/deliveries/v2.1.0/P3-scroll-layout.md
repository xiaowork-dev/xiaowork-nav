# V2.1.0 / P3 阶段交付报告：PC 端滚动布局左侧分类栏

> 本文件是 **V2.1.0 第 P3 阶段的正式交付文档**，可长期归档。
> 内容基于实际代码与真实浏览器验证编写；未验证内容均明确标注。
> 关联：[V2.1.0 实施计划](../../development-plan-v2.1.md)、[P2 交付报告](P2-icons.md)、[AGENTS.md](../../../AGENTS.md)

---

## 1. 阶段目标

为滚动导航模式增加 **PC 端固定分类侧边栏**：点击分类平滑定位、滚动时自动高亮当前分类、
侧栏固定不随页面滚走、分类多时可独立滚动、支持折叠与响应式适配，且不遮挡网站图标、
搜索框与管理入口。

## 2. 完成的工作

| 工作 | 结果 |
| --- | --- |
| 侧栏组件 | 新增 `CategorySidebar.vue`：分类图标/emoji、名称、网站数量、当前高亮、折叠、独立滚动 |
| 两列布局 | `ScrollLayout.vue` 采用 `grid-template-columns: 236px minmax(0,1fr)`，侧栏 `position: sticky` |
| 点击定位 | 平滑滚动到分类锚点，`clampScrollTop` 保证不越界 |
| 滚动高亮 | rAF 节流的滚动测量 + 纯函数 `pickActiveIndex`（含"滚到底部时最后一个分类生效"） |
| 高亮可见性 | 侧栏内部 `scrollIntoView({ block: 'nearest' })` 保持当前项在可视范围内 |
| 折叠 | 侧栏折叠为 64px 图标条，状态持久化到 `localStorage['xn-sidebar-collapsed']`，网格列同步收窄 |
| 响应式 | ≥1081px 完整侧栏；≤1080px 自动变图标条；≤700px 隐藏侧栏并改用吸顶横向分类胶囊条 |
| 无横向滚动 | 三个断点实测 `scrollWidth <= innerWidth` |
| 组件复用 | 抽出 `BookmarkGrid.vue`（磁贴网格）与 `useNavSearch.ts`（搜索逻辑），供 P4 标签布局共用 |
| 前端测试 | 引入 Vitest，新增 16 个纯逻辑单元测试（含滚动高亮规则与手动图标保护规则） |
| CI | 前端任务新增 `npm run test` |

## 3. 新增和修改的文件

### 3.1 新增

| 文件 | 作用 |
| --- | --- |
| `frontend/src/components/CategorySidebar.vue` | PC 左侧分类栏 |
| `frontend/src/components/ScrollLayout.vue` | 滚动布局（时钟 + 搜索 + 侧栏 + 分类列表 + 移动端胶囊条） |
| `frontend/src/components/BookmarkGrid.vue` | 磁贴网格（两种布局与搜索结果共用） |
| `frontend/src/composables/useNavSearch.ts` | 搜索状态与行为（两种布局共用） |
| `frontend/src/utils/scrollSpy.ts` | 纯函数：`pickActiveIndex` / `isAtBottom` / `clampScrollTop` |
| `frontend/src/utils/scrollSpy.test.ts` | 11 个单元测试 |
| `frontend/src/utils/icons.test.ts` | 5 个单元测试（手动图标保护规则、来源标签） |

### 3.2 修改

| 文件 | 变更 |
| --- | --- |
| `frontend/src/pages/HomePage.vue` | 精简为"加载数据 + 选择布局"的编排组件 |
| `frontend/src/components/CategorySection.vue` | 增加锚点 `id` 与 `data-category-id`，改用 `BookmarkGrid` |
| `frontend/src/styles/main.css` | 侧栏、胶囊条、两列网格、折叠态与三档响应式样式 |
| `frontend/package.json` | 新增 `vitest` 与 `test`/`test:watch` 脚本 |
| `frontend/package-lock.json` | 依赖锁定 |
| `.github/workflows/ci.yml` | 前端任务增加单元测试步骤 |

## 4. 实现的具体功能

### 4.1 布局结构（需求 3.2）

```
.page
├── AppHeader
└── ScrollLayout
    ├── .clock          时钟
    ├── SearchBar       搜索框
    └── .scroll-layout__body   （两列网格）
        ├── CategorySidebar    左侧固定分类栏（sticky）
        └── .scroll-layout__content
            ├── .chipbar            移动端吸顶分类胶囊条（≤700px 显示）
            └── CategorySection ×N  原有分类与磁贴
```

侧栏内容：分类图标/emoji、分类名称、分类内网站数量、当前选中高亮。

### 4.2 分类定位与高亮（需求 3.3）

- 点击侧栏 → `window.scrollTo({ behavior: 'smooth' })`，目标位置 = 分类顶部 − 24px，
  并由 `clampScrollTop` 保证不会滚到文档起始之上。
- 高亮由**滚动测量**驱动（`requestAnimationFrame` 节流）：取每个分类标题相对视口顶部的位置，
  选出"最后一个已越过 120px 阅读线"的分类。
  - 页面滚到底部时强制高亮最后一个分类——否则较短的末位分类永远无法激活。
  - 采用测量而非 `IntersectionObserver` 的原因：同一视口内出现多个分类、以及滚到底部这两种
    情况，观察者的回调顺序不足以稳定判定"当前分类"。
- 高亮项通过侧栏内部的 `scrollIntoView({ block: 'nearest' })` 保持在侧栏可视范围内。
- 分类锚点使用 `scroll-margin-top` + 显式偏移，避免标题被顶部区域遮挡。
- 侧栏 `position: sticky; top: 16px; max-height: calc(100vh - 32px); overflow-y: auto`：
  固定不随页面滚走，分类多时独立滚动。

### 4.3 响应式（需求 3.4）

| 视口 | 行为 |
| --- | --- |
| ≥ 1081px | 完整侧栏（236px，图标 + 名称 + 数量） |
| ≤ 1080px | 侧栏自动收为 64px 图标条（可手动展开/折叠） |
| ≤ 700px | 隐藏侧栏，改用**吸顶横向分类胶囊条**快速跳转（横向滚动，无滚动条） |

实测三个断点均无横向溢出（`documentElement.scrollWidth <= innerWidth`）。

### 4.4 折叠与展开（需求 3.3.6）

折叠按钮位于侧栏顶部；折叠后侧栏宽 64px、网格首列同步收窄，内容区自动扩展（实测 1440px 下
内容列由 858px 变为 1030px）。折叠状态写入 `localStorage['xn-sidebar-collapsed']`，刷新后保持。

## 5. 自动化测试结果

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 前端单元测试 | `npm run test`（Vitest） | ✅ **16 通过 / 0 失败** |
| 前端类型检查 + 构建 | `npm run build` | ✅ 通过 |
| 后端测试（回归） | `go test -count=1 ./...` | ✅ 115 通过 / 0 失败 |
| 后端构建与静态检查 | `go build ./...`、`go vet ./...` | ✅ 通过 |
| OpenAPI 语法 | `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过（本阶段未改契约） |

新增前端单元测试覆盖（需求 7.2 中可纯逻辑验证的部分）：

| 测试点 | 用例 |
| --- | --- |
| 滚动时自动高亮分类 | `pickActiveIndex` 6 个用例（含"滚到底部取最后一个"、边界恰好越过阅读线、索引不越界） |
| 滚动边界判定 | `isAtBottom` 3 个用例（含文档短于视口） |
| 定位不越界 | `clampScrollTop` 1 个用例 |
| 手动图标保护规则 | `isManualIconType` 2 个用例（P2 规则的前端侧一致性） |
| 图标来源标签 | 3 个用例（含"品牌图标必须标注推测"） |

## 6. 浏览器验证结果

真实前后端（Go API `127.0.0.1:8090` + Vite `127.0.0.1:5199`），Chrome DevTools 实测：

| 视口 / 项 | 结果 |
| --- | --- |
| 1440×900 滚动模式 | ✅ 左侧完整分类栏（图标 + 名称 + 数量），"Development" 高亮；右侧渐变背景、时钟、搜索、分类与磁贴保持原样 |
| 点击侧栏分类 | ✅ 平滑滚动定位；滚到底部时高亮切换到最后一个分类（Tools） |
| 滚动回顶部 | ✅ 高亮自动回到第一个分类（Development） |
| 折叠 | ✅ 侧栏变为 64px 图标条，内容区扩展，状态写入 localStorage 并在刷新后保持 |
| 834×1000 平板 | ✅ 侧栏自动变图标条；胶囊条隐藏；无横向滚动 |
| 390×844 手机 | ✅ 侧栏隐藏；吸顶横向分类胶囊条显示且可横向滚动；4 列磁贴；无横向滚动 |
| 控制台 | ✅ 无 JS 错误 |

截图（真实运行页面）：

| 文件 | 内容 |
| --- | --- |
| `p3-scroll-1440.png` | 1440px 滚动模式（侧栏展开，Development 高亮） |
| `p3-scroll-collapsed2.png` | 侧栏折叠为图标条 |
| `p3-scroll-834.png` | 834px 平板（图标条） |
| `p3-scroll-390.png` | 390px 手机（吸顶胶囊条） |
| `p3-scroll-jump-bottom.png` | 点击分类后定位到页面底部并高亮末位分类 |

> 截图保存在本机 `%TEMP%\xn-browser\`，**未纳入版本库**（含临时测试数据）。
> 深浅主题的完整截图集在 P5 统一产出。

## 7. 未执行或未通过的测试

| 项 | 原因 |
| --- | --- |
| 组件级（DOM）单元测试 | 有意不引入 jsdom 组件快照测试，避免过度设计；交互由真实浏览器验证 |
| 触屏滑动与真机 | 无真机环境，仅用浏览器视口模拟 |
| 无障碍审计（屏幕阅读器） | 未做；侧栏已提供 `aria-label`、`aria-current`、`aria-expanded` |
| 100+ 分类时的侧栏滚动表现 | 计划在 P5 用真实数据集验证 |

## 8. 当前项目进度

| 阶段 | 状态 |
| --- | --- |
| v2.0.0 / v2.0.1 发布 | ✅ 已完成 |
| V2.1.0 / P1 调研与架构 | ✅ 已完成 |
| V2.1.0 / P2 官方图标自动获取 | ✅ 已完成 |
| **V2.1.0 / P3 滚动布局左侧分类栏** | ✅ **本阶段完成** |
| V2.1.0 / P4 标签布局与一键切换 | ⏳ 待开始 |
| V2.1.0 / P5 集成测试与发布 | ⏳ 待开始 |

## 9. Git Commit Hash 与推送状态

| 项 | 内容 |
| --- | --- |
| 本阶段起点 | `5e8c7d7`（`feat(icons): fetch, cache and manage official site icons`） |
| 本阶段提交 | 见聊天总结与 `git log`（本文件不硬写自身提交 SHA） |
| 推送 | 本阶段代码、测试与文档一并推送 `origin/main` |
| 标签 | 未创建新标签（v2.1.0 在 P5 发布） |

## 10. 遗留问题

| 问题 | 影响 | 计划 |
| --- | --- | --- |
| 平滑滚动过程中高亮会扫过中间分类 | 视觉上的短暂切换 | 属正常滚动联动；如需可加"点击后短暂锁定" |
| 侧栏在 1080px 以下强制图标条 | 用户手动展开状态在该断点被覆盖 | 有意取舍（窄屏空间优先）；P4 起布局切换控件会同时出现 |
| 移动端胶囊条遮挡页面顶部内容 | 吸顶元素固有 | 胶囊条高度仅 50px 左右，实测未遮挡分类标题 |
| 无 DOM 级组件测试 | 交互回归依赖真实浏览器 | 由 P5 的浏览器验收覆盖 |

## 11. 下一阶段工作（V2.1-P4）

1. 新增 `TabsLayout.vue`：顶部横向分类标签（含默认「首页」）、居中搜索、网格磁贴、独立背景层次。
2. 新增 `LayoutSwitcher.vue` 与 `composables/useLayout.ts`：一键切换、`localStorage['xn-layout']` 持久化、
   与主题完全独立（互不影响）。
3. 站点默认布局读取 `settings.homepage_layout`（`scroll`/`tabs`，旧值已归一化）。
4. `HomePage.vue` 按布局渲染；两种布局共享 `site.categories`、搜索与权限。
5. 补充 Vitest 用例（布局偏好持久化、主题与布局独立性）与真实浏览器验证（深浅 × 两种布局、移动端）。
6. 阶段交付文档 `P4-tabs-layout.md`；更新 `docs/PROGRESS.md`；提交并推送。
