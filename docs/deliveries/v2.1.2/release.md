# V2.1.2 版本发布交付文档：滚动模式页面骨架重构

> 本文件是 **V2.1.2 正式版本交付文档**，可长期归档。
> 内容基于实际代码、真实数据、真实浏览器与真实 Docker/GitHub 发布流程编写；未验证内容均明确标注。
> 关联：[增量记录（开发过程）](../20261026-121645-scroll-shell-rework.md)、[V2.1.1 记录](../v2.1.1/sidebar-layout-fix.md)、[AGENTS.md](../../../AGENTS.md)

---

## 1. 版本基本信息

| 项 | 内容 |
| --- | --- |
| 项目名称 | XiaoWork Nav |
| 版本号 | **V2.1.2** |
| 版本类型 | 修复/优化版本（**纯前端**，无数据库结构变更） |
| 数据库结构版本 | **3（未变）** |
| 发布范围 | 仅滚动模式页面骨架与侧栏配色重构；**不新增功能** |
| 完成日期 | 2026-10-26 |
| 当前状态 | 已开发、已验收、已发布（详见 §7） |
| 上位依据 | `AGENTS.md`、[V2.1.0 实施计划](../../development-plan-v2.1.md) |

## 2. 本版本工作总结

### 2.1 背景

V2.1.1 把侧栏移到了页面左边缘，但生产使用后反馈四类问题：侧栏与主体之间存在大量空白、
深紫黑色侧栏与页面渐变严重割裂、顶部导航/搜索框/侧栏/正文缺乏统一对齐、整体像两个页面拼接。

### 2.2 根因

| 现象 | 根因 |
| --- | --- |
| 侧栏与主体大量空白 | `.rail-main__inner` 为 `width: min(100%, 1200px)` + `margin: 0 auto`，在 `1fr` 主列内**二次居中**：1440px 下约 54px、1920px 下约 **254px** 死空白 |
| 侧栏与渐变割裂 | 侧栏使用**独立深色实心背景** `rgba(26,21,58,.90)`（浅色主题同样为深色块），与页面不是同一套背景体系 |
| 对齐关系缺失 | 顶部导航在页面居中容器内，侧栏与主体在另一坐标系；品牌、按钮、搜索框、分类标题分属三套栅格 |

### 2.3 改动

**结构**：滚动模式改为单一应用外壳——整页一个 grid，左列固定为分类栏（品牌在顶部、从 y=0 延伸至视口底部、独立滚动），
右列占满剩余宽度且**内部不再二次居中**；顶栏承载操作按钮，与侧栏品牌行同为 64px，
并与内容共用同一水平间距令牌 `--shell-gutter`。

**配色**：侧栏改为在页面渐变上叠加低透明度蒙层（深色 `rgba(14,12,34,.30)` / 浅色 `rgba(255,255,255,.42)`），
文字改用主题文字变量，浅色主题不再出现深色块；分割线降至 7–10%，无阴影与装饰。

**组件**：新增 `BrandMark.vue`、`HeaderActions.vue`，两种布局共用品牌与操作区，避免样式漂移；
折叠态保留品牌标识（无 Favicon 时使用品牌渐变首字母块）。

### 2.4 与原计划的差异

本版本为 V2.1.0 发布后的 UI 修复迭代，原 P1–P5 计划未包含；已按规范补充独立交付文档
（开发过程记录 + 本发布文档）。

## 3. 代码和文件变更

### 3.1 新增

| 文件 | 作用 |
| --- | --- |
| `frontend/src/components/BrandMark.vue` | 品牌标识（Favicon 或品牌渐变首字母块 + 站点标题） |
| `frontend/src/components/HeaderActions.vue` | 右上角操作区（布局切换 / 主题 / 登录 / 管理后台） |
| `docs/deliveries/20261026-121645-scroll-shell-rework.md` | 开发过程增量记录 |
| `docs/deliveries/v2.1.2/release.md` | 本发布文档 |

### 3.2 修改

