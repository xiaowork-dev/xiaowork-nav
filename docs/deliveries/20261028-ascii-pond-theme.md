# ASCII Pond 可选主题 — 验收与修复交付记录

- 日期：2026-10-28
- 分支：`feat/ascii-pond-theme`（PR #1）
- 状态：**代码修复完成；前端类型检查/单元测试/构建、后端回归、真实浏览器验收（30 组配置）全部通过；未合并、未部署、未推送**
- 上游参考：<https://github.com/stqfdyr/ascii-pond>（MIT，Copyright (c) 2026 stqfdyr）

---

## 1. 本次任务目标

在完全保留 XiaoWork Nav 原有功能与布局的前提下，为导航站新增第三套可选视觉主题「ASCII Pond（锦鲤池塘）」，并让它达到可合并的质量标准：

1. 校验主题系统（三主题切换、本地持久化、旧偏好兼容、无闪烁、无重复挂载）。
2. 校验第三方 ASCII Pond 模块的接入方式（导入路径、Vite 构建产物、生产静态服务、生命周期与资源释放）。
3. 校验 CSS 覆盖是否完整（背景、侧栏、搜索框、卡片、对比度、两种布局、横向滚动）。
4. 执行 `typecheck` / `test` / `build`、后端 `build`/`vet`/`test`、Docker 构建（环境允许时）。
5. 在真实浏览器中按 5 种分辨率 × 2 种布局 × 3 种主题验收，并保留真实截图。
6. 修复属于本次改动的问题，生成交付文档并同步 `docs/PROGRESS.md`。

---

## 2. 原始代码检查结果

### 2.1 接入方式（正确，保留）

- `frontend/public/vendor/ascii-pond/{pond.js,koi.js,util.js,LICENSE}` 与上游 `src/` **逐字节一致**（sha256 全等），`LICENSE` 与上游 MIT 原文一致，未被改写。
- 引擎为原生 ES Module，`mount(target, options)` 返回 `{ pond, renderer, canvas, feed, setTheme, measure, destroy }`；`destroy()` 会移除 canvas 与全部 `window` 监听，符合「切走即释放」的要求。
- 组件按需 `import('/vendor/ascii-pond/pond.js')`（`@vite-ignore`），经典主题下不会下载/执行引擎。
- `palette.dark.page = [26,25,23]` 与 `--bg-base: #1a1917` 一致，符合上游「page 必须接近页面背景色」的约定，不会产生池塘与页面的色差。
- 面板类 CSS 变量在 `:root[data-theme='pond']` 中整体覆盖，深色/浅色主题的变量块未被污染。

### 2.2 发现的问题

| # | 问题 | 证据 | 影响 |
| --- | --- | --- | --- |
| 1 | **手机端顶部栏溢出**：新增的主题选择器宽 102px，导致 `.header-actions` 在 390px 视口右移到 425px，页面出现横向滚动条（main 分支同尺寸不溢出：390→390） | Chromium 实测 `scrollWidth 425 > innerWidth 390`，375px 时 425 > 375 | 移动端布局破相，属本次改动引入 |
| 2 | **首帧主题闪烁**：主题只在 `main.ts` 执行后才写入 `data-theme`，慢网络下浅色/池塘用户会先看到默认深色底 | 延迟 1.5s 加载 `/assets`：本分支首帧 `rgb(236,228,254)`；main 分支首帧 `rgb(80,67,150)`（应为浅色） | 视觉跳变，属既有缺陷，本次主题数量增加后更明显 |
| 3 | **站点占位字母对比度不足**：`.tile__letter/.tile__text/.tile__fallback` 用 `color: var(--accent)`，在池塘主题的暖白磁贴 `#e9e3da` 上对比度仅 **1.61:1** | 计算值：`#e6a878` on `#e9e3da` = 1.61:1（深色主题 3.49、浅色主题 5.92） | 无图标的站点几乎看不清首字母 |
| 4 | 主题选择器样式写在 `pond-theme.css`（该文件应只承载主题变量），且 `<label>` + `sr-only` + `aria-label` 三重重复 | 代码审阅 | 维护性问题，非视觉缺陷 |
| 5 | 动画生命周期与异步竞态逻辑直接写在组件里，无法用现有 node 环境单测覆盖；`ignore` 列表漏掉上游约定的 `[data-pond-ignore]` | 代码审阅 | 关键场景（快速切换、卸载释放）无回归保护 |
| 6 | 移动端锦鲤数量未按视口调整（固定 4 条） | 代码审阅 | 小屏多花 CPU，与「导航流畅优先」原则相悖 |

