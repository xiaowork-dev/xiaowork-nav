# 项目进度总览

> 最后更新：2026-10-25 19:40（Asia/Shanghai）
> 项目：XiaoWork Nav V2.0（版本 **2.0.1**；**V2.1.0 开发中**）
> 总体状态：**V2.1.0 开发中** —— v2.0.1 已发布；V2.1.0 完成 P1（调研与架构）、P2（官方图标自动获取），进入 P3（滚动布局侧栏）
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

## 正在进行 / 未完成

| 工作项 | 现状 | 下一动作 |
| --- | --- | --- |
| **V2.1.0 / P1 调研与架构** | ✅ 已完成 | 见 [V2.1.0 实施计划](development-plan-v2.1.md) |
| **V2.1.0 / P2 官方图标自动获取** | ✅ 已完成 | 迁移 0003、抓取与缓存、SSRF 加固、异步批量、后台 UI、115 个后端测试、浏览器实测 19/19 成功 |
| **V2.1.0 / P3 滚动布局左侧分类栏** | ✅ 已完成 | 侧栏组件、点击定位、滚动高亮（11 个单元测试）、折叠与三档响应式、浏览器实测 |
| **V2.1.0 / P4 标签布局与一键切换** | ⏳ 进行中 | 标签布局、布局切换控件、偏好持久化、主题与布局独立 |
| **V2.1.0 / P5 集成测试与发布** | ⏳ 未开始 | 全量回归 + 真实浏览器验收 + 升级/回滚 + 发布 v2.1.0 |
| 独立测试服务器真实部署 + NPM 反代 HTTPS | 未验证 | 在具备 NPM 与域名的环境完成 |
| GHCR 遗留标签 `2.0.0` / `2.0` 清理 | 可选 | 包管理页删除摘要 `cf6040d5…` 的版本（P5 报告 §9.3） |
| 替换占位提交身份 | 待用户提供 | 提供真实 Git 身份后更新 |
| 跨浏览器 / 真机触控 / 无障碍审计 | 未验证 | 后续版本 |
| OpenAPI 规范级 lint / 前端类型自动生成 | 未接入 | 后续接入 CI |
| 旧站真实数据的正式导入 | 已可在容器中导入 | 剩余 23 个占位图标见迁移报告 §13.5 |
| 用已发布镜像做一次真实 `latest` 升级验证 | 未执行 | 在测试机执行 `./scripts/update.sh latest` |
| 真实 NPM 反代 HTTPS 端到端 | 未验证 | 在具备 NPM 与域名的环境完成 |

## 当前阻塞与风险

- **无 NPM 实例与公网域名**：反代与 HTTPS 未实测。
- **受限网络下 Go 模块代理不可达**：构建镜像需传 `--build-arg GOPROXY=...`（已文档化）。
- **提交身份为占位**：待用户替换。
- **部分站点图标依赖第三方**：23 个站点显示首字母、4 个第三方 Favicon 内容透明。
- **异常注入测试需要 Docker 与可用的 Go 模块代理**：构建测试镜像时可用 `XN_TEST_GOPROXY` 指定镜像源。

## 下一步（按优先级）

1. 用已发布的 `v2.0.1` 镜像做一次真实的 `latest` 与固定版本升级验证。
2. 在独立测试服务器完成真实部署与 NPM 反代 HTTPS 验证。
3. 可选：清理 GHCR 遗留标签、替换提交身份。
4. 规划下一版本增强项（`docs/requirements.md` §2）。

## 最近交付记录

- **[V2.1.0 实施计划](development-plan-v2.1.md)**（当前版本计划）
- **[V2.1.0 / P3 阶段交付报告：PC 端滚动布局左侧分类栏](deliveries/v2.1.0/P3-scroll-layout.md)**
- **[V2.1.0 / P2 阶段交付报告：官方图标自动获取](deliveries/v2.1.0/P2-icons.md)**
- **[V2.1.0 / P1 阶段交付报告](deliveries/v2.1.0/P1-architecture.md)**
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