| 文件 | 变更 |
| --- | --- |
| `frontend/src/components/ScrollLayout.vue` | 渲染完整外壳：侧栏 + 主列顶栏 + 内容（含时钟/搜索/胶囊条/分类） |
| `frontend/src/components/CategorySidebar.vue` | 顶部增加品牌行；类名与结构随外壳调整 |
| `frontend/src/components/AppHeader.vue` | 改为组合 `BrandMark` + `HeaderActions`（标签模式与加载/错误态使用） |
| `frontend/src/pages/HomePage.vue` | 标签模式渲染 `AppHeader`，滚动模式交给外壳；标签模式结构不变 |
| `frontend/src/styles/main.css` | 新增外壳/侧栏令牌与规则；侧栏配色并入主题变量；响应式三档更新 |
| `frontend/src/styles/rail.test.ts` | 布局契约用例 7 → 12（含"内容列不得二次居中"回归守卫） |
| `CHANGELOG.md`、`docs/PROGRESS.md`、`docs/README.md` | 版本条目、进度与索引同步 |

> 后端、数据库、部署脚本、OpenAPI 契约**均未改动**。

## 4. 测试与验证结果

### 4.1 自动化回归（本机，2026-10-26）

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 后端构建 | `go build ./...` | ✅ 通过 |
| 后端静态检查 | `go vet ./...` | ✅ 通过 |
| 后端测试 | `go test -count=1 ./...` | ✅ **115 通过 / 0 失败** |
| 前端单元测试 | `npm run test`（Vitest） | ✅ **33 通过 / 0 失败**（4 个文件） |
| 前端类型检查 + 构建 | `npm run build` | ✅ 通过 |
| 脚本语法 | `bash -n scripts/*.sh scripts/tests/*.sh deploy/docker-entrypoint.sh` | ✅ 通过 |
| OpenAPI 语法 | `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过 |
| Compose 配置 | 独立模式 / NPM 模式 | ✅ 通过 |
| 部署异常注入 | `scripts/tests/deploy-faults.test.sh` | 见 §4.3 |

前端单元测试分布：布局契约 12、滚动高亮 11、图标规则 5、布局偏好 5。

### 4.2 真实浏览器验收（已由用户确认）

真实前后端 + 真实导航数据（10 分类 / 110 站点），Chrome DevTools 实测：

| 项 | 结果 |
| --- | --- |
| 侧栏右边缘 → 内容左边缘 @1440 | **0px**（V2.1.1 为 ≈54px） |
| 侧栏右边缘 → 内容左边缘 @1920 | **0px**（V2.1.1 为 ≈254px） |
| 分类标题左边界 @1920 | 252px（V2.1.1 为 ≈466px） |
| 时钟中心 = 内容列中心 @1440 | ✅ 818 = 818 |
| 顶栏高度 = 侧栏品牌行高度 | ✅ 64px = 64px |
| 横向溢出（1920 / 1440 / 834 / 390） | ✅ 均无 |
| 浅色主题侧栏 | ✅ 浅色半透明层，无深色块 |
| 折叠态 | ✅ 60px 图标条，保留品牌标识 |
| 手机端 | ✅ 侧栏隐藏、顶栏显示品牌、吸顶分类胶囊条 |
| 标签模式回归 | ✅ 结构未变（居中 1200px 容器、无侧栏） |
| 控制台 | ✅ 无 JS 错误 |

截图（真实运行页面，保存在本机 `%TEMP%\xn-browser\`，**未纳入版本库**）：
`v212-compare-vs-v211.png`（前后对比）、`v212-scroll-light-1440.png`、`v212-scroll-dark-1440.png`、
`v212-scroll-dark-1440-full.png`、`v212-scroll-light-1920.png`、`v212-scroll-collapsed-1440.png`、
`v212-scroll-dark-390.png`、`v212-tabs-light-1440.png`。

**用户已于 2026-10-26 完成视觉验收并批准发布。**

### 4.3 部署异常注入（真实 Docker）

`scripts/tests/deploy-faults.test.sh`：**14 个场景 / 57 个断言全部通过，0 失败（退出码 0）**。
场景清单与结果见 §7.4（该套件覆盖用户升级到 v2.1.2 时使用的完整路径）。

### 4.4 未执行的测试

| 项 | 原因 |
| --- | --- |
| 生产服务器升级 | **按用户要求不做**（用户将自行执行一键升级） |
| 真实 NPM 反代 + HTTPS 端到端 | 环境无 NPM 实例与公网域名 |
| 跨浏览器（Firefox / Safari） | 仅 Chrome 实测 |
| 真机触控 / 无障碍审计 | 无真机环境 |
| 长期稳定性与并发压力 | 未做 |

## 5. 项目总体进度

| 阶段/版本 | 状态 |
| --- | --- |
| P1–P5（V2.0.0） | ✅ 已发布 |
| V2.0.1 | ✅ 已发布 |
| V2.1.0（图标自动获取 / 侧栏 / 标签布局） | ✅ 已发布 |
| V2.1.1（侧栏贴边与视觉） | ✅ 已发布 |
| **V2.1.2（页面骨架重构）** | ✅ **本版本已发布** |

## 6. Git 版本记录

| 项 | 内容 |
| --- | --- |
| 当前分支 | `main` |
| 本版本起点 | `c8b8c70`（v2.1.1 发布记录） |
| 功能提交 | `8031fd8` — `refactor(ui): rebuild the scroll layout as one application shell` |
| 发布提交 | 见 §7.5（本文件不硬写自身 SHA） |
| 推送 | ✅ 已推送 `origin/main` |
| 发布标签 | ✅ `v2.1.2`（附注标签） |
| 历史标签 | ✅ `v2.0.0` / `v2.0.1` / `v2.1.0` / `v2.1.1` **均未移动、未重建** |

## 7. 发布结果

### 7.1 GitHub Actions

| 工作流 | 结果 |
| --- | --- |
| CI（`main`，提交 `903519d`） | ✅ 成功（[run 40272127455](https://github.com/xiaowork-dev/xiaowork-nav/actions/runs/40272127455)） |
| CI（标签 `v2.1.2`） | ✅ 成功 |
| Release 流水线 | ✅ 成功（[run 40272624224](https://github.com/xiaowork-dev/xiaowork-nav/actions/runs/40272624224)） |

### 7.2 容器镜像

| 项 | 值 |
| --- | --- |
| 镜像地址 | `ghcr.io/xiaowork-dev/xiaowork-nav:v2.1.2` |
| 同步标签 | `v2.1.2` / `v2.1` / `latest`（同一摘要） |
| 摘要 | `sha256:91985c0b2b4d72b20ae7a88dfb66e43686eeddcb150fce4e5eb84dfb16207f58` |
| 架构 | ✅ `linux/amd64` + `linux/arm64` |
| 匿名拉取 | ✅ `docker pull ghcr.io/xiaowork-dev/xiaowork-nav:v2.1.2` 成功（无需登录） |
| 启动与健康检查 | ✅ 容器 `healthy`，非 root（`uid=100(app)`） |
| `/healthz` | ✅ `{"status":"ok","version":"v2.1.2"}` |
| `/api/v1/version` | ✅ `version=v2.1.2`、`commit=903519d`、**`schema_version=3`** |
| `--db-info` | ✅ 返回 `exists=true`、`schema_version=3` |
| 产物含本次布局 | ✅ 已发布镜像的静态资源包含 `page--rail`、`shell__topbar`、`shell__content`、`rail__brand`、`brand__mark`，且**不再包含**旧的 `rail-main__inner` |
| 历史镜像未变 | ✅ `v2.1.1` = `sha256:2cc28538…`、`v2.1.0` = `sha256:7a87f947…`、`v2.0.1` = `sha256:4083919b…`、`v2.0.0` = `sha256:c8e2151f…` |

### 7.3 Release

| 项 | 值 |
| --- | --- |
| 地址 | https://github.com/xiaowork-dev/xiaowork-nav/releases/tag/v2.1.2 |
| 名称 | XiaoWork Nav v2.1.2 |
| 状态 | 非 draft、非 prerelease |
| 发布时间（UTC） | 2026-10-26T05:08:11Z |

### 7.4 部署异常注入结果（真实 Docker）

`scripts/tests/deploy-faults.test.sh`：**14 个场景 / 57 个断言全部通过，0 失败（退出码 0）**。

| # | 场景 | 结果 |
| --- | --- | --- |
| 1 | 应用运行时读取数据库文件 | ✅ |
| 2 | 固定版本已部署 → 幂等 | ✅ |
| 3 | `latest` 镜像未变化 → 不重建容器 | ✅ |
| 4 | 目标镜像不可用 → 不改动现场 | ✅ |
| 5 | 镜像拉取失败（本地无副本） | ✅ |
| 6 | 容器启动失败 → 回滚并验证 | ✅ |
| 7 | 迁移后健康检查失败 → 先恢复快照再回滚 | ✅ |
| 8 | 新镜像无法报告结构版本 → 保守回滚 | ✅ |
| 9 | 应用停止时读取结构版本 | ✅ |
| 10 | `restore.sh` 启动前切换版本 | ✅ |
| 11 | 快照版本未知 → 拒绝启动 | ✅ |
| 12 | NPM 模式（无宿主机端口）升级失败 → 回滚 | ✅ |
| 13 | 从发布的 v2.0.1 升级：结构 2→3、数据与手动图标保留 | ✅ |
| 14 | 只读数据目录 → 快速失败并给出修复指引 | ✅ |

> 该套件覆盖的正是用户升级到 v2.1.2 时使用的路径（快照 → 切换 → 健康检查 → 失败回滚）。

### 7.5 发布提交

| 项 | 值 |
| --- | --- |
| 功能提交 | `8031fd8` — `refactor(ui): rebuild the scroll layout as one application shell` |
| 发布文档提交 | `903519d` — `docs(v2.1.2): prepare the release notes and delivery report`（标签 `v2.1.2` 指向此提交） |
| 标签 | `v2.1.2`（附注标签，指向 `903519d`） |
| 本文件的后续更新提交 | 见 `git log`（不硬写自身 SHA） |

## 8. 遗留问题与风险

| 问题 | 影响 | 建议 |
| --- | --- | --- |
| 浅色主题侧栏为浅色磨砂层，比主体略亮 | 视觉偏好差异 | 可降低 `--rail-bg` 透明度或改为仅留分隔线 |
| 时钟/搜索框按"主内容区"居中，宽屏下偏右于页面几何中心 | 设计取舍（符合"在主体区域居中"） | 如需改为页面居中可调整 |
| 极宽屏（>1920）瓦片铺满主列，列数较多 | 视觉密度 | 如需收敛可引入上限，但会重新产生右侧留白 |
| 跨浏览器 / 真机 / 无障碍审计未做 | 未知差异 | 后续补充 |

## 9. 生产升级指引（**由用户自行执行**）

生产环境当前为 `v2.1.1`，本次升级**无数据库结构变更**（`schema_version` 仍为 3）：

```bash
ssh <user>@<server>
cd /opt/xiaowork-nav

./scripts/backup.sh                 # 可选但建议
git pull --ff-only
./scripts/update.sh v2.1.2          # 自动快照 → 切换 → 健康检查 → 失败自动回滚

curl -fsS http://127.0.0.1:<port>/api/v1/version   # 期望 version=v2.1.2, schema_version=3
docker inspect -f '{{.State.Health.Status}}' xiaowork-nav   # healthy
```

> 浏览器可能需要强制刷新（`Ctrl/Cmd + Shift + R`）以更新缓存的静态资源。
> 升级不会影响任何数据；手动设置与自动缓存的图标均保留。
> **本次发布未对生产服务器做任何操作。**
