# 项目进度总览

> 最后更新：2026-10-28 05:55（Asia/Shanghai）
> 项目：XiaoWork Nav V2.0（版本 **2.1.2**；`feat/ascii-pond-theme` 分支在研）
> 总体状态：**v2.1.2 已发布** —— 滚动模式页面骨架重构（纯前端，无数据库结构变更）；
> 用户已完成视觉验收。**生产服务器仍为 v2.1.1，升级由用户自行执行。**
> 另有 `feat/ascii-pond-theme`（PR #1）：ASCII Pond 可选主题已完成修复与真实浏览器验收，
> **尚未推送、未合并**。
>
> 交付规范见 [`../AGENTS.md`](../AGENTS.md)：每阶段必须产出 `docs/deliveries/P<N>-delivery.md`。

## 项目目标与当前阶段

- 核心目标：将旧版个人导航站重构为轻量、可自托管、可公开开源、支持一键部署与一键更新的产品。
- 当前阶段：**P5 Docker 部署、GitHub 开源与版本发布** —— ✅ 已完成；验收复审后进入
  **P5 补充（v2.0.1 一键更新与回滚修复）** —— ✅ 修复完成并通过真实验证。
- 本轮里程碑：仓库创建并推送、CI 全绿、多架构（amd64/arm64）镜像发布到 GHCR 且匿名可拉取、GitHub Release `v2.0.0` 发布；发布过程中修复了 QEMU 构建过慢与镜像标签缺少 `v` 前缀两个问题。**v2.0.1** 修复了 `latest` 更新判定、NPM 模式结构版本检测、失败自动回滚与 `restore.sh` 安全恢复。

## 已完成（仅列已经验证的成果）

