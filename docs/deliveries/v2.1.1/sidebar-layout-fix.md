# V2.1.1 增量交付记录：滚动模式左侧分类栏布局优化

> 本文件是 **V2.1.1 的正式增量交付文档**，可长期归档。
> 内容基于实际代码与真实浏览器验证编写；未验证内容均明确标注。
> 关联：[V2.1.0 实施计划](../../development-plan-v2.1.md)、[P3 交付报告](../v2.1.0/P3-scroll-layout.md)、[P5 交付报告](../v2.1.0/P5-release.md)、[AGENTS.md](../../../AGENTS.md)

---

## 1. 阶段目标

V2.1.0 发布后在生产环境实际使用中发现：滚动模式的左侧分类栏被放在**居中的内容容器内部**，
距离屏幕左边缘较远、且从搜索框下方才开始显示，整体像一张独立的悬浮卡片，不像真正的网站侧边导航。

本次目标：**让左侧分类栏真正成为页面整体布局的一部分**——只调整前端布局与视觉，
不改变任何数据、交互与后端行为。

## 2. 完成的工作

| 工作 | 结果 |
| --- | --- |
| 布局结构重构 | 滚动布局改为**全宽两栏外壳**（`page--rail`）：侧栏贴页面真实左边缘、从顶部导航下方延伸至视口底部；主体内容在剩余空间内重新居中 |
| 侧栏视觉重做 | 宽度 212px（要求 200–220px）、低饱和度深色半透明紫黑背景、去掉大圆角与阴影、仅保留 1px 右分隔线、当前分类柔和高亮 |
| 对齐规范 | 分类项改为 `图标 22px / 名称 1fr / 数量 auto` 三列网格，数量使用等宽数字，逐项整齐对齐 |
| 折叠 | 折叠为 60px 图标条（仅图标、居中），状态仍写入 `localStorage` |
| 响应式 | ≥1081px 完整侧栏；≤1080px 自动收为图标条；≤700px 隐藏侧栏、沿用吸顶横向分类胶囊条；三个断点均无横向溢出 |
| 标签模式 | **完全未改动**（仍使用居中容器，无侧栏） |
| 前端测试 | 新增 7 个布局契约单元测试（宽度区间、sticky + 100vh、无圆角/阴影、全宽外壳、内容重新居中、移动端隐藏侧栏、中等宽度收为图标条） |
| 真实浏览器验证 | 优化前后 1440px 截图、折叠截图、移动端与平板回归截图、独立滚动与滚动高亮实测 |

## 3. 新增和修改的文件

### 3.1 新增

| 文件 | 作用 |
| --- | --- |
| `frontend/src/styles/rail.test.ts` | 侧栏布局契约单元测试（7 个用例） |
| `docs/deliveries/v2.1.1/sidebar-layout-fix.md` | 本记录 |

### 3.2 修改

| 文件 | 变更 |
| --- | --- |
| `frontend/src/pages/HomePage.vue` | 滚动模式使用 `page--rail`（全宽外壳），标签模式保持 `page--tabs` |
| `frontend/src/components/ScrollLayout.vue` | DOM 重构为 `rail-layout`（侧栏 + 主体列），时钟/搜索/分类全部移入主体列的内层居中容器 |
| `frontend/src/components/CategorySidebar.vue` | 类名由 `sidebar*` 改为 `rail*`，网格化对齐，折叠态只渲染图标 |
| `frontend/src/styles/main.css` | 新增 `--rail-*` 主题变量与宽度变量；重写侧栏与两栏外壳样式；更新三档响应式规则 |
| `CHANGELOG.md` | 新增 `[2.1.1]` 条目 |
| `docs/PROGRESS.md` | 版本与交付记录同步 |

> **仅修改前端**：后端、数据库、部署脚本、OpenAPI 契约均未改动。

## 4. 实现的具体功能

### 4.1 整体布局结构（需求 1）

```
.page.page--rail              ← 全宽外壳（width:100%; max-width:none; padding:0）
├── .site-header              ← 仍为居中容器（min(100%, 1200px)），保留原有内边距
└── .rail-layout              ← grid: var(--rail-width) | minmax(0, 1fr)
    ├── .rail                 ← 侧栏：sticky top:0、height:100vh、贴左边缘、独立滚动
    └── .rail-main
        └── .rail-main__inner ← min(100%, 1200px) + margin:0 auto → 在剩余空间内居中
            ├── 时钟 / 搜索框
            ├── .chipbar      ← ≤700px 的吸顶横向分类条
            └── 分类区块
```