### 2.3 未发现问题的项（已核对）

- 主题状态机、`xn-theme` 存储键、`light/dark` 旧值兼容、`auto` 跟随系统：逻辑正确。
- 与服务端默认主题的优先级（本地选择优先）正确，后台设置不会覆盖浏览器选择。
- `pointer-events: none` + `aria-hidden="true"` + `z-index: 0`（`#app` 提到 `z-index: 1`）：画布不会拦截点击。
- 切换/卸载时 `destroy()` 调用完整，无重复监听（见 §7 的 19 项浏览器断言与 §6 的单元测试）。

---

## 3. 修复方案

1. **手机端顶部栏（问题 1）**
   - `HeaderActions.vue`：用 `matchMedia('(max-width: 700px)')` 切换紧凑文案（`深色/浅色/池塘`），选择器宽度 102px → 54px。
   - `main.css`（`≤700px`）：`.site-header` 间距 12→6、`.header-actions` 间距 8→6、`.header-actions .btn--sm` 内边距 11→9、`.layout-switch__item` 内边距 9→7。
   - `main.css`：`.brand { min-width: 0 }`，让站名走既有省略号，而不是把页面顶宽。
   - 结果：390/375/360 全部 `scrollWidth == innerWidth`，且 375px 下站名 `XiaoWork Nav` 完整显示（128/128），无截断。
2. **首帧闪烁（问题 2）**：`index.html` 头部加入内联脚本，在样式表加载前按 `xn-theme` 写入 `data-theme`；`useTheme` 仍是唯一事实来源，启动后会重新应用同一值。
3. **占位字母对比度（问题 3）**：`pond-theme.css` 中为三类磁贴文字指定深烤色 `#7a4a24`，对比度 1.61 → **5.81:1**（其余主题不受影响）。
4. **样式归位（问题 4）**：`.theme-picker__select` 规则移入 `main.css` 的头部区域；去掉多余 `<label>/sr-only`，保留 `aria-label`；`option` 的浅色/池塘配色分别补进对应主题块。
5. **生命周期可测化（问题 5）**：新增 `composables/useAsciiPond.ts`，把「懒加载 + generation 竞态守卫 + destroy + dispose」收敛为无 DOM 依赖的控制器；组件只做 `watch → sync` 与 `onUnmounted → dispose`；`ignore` 列表补 `[data-pond-ignore]`。
6. **小屏性能（问题 6）**：`<700px` 时锦鲤 4 → 3 条（上游 `grow` 会按尺寸自行调整个体，这里只调初始数量）。

以上修改均不触碰业务逻辑、数据库、API 契约、部署配置与既有布局尺寸（除手机端顶部栏的必要收紧）。

---

## 4. 修改文件清单

### 4.1 本轮修复（本次提交）

| 文件 | 变更 |
| --- | --- |
| `frontend/index.html` | +14 行：首帧主题内联脚本（防闪烁） |
| `frontend/src/components/HeaderActions.vue` | 主题选择器紧凑文案 + `matchMedia` 监听；移除冗余 label |
| `frontend/src/components/AsciiPondBackground.vue` | 改用 `createPondController`；小屏锦鲤数量；`ignore` 补 `[data-pond-ignore]` |
| `frontend/src/composables/useAsciiPond.ts` | **新增**：动画生命周期控制器（懒加载/竞态/释放） |
| `frontend/src/composables/useAsciiPond.test.ts` | **新增**：9 个生命周期与竞态用例 |
| `frontend/src/composables/useTheme.test.ts` | **新增**：11 个主题切换/持久化/兼容用例 |
| `frontend/src/styles/main.css` | 主题选择器样式、`.brand { min-width: 0 }`、≤700px 顶部栏收紧、`option` 浅色配色 |
| `frontend/src/styles/pond-theme.css` | 移除选择器样式；新增磁贴文字深色覆盖；`option` 池塘配色 |
| `docs/deliveries/20261028-ascii-pond-theme.md` | 本文件（重写为真实验收记录） |
| `docs/PROGRESS.md` | 同步本轮进度 |

### 4.2 PR 原有内容（保留，未改动）

`frontend/public/vendor/ascii-pond/{pond.js,koi.js,util.js,LICENSE}`、`frontend/src/composables/useTheme.ts`、`frontend/src/pages/HomePage.vue`、`frontend/src/main.ts`、`frontend/src/components/AsciiPondBackground.vue`（本轮改造）、`frontend/src/components/HeaderActions.vue`（本轮改造）、`frontend/src/styles/pond-theme.css`（本轮改造）。

---

## 5. 自动化测试结果