| 成果 | 状态 | 验证依据 | 交付记录 |
| --- | --- | --- | --- |
| P1 需求/契约/初始化/后端骨架 | 已完成 | `go test` + 运行时冒烟 | [P1 报告](deliveries/P1-delivery.md) |
| P2 后端核心业务 | 已完成 | 53 个后端测试 + 二进制冒烟 | [P2 报告](deliveries/P2-delivery.md) |
| 契约一致性自动检查 | 已完成 | `TestRoutesMatchOpenAPIContract` | [P3 报告](deliveries/P3-delivery.md) |
| SQLite 持久化与迁移幂等 | 已完成 | `TestDataPersistsAcrossReopen` | [P3 报告](deliveries/P3-delivery.md) |
| 导航首页与搜索 | 已完成 | 浏览器实测（搜索命中 ChatGPT） | [P3 报告](deliveries/P3-delivery.md) |
| 初始化向导 / 登录 / 退出 | 已完成 | 浏览器实测 | [P3 报告](deliveries/P3-delivery.md) |
| 分类管理（含拖拽排序） | 已完成 | 浏览器实测（顺序持久化为 10/20/30） | [P3 报告](deliveries/P3-delivery.md) |
| 网站管理（含图标/移动） | 已完成 | 浏览器实测 | [P3 报告](deliveries/P3-delivery.md) |
| 系统设置 | 已完成 | 浏览器实测（标题即时生效） | [P3 报告](deliveries/P3-delivery.md) |
| 公开/私有权限区分 | 已完成 | 浏览器实测（私有站点不公开；未登录 401/跳转） | [P3 报告](deliveries/P3-delivery.md) |
| 数据导入导出与备份 | 已完成 | 浏览器实测（导出下载、导入 legacy 预览与执行、备份创建） | [P3 报告](deliveries/P3-delivery.md) |
| 前端类型检查与构建 | 已完成 | `npm run build` 通过 | [P3 报告](deliveries/P3-delivery.md) |
| 旧站真实数据迁移（测试库） | 已完成 | 预览/导入 API + 浏览器渲染验证；10 分类 / 110 站点 / 顺序与图标保留 | [迁移报告](legacy-data-migration-report.md) |
| 图标显示链（url/upload/simple/text/favicon→Favicon→首字母） | 已完成 | 浏览器实测：有图标网站 49 → 88，占位 61 → 22 | [迁移报告 §13](legacy-data-migration-report.md) |
| 数据模型扩展（分类图标/颜色、标签、颜色） | 已完成 | 迁移 `0002` 非破坏性加列 + `TestExtendedColumnsExist` | [迁移报告 §13](legacy-data-migration-report.md) |
| Simple Icons 白名单校验 | 已完成 | 3,494 个官方 slug 内置；4/8 命中，4 项无效保留为关键词 | [迁移报告 §13](legacy-data-migration-report.md) |
| 迁移安全加固（SVG/上传/回滚/孤儿文件/备份） | 已完成 | 7 项专项测试全部通过 | [迁移报告 §13](legacy-data-migration-report.md) |
| UI/交互/主题优化（P4） | 已完成 | 真实截图 20 张 + 浏览器交互验证；空白磁贴 0；图标 49→87 | [P4 报告](deliveries/P4-delivery.md) |
| P4 收尾优化（移动端后台/分页/排序模式/表单） | 已完成 | 390px 无横向溢出与竖排按钮；分页 20/50/100；排序模式拖拽已持久化 | [P4 报告 §9](deliveries/P4-delivery.md) |
| Docker 一键部署工件与脚本 | 已完成 | 镜像构建成功（36.2 MB）；`install/deploy/update/backup/restore` 脚本通过语法检查 | [P5 报告](deliveries/P5-delivery.md) |
| 真实 Docker 环境部署测试 | 已完成 | 16 项测试全部通过（全新安装、持久化、旧数据导入、备份、升级迁移、降级保护、回滚） | [P5 报告 §4.2](deliveries/P5-delivery.md) |
| 数据库降级保护 | 已完成 | 旧镜像打开新结构数据库时拒绝启动（实测） | [P5 报告 §2.4](deliveries/P5-delivery.md) |
| CI / Release 流水线 | 已完成 | CI 全绿（main + tag）；Release 成功构建并推送多架构镜像 | [P5 报告 §9](deliveries/P5-delivery.md) |
| GitHub 仓库与镜像发布 | 已完成 | 仓库公开；`ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.0` 匿名拉取成功并运行正常 | [P5 报告 §9](deliveries/P5-delivery.md) |
| GitHub Release | 已完成 | `v2.0.0`（Latest）已创建并含安装/升级说明 | [Release](https://github.com/xiaowork-dev/xiaowork-nav/releases/tag/v2.0.0) |
| **v2.0.1 一键更新逻辑修复** | 已完成 | 异常注入场景 3/4/5：`latest` 未变化不重建容器、镜像缺失与拉取失败均不改动现场 | [补充记录 §2.1](deliveries/20261025-170154-p5-update-rollback-fix.md) |
| **v2.0.1 NPM 模式结构版本检测** | 已完成 | 新增 `--db-info`；场景 1/9 实测：容器运行时与停止时均可读 `schema_version=2`，不依赖宿主机端口 | [补充记录 §2.2](deliveries/20261025-170154-p5-update-rollback-fix.md) |
| **v2.0.1 更新失败自动回滚** | 已完成 | 场景 6/7/8/12：启动失败、迁移后健康检查失败、结构版本不可读、NPM 模式失败均回滚并**再次验证**健康与数据 | [补充记录 §2.3](deliveries/20261025-170154-p5-update-rollback-fix.md) |
| **v2.0.1 restore.sh 安全恢复** | 已完成 | 场景 10/11：启动前切换到快照版本；版本未知时拒绝启动并保持停止 | [补充记录 §2.4](deliveries/20261025-170154-p5-update-rollback-fix.md) |
| **部署脚本异常注入测试** | 已完成 | `scripts/tests/deploy-faults.test.sh` 12 场景 / 43 断言全部通过（真实 Docker） | [补充记录 §4.2](deliveries/20261025-170154-p5-update-rollback-fix.md) |
| **v2.0.1 提交、推送与发布** | 已完成 | 提交 `825f5bc` 已推送；CI（main/标签）与 Release 流水线成功；多架构镜像已推送 GHCR 且匿名拉取并运行通过；Release `v2.0.1` 已创建 | [补充记录 §4.4](deliveries/20261025-170154-p5-update-rollback-fix.md) |
| 历史标签 `v2.0.0` 保持不变 | 已完成 | 标签仍指向 `d18539d`；镜像摘要仍为 `sha256:c8e2151f…` | [补充记录 §4.4](deliveries/20261025-170154-p5-update-rollback-fix.md) |
| **V2.1.0 / P1 调研与架构** | ✅ 已完成 | 见 [V2.1.0 实施计划](development-plan-v2.1.md) | [P1 报告](deliveries/v2.1.0/P1-architecture.md) |
| **V2.1.0 / P2 官方图标自动获取** | ✅ 已完成 | 迁移 0003、抓取与缓存、SSRF 加固、异步批量、后台 UI；真实数据补全 45/53 | [P2 报告](deliveries/v2.1.0/P2-icons.md) |
| **V2.1.0 / P3 滚动布局左侧分类栏** | ✅ 已完成 | 侧栏、点击定位、滚动高亮（11 个单元测试）、折叠与三档响应式 | [P3 报告](deliveries/v2.1.0/P3-scroll-layout.md) |
| **V2.1.0 / P4 标签布局与一键切换** | ✅ 已完成 | 标签布局、切换控件、偏好持久化、主题与布局独立 | [P4 报告](deliveries/v2.1.0/P4-tabs-layout.md) |
| **V2.1.0 / P5 集成测试与发布** | ✅ 已完成 | 14 个异常注入场景、真实数据兼容、Docker 权限修复、v2.0.1→v2.1.0 升级/回滚、完整截图、发布 v2.1.0 | [P5 报告](deliveries/v2.1.0/P5-release.md) |
| **v2.1.0 发布核验** | ✅ 已完成 | CI/Release 成功；`ghcr.io/.../v2.1.0`（amd64+arm64）匿名拉取并运行；Release `v2.1.0` 已创建 | [P5 报告 §10.1](deliveries/v2.1.0/P5-release.md) |
| **v2.1.1 侧栏布局优化** | ✅ 已完成 | 侧栏改为贴左边缘、通高、独立滚动、无卡片感；212px 宽；28 个前端单元测试；真实数据浏览器实测（含优化前后截图） | [V2.1.1 增量记录](deliveries/v2.1.1/sidebar-layout-fix.md) |
| **滚动模式页面骨架重构** | ✅ 已完成 | 整页单一 grid 外壳；修复主列二次居中造成的 54–254px 死空白；侧栏并入主题背景体系（浅色不再用深色块）；品牌入栏、顶栏 64px 对齐；33 个前端单元测试 | [增量记录](deliveries/20261026-121645-scroll-shell-rework.md) |
| **v2.1.2 版本发布** | ✅ 已完成 | 用户视觉验收通过；完整回归（后端 115 / 前端 33）；标签 `v2.1.2` → CI/Release → GHCR 多架构 → 匿名拉取与健康检查 | [V2.1.2 发布文档](deliveries/v2.1.2/release.md) |
| **ASCII Pond 可选主题（`feat/ascii-pond-theme`）** | ✅ 已推送、PR #1 已更新、CI 全绿；待维护者人眼终审 | 前端 53 / 后端 115 测试、typecheck、build、OpenAPI、compose 全绿；30 组浏览器配置零溢出零报错；深色·浅色内容区与 main 像素 0.000% 差异；磁贴区动画干扰 0.000%；CI 四项 pass（含 Docker 镜像构建与容器冒烟 1m21s）；性能 1920≈32–39 / 1440≈43–49 / 390≈60 FPS（软渲染） | [交付记录](deliveries/20261028-ascii-pond-theme.md) |

## 正在进行 / 未完成

| 工作项 | 现状 | 下一动作 |
| --- | --- | --- |
| **ASCII Pond 人眼视觉终审** | 未完成：执行环境无图像输入能力 | 维护者目视 `docs/screenshots/ascii-pond/final-*` 六张截图，确认后合并 PR #1 |
| **池塘主题容器内带数据冒烟** | CI 仅覆盖首页与静态资源 | 需要时在本地 Docker 建管理员、导数据、验三主题切换 |
| **池塘主题真机 / 真实 GPU 帧率** | 未验证：验收环境为 SwiftShader 软渲染 | 在真机（含 1920 集显与 iPhone）复核帧率与观感 |
| **生产服务器升级到 v2.1.2** | 生产当前为 v2.1.1；**由用户自行执行** | 按 [V2.1.2 发布文档 §9](deliveries/v2.1.2/release.md) 执行 `git pull --ff-only && ./scripts/update.sh v2.1.2` |
| 独立测试服务器真实部署 + NPM 反代 HTTPS | 未验证 | 在具备 NPM 与域名的环境完成 |
| GHCR 遗留标签 `2.0.0` / `2.0` 清理 | 可选 | 包管理页删除摘要 `cf6040d5…` 的版本（P5 报告 §9.3） |
| 替换占位提交身份 | 待用户提供 | 提供真实 Git 身份后更新 |
| 跨浏览器 / 真机触控 / 无障碍审计 | 未验证 | 后续版本 |
| OpenAPI 规范级 lint / 前端类型自动生成 | 未接入 | 后续接入 CI |
| 旧站真实数据的正式导入 | 已可在容器中导入 | 剩余 23 个占位图标见迁移报告 §13.5 |
| v2.1.0 图标自动补全后的少量失败站点 | 8 个站点因 403/404/超时无法自动获取 | 可在后台手动指定图标 |
| 真实 NPM 反代 HTTPS 端到端 | 未验证 | 在具备 NPM 与域名的环境完成 |

## 当前阻塞与风险

- **无 NPM 实例与公网域名**：反代与 HTTPS 未实测。
- **受限网络下 Go 模块代理不可达**：构建镜像需传 `--build-arg GOPROXY=...`（已文档化）。
- **提交身份为占位**：待用户替换。
- **部分站点图标依赖第三方**：23 个站点显示首字母、4 个第三方 Favicon 内容透明。
- **异常注入测试需要 Docker 与可用的 Go 模块代理**：构建测试镜像时可用 `XN_TEST_GOPROXY` 指定镜像源。

## 下一步（按优先级）

1. **由用户自行将生产服务器从 v2.1.1 升级到 v2.1.2**（指令见 [V2.1.2 发布文档 §9](deliveries/v2.1.2/release.md)）。
2. 在独立测试服务器完成真实部署与 NPM 反代 HTTPS 验证。
3. 可选：清理 GHCR 遗留标签 `2.0.0`/`2.0`、替换占位提交身份。
4. 规划后续版本增强项（标签模式"常用网站"、图标清理工具、跨浏览器验证）。

## 最近交付记录

- **[V2.1.2 版本发布交付文档：滚动模式页面骨架重构](deliveries/v2.1.2/release.md)**（含生产升级指引）
- **[滚动模式页面骨架重构（增量记录）](deliveries/20261026-121645-scroll-shell-rework.md)**
- **[V2.1.1 增量交付记录：滚动模式左侧分类栏布局优化](deliveries/v2.1.1/sidebar-layout-fix.md)**
- **[V2.1.0 / P5 阶段交付报告：集成测试、部署修复与版本发布](deliveries/v2.1.0/P5-release.md)**
- **[V2.1.0 / P4 阶段交付报告：标签布局与一键切换](deliveries/v2.1.0/P4-tabs-layout.md)**
- **[V2.1.0 / P3 阶段交付报告：PC 端滚动布局左侧分类栏](deliveries/v2.1.0/P3-scroll-layout.md)**
- **[V2.1.0 / P2 阶段交付报告：官方图标自动获取](deliveries/v2.1.0/P2-icons.md)**
- **[V2.1.0 / P1 阶段交付报告：调研与架构](deliveries/v2.1.0/P1-architecture.md)**
- **[V2.1.0 实施计划](development-plan-v2.1.md)**
- **[P5 阶段交付报告](deliveries/P5-delivery.md)**（正式归档，含 §12 v2.0.1 补充）
- **[P5 补充交付记录：一键更新与回滚修复（v2.0.1）](deliveries/20261025-170154-p5-update-rollback-fix.md)**
- **[P4 阶段交付报告](deliveries/P4-delivery.md)**（正式归档）
- **[P3 阶段交付报告](deliveries/P3-delivery.md)**（正式归档）
- [旧站数据迁移报告（含 §13 P3 收尾补充）](legacy-data-migration-report.md)
- **[P2 阶段交付报告](deliveries/P2-delivery.md)**（正式归档）
- [P1 阶段交付报告](deliveries/P1-delivery.md)（正式归档）

## 验证与环境说明

- 最近一次验证（2026-10-25 17:00，v2.0.1 修复）：
  - 后端：`go build ./...`、`go vet ./...`、`go test -count=1 ./...` 通过（**75 通过 / 0 失败**）。
  - 前端：`npm run build`（含 `vue-tsc` 类型检查）通过。
  - 脚本：`bash -n` 全部通过；`docker compose config`（独立模式、NPM 模式）通过；OpenAPI YAML 语法通过。
  - **真实 Docker 异常注入**：`scripts/tests/deploy-faults.test.sh` **12 场景 / 43 断言全部通过**
    （覆盖 `latest` 未变化不重建、镜像缺失、拉取失败、启动失败、迁移后健康检查失败、
    结构版本不可读、容器停止时读结构版本、`restore.sh` 版本切换与拒绝启动、NPM 模式失败回滚）。
  - **未执行**：真实 NPM 反代与 HTTPS 实测、真实 GHCR 新版本拉取升级（v2.0.1 发布前镜像不存在）、
    长期稳定性与并发、跨浏览器与真机、Linux 原生运行测试脚本。
- 上一次验证（2026-10-25 08:00，v2.0.0 发布）：
  - 后端：`go build ./...`、`go vet ./...`、`go test -count=1 ./...` 通过（**71 通过 / 0 失败**）。
  - 前端：`npm run build`（含 `vue-tsc` 类型检查）通过。
  - Docker：镜像构建成功（36.2 MB）；真实环境完成 16 项部署测试（全新安装、非 root、健康检查、
    宿主机持久化、初始化、旧数据导入 10 分类/110 站点、down+up 数据保留、备份、
    升级迁移 schema 2→3、**降级保护拒绝启动**、快照回滚后数据完整）。
  - GitHub：仓库 `xiaowork-dev/xiaowork-nav` 已推送；CI（main / tag）全部成功；
    Release 成功构建 `linux/amd64` + `linux/arm64` 并推送 GHCR；Release `v2.0.0` 已创建；
    **匿名 `docker pull ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.0` 成功**，运行后 `/healthz` 正常、
    版本 `v2.0.0`、`schema_version=2`。
  - **未执行**：NPM 反代与 HTTPS 实测、长期稳定性与并发、跨浏览器与真机。
- 部署环境：本地 Docker Desktop（测试用临时数据目录，测试后已清理）；**未操作任何现有服务器**。
- v2.1.2 验证（2026-10-26，纯前端页面骨架重构）：
  - 后端回归 `go build` / `go vet` / `go test` **115 通过 / 0 失败**；
    前端 `npm run test` **33 通过 / 0 失败**（布局契约用例 7 → 12）；前端构建、脚本语法、OpenAPI、Compose 均通过。
  - **部署异常注入**：`scripts/tests/deploy-faults.test.sh` **14 个场景 / 57 个断言全部通过**（退出码 0）。
  - **浏览器验收**（真实数据 10 分类 / 110 站点，已由用户确认）：侧栏右边缘→内容左边缘
    1440 下 54px → **0px**、1920 下 254px → **0px**；顶栏与侧栏品牌行同为 64px；
    浅色主题侧栏改为浅色半透明层（无深色块）；折叠保留品牌标识；手机端品牌回到顶栏；标签模式未变。
  - **发布核验**：CI（main `40272127455`、标签）与 Release 流水线（`40272624224`）成功；
    `ghcr.io/xiaowork-dev/xiaowork-nav:v2.1.2`（amd64 + arm64，`sha256:91985c0b…`）**匿名拉取成功**，
    运行后 `healthy`、非 root、`/healthz` = `v2.1.2`、`/api/v1/version` = `v2.1.2` / `schema_version=3`；
    产物含新外壳类名且不含旧的 `rail-main__inner`；GitHub Release `v2.1.2` 已创建；
    历史镜像 `v2.0.0` / `v2.0.1` / `v2.1.0` / `v2.1.1` 摘要未变。
  - **未执行**：生产服务器升级（由用户自行执行）、真实 NPM 反代与 HTTPS 端到端、
    跨浏览器（Firefox/Safari）、真机触控与无障碍审计、长期稳定性与并发。
- v2.1.1 验证（2026-10-26，纯前端 UI 优化）：
  - 前端 `npm run test` **28 通过 / 0 失败**（v2.1.0 为 21，新增 7 个侧栏布局契约用例）；构建通过。
  - 后端回归 `go build` / `go vet` / `go test` **115 通过 / 0 失败**；脚本语法与 OpenAPI 语法通过。
  - **浏览器验收**（真实数据 10 分类 / 110 站点）：侧栏贴页面左边缘（left=0）、顶部 65px、宽 212px、
    通高；滚动后固定为 `top=0 / bottom=视口高度`；分类列表可独立滚动；390px 隐藏侧栏改用吸顶胶囊条；
    834px 收为 60px 图标条；标签模式无侧栏、居中容器不变。
  - **发布核验**：CI 与 Release 流水线成功；`ghcr.io/.../v2.1.1` 匿名拉取并运行（`v2.1.1` /
    `schema_version=3` / 非 root），产物含新布局类名；历史镜像摘要未变。
  - **未执行**：生产服务器升级（等待用户确认）、跨浏览器（仅 Chrome）、真机触控与无障碍审计。
- v2.1.0 验证（2026-10-25）：
  - 后端 `go build` / `go vet` / `go test` **115 通过 / 0 失败**；前端 `npm run test` **21 通过 / 0 失败**；
    前端构建、`bash -n`、OpenAPI YAML、Compose 配置（独立 + NPM）全部通过。
  - **部署异常注入**：`scripts/tests/deploy-faults.test.sh` **14 个场景 / 57 个断言全部通过**
    （含"从发布的 v2.0.1 升级，结构 2→3 且数据与手动图标保留"与"只读数据目录快速失败"）。
  - **真实数据兼容**：导入 10 分类 / 110 站点；31 个内嵌 SVG 转本地图标、19 个外链保留、4 个 Simple Icons；
    批量补全 **45 成功 / 8 失败**（失败均带原因），**57 个手动图标全部未被改动**；缓存文件 31 → 76。
  - **浏览器验收**：真实数据 + 真实前后端，覆盖 1440 / 834 / 390 / 375 与深浅主题、两种布局、
    侧栏展开/折叠、后台图标管理；主题与布局互不影响；两种布局数据一致；无 JS 错误。
  - **发布核验**：CI 与 Release 流水线成功；`ghcr.io/xiaowork-dev/xiaowork-nav:v2.1.0`（amd64 + arm64）
    **匿名拉取成功**，运行后 `/healthz` = v2.1.0、`/api/v1/version` = `v2.1.0` / `schema_version=3`、
    容器非 root；`--db-info` 返回 `schema_version=3`；GitHub Release `v2.1.0` 已创建；
    历史镜像 `v2.0.0` / `v2.0.1` 摘要未变。
  - **未执行**：生产服务器升级（等待用户确认）、真实 NPM 反代与 HTTPS 端到端、
    跨浏览器（Firefox/Safari）、真机触控与无障碍审计、长期稳定性与并发。
- 发布验证（2026-10-25，v2.0.1）：
  - 提交 `825f5bc` 已推送；CI（`main` run 40179651451、标签 run 40179801765）与 Release
    流水线（run 40179801770）全部成功。
  - GHCR：`ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.1`（`linux/amd64` + `linux/arm64`），
    摘要 `sha256:4083919b…`；`v2.0` / `latest` 同步指向该摘要。
  - **匿名 `docker pull` 成功**；运行后 `/healthz` = `v2.0.1`、`/api/v1/version` = `v2.0.1` /
    commit `825f5bc` / `schema_version=2`、容器非 root；已发布镜像的 `--db-info` 返回
    `exists=true` / `schema_version=2`。
  - GitHub Release `v2.0.1` 已创建（非 draft / 非 prerelease）。
  - 历史标签 `v2.0.0` **未移动**（仍指向 `d18539d`，镜像摘要仍为 `sha256:c8e2151f…`）。
- Git 信息：远程 `origin` → `https://github.com/xiaowork-dev/xiaowork-nav.git`；
  分支 `main` = `825f5bc`（已推送，与远程一致）；标签 `v2.0.1` 已推送。提交身份为本地占位（待用户替换）。

> 说明：本仓库当前使用本地占位提交身份（`xiaowork-dev <xiaowork-dev@users.noreply.github.com>`），可在配置 GitHub 远程时替换为用户真实身份。