实测（1440×900，真实数据）：

| 指标 | 优化前 | 优化后 |
| --- | --- | --- |
| 侧栏左边缘 | 约 155px（居中容器内） | **0px（页面真实左边缘）** |
| 侧栏顶部 | 搜索框下方（约 380px） | **65px（顶部导航下方）** |
| 侧栏底部 | 内容高度（卡片式） | **视口底部**（滚动后实测 top=0 / bottom=900） |
| 侧栏宽度 | 236px | **212px** |
| 主体内容 | 与侧栏同处居中容器 | 在剩余空间内重新居中（时钟中心偏移 +99px） |

- 页面滚动时侧栏保持固定（`position: sticky; top: 0`），滚动后实测 `railTop=0`、`railBottom=视口高度`。
- 分类列表**独立滚动**：在 420px 高视口下实测列表 `scrollHeight > clientHeight`，
  滚动列表时页面 `scrollY` 保持 0。

### 4.2 侧栏视觉（需求 2）

| 要求 | 实现 |
| --- | --- |
| 宽度 200–220px | `--rail-width: 212px` |
| 低饱和度深色半透明紫黑背景 | `--rail-bg: rgba(19,15,42,0.62)`（深色主题）/ `rgba(26,21,58,0.90)`（浅色主题）+ `backdrop-filter: blur(20px)` |
| 去除大圆角浮动卡片感 | 侧栏无 `border-radius`、无 `box-shadow`；仅 1px 右分隔线（`--rail-border`） |
| 减少过重边框与高亮 | 分类项默认无边框；悬停为 6% 白色叠加；当前项为 20–26% 紫色柔和底（`--rail-active`），不加边框、不加粗 |
| 图标/文字/数量整齐对齐 | `grid-template-columns: 22px minmax(0,1fr) auto`，数量 `font-variant-numeric: tabular-nums` |
| 折叠只显示图标 | 折叠为 60px，分类项单列居中，标题与数量不渲染 |

配色仍由主题变量派生，与原有紫蓝渐变风格一致（未引入任何第三方资源）。

### 4.3 响应式（需求 3）

| 视口 | 行为 | 实测 |
| --- | --- | --- |
| ≥1081px | 完整侧栏（212px，图标 + 名称 + 数量） | ✅ 1440×900 |
| ≤1080px | 自动收为 60px 图标条（名称/数量由 CSS 隐藏） | ✅ 834×1000（`railWidth=60`） |
| ≤700px | 隐藏侧栏，改用吸顶横向分类胶囊条 | ✅ 390×844（`rail display:none`、`chipbar display:flex`） |

三个断点实测 `documentElement.scrollWidth <= innerWidth`（无横向滚动），
且侧栏不遮挡搜索框与网站图标（侧栏与主体为并列两栏，不存在覆盖）。

### 4.4 功能保持（需求 4）

| 功能 | 状态 |
| --- | --- |
| 点击分类平滑滚动 | ✅ 保留（`scrollToCategory` 未改动） |
| 滚动时自动高亮当前分类 | ✅ 保留（`pickActiveIndex` 纯函数 + rAF 节流；滚动后实测高亮跟随） |
| 分类栏独立滚动 | ✅ 保留并实测 |
| 折叠状态记忆 | ✅ 保留（`localStorage['xn-sidebar-collapsed']`） |
| 双布局一键切换 | ✅ 保留 |
| 明暗主题独立切换 | ✅ 保留（`xn-theme` / `xn-layout` 互不影响） |
| 分类与网站原有数据及顺序 | ✅ 未触碰数据层（仅前端样式与 DOM 结构） |