环境：Node v22.23.4 / npm 10.9.9；Go 1.27.3（本次为回归临时安装到 `~/.local/go`）。
注意：该环境 `NODE_ENV=production` 会让 `npm ci` 跳过 devDependencies，故使用 `NODE_ENV=development npm ci --include=dev`。

| 命令 | 结果 | 说明 |
| --- | --- | --- |
| `cd frontend && npm run typecheck` | ✅ 通过 | `vue-tsc --noEmit` 无错误 |
| `cd frontend && npm run test` | ✅ **53 passed / 6 files** | 原 33 个 + 新增 20 个（`useTheme` 11、`useAsciiPond` 9） |
| `cd frontend && npm run build` | ✅ 通过 | 102 模块；`dist` 424K，含 `dist/vendor/ascii-pond/*`（84KB） |
| `cd backend && go build ./...` | ✅ 通过 | |
| `cd backend && go vet ./...` | ✅ 通过 | |
| `cd backend && go test ./...` | ✅ 通过 | **115 个测试**（`-count=1` 全量） |
| `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过 | 契约未变更 |
| `docker compose -f deploy/compose.yaml --env-file deploy/.env config --quiet` | ✅ 通过 | 临时 `deploy/.env`（由 `.env.example` 复制），校验后已删除 |
| `docker build -t xiaowork-nav:pond-test .` | ⛔ **未执行** | `permission denied ... /var/run/docker.sock`：当前用户不在 `docker` 组，`sudo` 被 `no_new_privs` 禁用 |

新增单测覆盖的关键场景：

- 三主题切换与 `data-theme` 写入、`xn-theme` 持久化、刷新恢复；
- 旧版二进制偏好（`light`/`dark`）兼容、未知存储值忽略、`auto` 解析；
- 本地选择优先于服务端设置、清除覆盖后回落；
- 连续快速切换后落到最后一次选择；
- 惰性加载只挂载一个实例、切换离开即 `destroy`、卸载后 `dispose` 且不再挂载、模块加载失败静默降级。

---

## 6. 真实浏览器验收结果

工具：Playwright 1.64.1 + Chromium for Testing 156.0.8078.4（headless shell）。
运行环境：Linux 容器、无 GPU，Chromium 使用 **SwiftShader 软件渲染**（此前提会影响帧率绝对值，见 §9）。
被测站点：`vite preview`（生产 dist）+ 本地 Go 后端（`/api` 代理），数据为 8 个分类 / 96 个站点的本地测试数据（含示例数据，共 105 个磁贴）。
另用「内嵌 dist 的 Go 二进制」（生产静态路径）单独复核一次。

### 6.1 30 组配置矩阵（5 分辨率 × 2 布局 × 3 主题）

| 分辨率 | 布局 | dark | light | pond | 横向溢出 | 控制台错误 |
| --- | --- | --- | --- | --- | --- | --- |
| 1920×1080 | 滚动 / 标签 | ✅ | ✅ | ✅ | 0 | 0 |
| 1440×900 | 滚动 / 标签 | ✅ | ✅ | ✅ | 0 | 0 |
| 834×1194 | 滚动 / 标签 | ✅ | ✅ | ✅ | 0 | 0 |
| 390×844 | 滚动 / 标签 | ✅ | ✅ | ✅ | 0 | 0 |
| 375×812 | 滚动 / 标签 | ✅ | ✅ | ✅ | 0 | 0 |

每组均校验：主题正确落盘、105 个磁贴渲染、滚动布局 10 个分类项 / 标签布局 11 个标签、pond 下 canvas 存在且经典主题下为 0、无 `console.error` / `pageerror`。

### 6.2 功能断言（19 项，全部通过）

| 断言 | 实测 |
| --- | --- |
| 进入首页即挂载池塘画布 | canvas = 1 |
| 侧栏渲染分类 | 10 项 |
| 滚动布局不渲染标签栏 | tabs = 0 |
| 点击侧栏分类滚动定位 | `scrollY = 760` |
| 侧栏折叠 | 宽度 212 → 60 |
| 画布在滚动时保持贴视口 | `rect = (0,0,1440,900)` |
| 搜索框可获得焦点（未被画布遮挡） | `document.activeElement` = `.search__input` |
| 搜索过滤生效 | 输入 `Figma` → 2 个磁贴 |
| 点击空白水面投喂 | 该区域水面字符像素 0 → **756** |
| 投喂不触发跳转 | URL 不变 |
| 点击主题控件不投喂 | 4 → 0 |
| 切换到标签布局 | 11 个标签 |
| 切换布局后池塘仍在 | canvas = 1 |
| 点击标签切换分类 | 首页 → 开发工具 |
| 磁贴中心命中的是磁贴（非画布） | `elementFromPoint` 命中 `.tile__letter` |
| 离开池塘主题销毁画布 | canvas = 0，`data-theme = light` |
| 登录页在池塘配色下正常渲染 | 表单存在 |
| 控制台/页面错误 | 0 |

补充：鼠标在空水面上移动会搅动水面（引擎 `stir`）并让锦鲤转向指针（`koi.js` 中 `pointer.active` 分支），实测该区域水面字符像素 0 → 4（示意性，强度按上游默认 0.12）。

### 6.3 动画与资源

- 锦鲤游动：连续两帧 1.2s 间隔的全屏截图差异 **4.25%**（生产内嵌二进制）／4.35%（preview），确认动画在跑。
- 快速切换压力：`pond→dark→pond→light→pond…` 连续 8 次后，DOM 中 canvas 恒为 1，无重复实例、无监听泄漏迹象。
- 减少动态效果：`prefers-reduced-motion: reduce` 下同窗口水面像素变化 899，常规模式 2210（≈0.2 倍速，与上游设计一致）。

### 6.4 深色 / 浅色主题回归（与 main 分支逐像素比对）

对 5 分辨率 × 2 布局 × 2 主题共 20 组：

- **内容区（y > 96px）像素差异：0.000%**（20/20 组）；
- 全图差异 0.29%–1.26%，且全部落在顶部 96px 的头部区域——即「二态切换按钮 → 三态选择器」这一预期变更本身；
- 结论：深色/浅色主题的排版、间距、配色与原版本一致，未被池塘主题污染。

### 6.5 对比度

| 元素 | 池塘主题 | 深色主题 | 浅色主题 |
| --- | --- | --- | --- |
| 站名 `.brand__title` | 16.58 | — | — |
| 站点名称 `.tile__label` | 16.58 | — | — |
| 搜索输入框文字 | 14.11 | — | — |
| 时钟 `.clock__time` | 10.63 | — | — |
| 磁贴占位字母 | **5.81**（修复前 1.61） | 3.49（既有） | 5.92 |

### 6.6 首帧闪烁（修复验证）

延迟 `/assets/**` 1.5 秒后截图：

| 配置 | 首帧背景 |
| --- | --- |
| 本分支 + `light` | `rgb(236,228,254)` ✅ 直接是浅色 |
| 本分支 + `pond` | `rgb(26,25,23)` ✅ 直接是池塘底色 |
| main + `light` | `rgb(80,67,150)` ❌ 先闪深色（既有缺陷，已修） |

### 6.7 生产静态路径（替代 Docker 的等价验证）

以「Go 二进制 + `go:embed` 内嵌 dist」启动真实服务（`XN_DATA_DIR=/tmp/xn-embed-data`）：

- `/`、`/vendor/ascii-pond/pond.js`、`/vendor/ascii-pond/koi.js`、`/vendor/ascii-pond/LICENSE` 均 200；
- `Content-Type: text/javascript; charset=utf-8` + `X-Content-Type-Options: nosniff`（MIME 正确，否则 `nosniff` 会直接拦截模块加载）；
- 浏览器打开：`data-theme=pond`、canvas=1、96 个磁贴、动画差异 4.25%、无控制台错误。

说明：静态服务未设置 `Cache-Control`（所有静态资源一致，属既有行为，本次未改动）。

---

## 7. 截图保存位置

- 目录：`docs/screenshots/ascii-pond/`（**本地工件，按 `.gitignore` 约定不入库**）
- 共 34 个真实截图（Chromium 实际渲染，非 AI 生成）：
  - `{1920x1080,1440x900,834x1194,390x844,375x812}-{scroll,tabs}-{dark,light,pond}.png` —— 30 张矩阵截图；
  - `fullpage-1440x900-pond.png`、`fullpage-390x844-pond.png` —— 整页长图；
  - `admin-1440-pond.png`、`admin-settings-1440-pond.png` —— 后台在池塘配色下的复核。
- 说明：本环境无法在会话内做「人眼目视」复核，验收以**真实浏览器渲染 + 像素/几何/对比度量化断言**完成；截图已留存，建议合并前由维护者目视确认一张 PC 与一张手机池塘图。

---

## 8. Docker 构建结果

- `docker build -t xiaowork-nav:pond-test .` —— **未执行**：`permission denied while trying to connect to the docker API at unix:///var/run/docker.sock`；当前用户不在 `docker` 组，`sudo` 被禁用（容器安全策略）。
- 替代验证（已完成，见 §6.7）：本地构建带 `go:embed` 的二进制并实际运行，验证了 Dockerfile 中同一条链路的后半段（`dist` → embed → HTTP 静态服务 → 浏览器加载池塘模块）。
- `docker compose config --quiet` —— ✅ 通过（未触碰任何生产容器、镜像、数据目录或卷）。

---

## 9. 性能验证结果

测试机：容器内 Chromium，**SwiftShader 软件光栅化，无 GPU**；绝对帧率会明显低于真实设备，仅用于横向对比。

| 场景 | 空闲 FPS | 3 秒内 long task | 单帧 CPU 时间 |
| --- | --- | --- | --- |
| 1920×1080 深色 | 60.0 | 0 | 0.12 ms |
| 1920×1080 池塘 | 33.3 | 0 | **13.24 ms**（脚本 6.92 ms） |
| 1440×900 深色 | 60.2 | 0 | — |
| 1440×900 池塘 | 36.3–39.5 | 0 | — |
| 390×844 池塘 | 60.2 | 0 | — |

- 滚动流畅度：1920 空闲 33.3 → 滚动 38.6 FPS；1440 空闲 36.3 → 滚动 43.2 FPS。**滚动不比空闲更差**，即池塘没有拖慢交互本身。
- 成本结构（1920）：水面绘制 ≈7.6 ms/帧为固定成本，每条锦鲤 ≈1.1 ms 脚本；`count=0/2/4/8` 对应脚本 2.6/4.7/6.9/13.2 ms。
- 已采取的优化：手机（<700px）锦鲤 4→3；依赖上游「减少动态效果 → 0.2 倍速」「强制颜色模式隐藏画布」「rAF 随页面隐藏自动暂停」。
- 未做：修改上游引擎的内部渲染分辨率/帧率（会改变上游视觉与代码，收益不确定，暂不引入）。

---

## 10. 已知问题与未验证事项

1. **真实 GPU / 真机帧率未验证**：本环境为软件渲染，上述 FPS 偏保守；需在真实设备（尤其 1920 以上 + 集显）复核一次。
2. **Docker 镜像构建未执行**（权限原因），仅以等价链路验证。
3. **未推送、未更新 PR #1**：当前环境无 GitHub 凭据（无 token、无 SSH key、无 `gh`），`git push` 报 `could not read Username`；PR 描述与测试结果需由维护者推送后同步。
4. **iOS Safari / 微信内置浏览器未验证**（引擎依赖 ES Module、`ResizeObserver`、`color-mix()`）。
5. **真机触控投喂未验证**：仅验证桌面 `pointer` 事件；触摸路径在引擎中为「手指划过产生涟漪」，未在真机复核。
6. **`prefers-contrast: forced-colors`** 未实测（引擎按设计隐藏画布）。
7. **无障碍/屏幕阅读器**未审计；主题控件为原生 `<select>` + `aria-label`，键盘可用。
8. `<meta name="theme-color">` 仍为品牌蓝 `#303a83`，池塘/浅色主题下移动端地址栏颜色不随主题变化（既有行为，本次未改）。
9. 移动端顶部栏为容纳三态选择器收紧了间距（`≤700px`：6/6/9/7px），桌面端间距未变。

---

## 11. 当前 Git 状态

- 分支：`feat/ascii-pond-theme`（基于 `origin/feat/ascii-pond-theme`）
- 工作区：本轮修复已提交（提交信息见聊天总结；按 `AGENTS.md` 约定，本文件不写入 SHA）
- 远程：**未推送**（无凭据）
- 未执行：合并 `main`、创建 Release、推送/覆盖正式镜像、连接生产服务器、修改生产数据库、删除容器或数据卷

---

## 12. 下一步建议

1. 维护者在本机执行 `git push origin feat/ascii-pond-theme`，并把本文件的「验证情况」摘要贴到 PR #1 描述（不要新建 PR）。
2. 在具备 Docker 的机器上跑一次 `docker build -t xiaowork-nav:pond-test .` 与容器内冒烟（`/vendor/ascii-pond/pond.js` 200 + 页面可切换三主题）。
3. 在真实设备（含一台 1920 集显笔记本、一台 iPhone）目视复核池塘主题，并确认帧率与观感。
4. 若真机上大屏帧率不足，优先考虑「按视口降低锦鲤数量」或与上游同步 `cellW` 上限，而不是自改渲染器。
5. 合并后按既有流程发布下一版本（`v2.1.3` 或 `v2.2.0`），并在发布文档中说明「锦鲤主题为浏览器本地外观，不进入服务端主题枚举」。