## 5. 自动化测试结果

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 前端单元测试 | `npm run test`（Vitest） | ✅ **28 通过 / 0 失败**（v2.1.0 为 21，新增 7） |
| 前端类型检查 + 构建 | `npm run build` | ✅ 通过 |
| 后端构建 / 静态检查 / 测试 | `go build ./...`、`go vet ./...`、`go test -count=1 ./...` | ✅ 115 通过 / 0 失败（本次未改后端，回归确认） |
| 脚本语法 | `bash -n scripts/*.sh scripts/tests/*.sh deploy/docker-entrypoint.sh` | ✅ 通过 |
| OpenAPI 语法 | `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过（本次未改契约） |

新增用例（`frontend/src/styles/rail.test.ts`）：

| 用例 | 断言 |
| --- | --- |
| 展开宽度在 200–220px | `--rail-width` ∈ [200, 220] |
| 折叠为图标条 | `--rail-width-collapsed` < 80 |
| 是整列而非悬浮卡片 | `.rail` 含 `position: sticky`、`height: 100vh`，且**不含** `box-shadow` / `border-radius` |
| 跳出居中容器 | `.page--rail` 为 `width:100%` + `max-width:none` + `padding:0` |
| 内容重新居中 | `.rail-main__inner` 为 `margin:0 auto` + `width:min(100%, var(--page-width))` |
| 移动端隐藏侧栏 | ≤700px 区间内 `.rail{display:none}` 且 `.chipbar{display:flex}` |
| 中等宽度收为图标条 | ≤1080px 区间内使用 `--rail-width-collapsed` 并隐藏名称 |

## 6. 浏览器验证结果

真实前后端（Go API `127.0.0.1:8090` + Vite `127.0.0.1:5199`）+ **真实导航数据**
（10 分类 / 110 站点，含 46 个自动获取图标），Chrome DevTools 实测：

| # | 项 | 文件 | 结果 |
| --- | --- | --- | --- |
| 1 | **优化前** 1440px 浅色 | `p5-scroll-light-1440.png` | 侧栏为居中容器内的悬浮卡片、从搜索框下方开始 |
| 2 | **优化前** 1440px 深色 | `p5-scroll-dark-1440.png` | 同上 |
| 3 | **优化前** 折叠 | `p5-sidebar-collapsed-1440.png` | 236px → 64px，仍为卡片 |
| 4 | **优化后** 1440px 浅色 | `v211-scroll-light-1440-after.png` | 侧栏贴左边缘（left=0）、顶部 65px、宽 212px、通高；主体内容重新居中 |
| 5 | **优化后** 1440px 深色 | `v211-scroll-dark-1440-after.png` | 同上（深色主题） |
| 6 | **优化后** 折叠 | `v211-scroll-collapsed-1440-after.png` | 60px 图标条，仅图标、居中；主体列同步扩展（60px + 1366px） |
| 7 | **优化后** 滚动状态 | `v211-scroll-scrolled-1440-after.png` | 侧栏固定（top=0 / bottom=视口高度），高亮跟随到当前分类 |
| 8 | 侧栏独立滚动 | `v211-rail-independent-scroll.png` | 420px 高视口下列表独立滚动，页面不动 |
| 9 | 移动端回归 | `v211-scroll-dark-390-after.png` | 390×844：侧栏隐藏、吸顶胶囊条、4 列磁贴、无横向溢出 |
| 10 | 平板回归 | `v211-scroll-light-834-after.png` | 834×1000：60px 图标条、无横向溢出 |
| 11 | 标签模式回归 | `v211-tabs-dark-1440-after.png` | 无侧栏、居中容器（1200px）、标签条与网格不变 |

其他实测：控制台无 JS 错误（仅 Vite 连接日志与浏览器自身的表单 `id` 提示）。

> 截图保存在本机 `%TEMP%\xn-browser\`，**未纳入版本库**（含真实站点数据）。

## 7. 未执行或未通过的测试

| 项 | 原因 |
| --- | --- |
| 跨浏览器（Firefox / Safari） | 仅 Chrome 实测；`position: sticky` 与 `backdrop-filter` 均为广泛支持特性 |
| 真机触控 | 无真机，仅用浏览器移动视口模拟 |
| 无障碍审计（屏幕阅读器） | 未做；侧栏保留 `aria-label` / `aria-current` / `aria-expanded`，折叠按钮含 `sr-only` 文本 |
| 生产服务器验证 | **按用户要求不做**，等待确认后升级 |

## 8. 当前项目进度

| 阶段 | 状态 |
| --- | --- |
| V2.0.0 / V2.0.1 | ✅ 已发布 |
| V2.1.0（P1–P5） | ✅ 已发布 |
| **V2.1.1 侧栏布局优化** | ✅ **本次完成并发布** |

## 9. Git Commit Hash 与推送状态

| 项 | 内容 |
| --- | --- |
| 本次起点 | `199eb96`（`docs(v2.1): record the v2.1.0 release and verification`） |
| 本次提交 | 见聊天总结与 `git log`（本文件不硬写自身提交 SHA） |
| 推送 | ✅ 已推送 `origin/main` |
| 发布标签 | ✅ `v2.1.1`（附注标签，指向本次修复提交） |
| 历史标签 | ✅ `v2.0.0` / `v2.0.1` / `v2.1.0` **未移动、未重建** |

### 9.1 发布验证（2026-10-26）

| 检查 | 结果 |
| --- | --- |
| CI（`main`） | ✅ 成功（run 40207896486） |
| CI（标签 `v2.1.1`） | ✅ 成功 |
| Release 流水线 | ✅ 成功（[run 40208136943](https://github.com/xiaowork-dev/xiaowork-nav/actions/runs/40208136943)） |
| GitHub Release | ✅ `v2.1.1` 已创建（非 draft / 非 prerelease）：`https://github.com/xiaowork-dev/xiaowork-nav/releases/tag/v2.1.1` |
| GHCR 镜像 | ✅ `ghcr.io/xiaowork-dev/xiaowork-nav:v2.1.1`，`linux/amd64` + `linux/arm64`，摘要 `sha256:2cc28538…` |
| 镜像标签 | ✅ `v2.1.1` / `v2.1` / `latest` 指向同一摘要 |
| 匿名拉取 | ✅ `docker pull ghcr.io/xiaowork-dev/xiaowork-nav:v2.1.1` 成功（无需登录） |
| 运行已发布镜像 | ✅ `/healthz` = `{"status":"ok","version":"v2.1.1"}`；`/api/v1/version` = `v2.1.1` / commit `c84aa52` / `schema_version=3`；容器非 root（`uid=100(app)`） |
| 发布产物含新布局 | ✅ 已发布镜像的静态资源包内包含 `page--rail`、`rail-layout`、`rail-main__inner`、`rail__item` |
| 历史镜像 | ✅ `v2.1.0` 仍为 `sha256:7a87f947…`、`v2.0.1` 仍为 `sha256:4083919b…`、`v2.0.0` 仍为 `sha256:c8e2151f…` |

## 10. 遗留问题

| 问题 | 影响 | 建议 |
| --- | --- | --- |
| 浅色主题下侧栏为深色 | 与页面浅色渐变形成较强对比（属本次设计意图） | 若希望浅色主题也用浅色侧栏，可调整 `--rail-bg` |
| 侧栏通高 100vh，首屏底部略超出视口 | 视觉上被裁切、无空白；滚动后完全贴合 | 有意取舍（滚动时才能满高固定） |
| 分类数量很多时需滚动侧栏 | 已支持独立滚动，无滚动条样式差异 | — |

## 11. 下一阶段工作

1. **等待用户确认后升级生产服务器到 v2.1.1**（指令见 §12）。
2. 按反馈微调侧栏配色（例如浅色主题的深浅程度）。
3. 规划后续版本（标签模式"常用网站"、图标清理工具、跨浏览器验证）。

## 12. 生产升级指令（**需用户确认后再执行**）

本次为**纯前端样式与结构优化**，无数据库结构变更（仍为 `schema_version=3`）。

```bash
ssh <user>@<server>
cd /opt/xiaowork-nav

git pull --ff-only
./scripts/update.sh v2.1.1          # 自动快照 → 切换 → 健康检查 → 失败自动回滚

curl -fsS http://127.0.0.1:<port>/api/v1/version   # 期望 version=v2.1.1, schema_version=3
```

> 升级不会影响任何数据；手动设置的图标与自动缓存的图标均保留。
> 浏览器可能需要强制刷新（`Ctrl/Cmd + Shift + R`）以更新缓存的静态资源。

**本次开发未对生产服务器做任何操作。**
