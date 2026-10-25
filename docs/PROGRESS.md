# 项目进度总览

> 最后更新：2026-10-25 08:00（Asia/Shanghai）
> 项目：XiaoWork Nav V2.0（版本 **2.0.0**）
> 总体状态：**已发布** —— 仓库已开源、CI 通过、多架构镜像已推送 GHCR、GitHub Release 已创建
>
> 交付规范见 [`../AGENTS.md`](../AGENTS.md)：每阶段必须产出 `docs/deliveries/P<N>-delivery.md`。

## 项目目标与当前阶段

- 核心目标：将旧版个人导航站重构为轻量、可自托管、可公开开源、支持一键部署与一键更新的产品。
- 当前阶段：**P5 Docker 部署、GitHub 开源与版本发布** —— ✅ 已完成。
- 本轮里程碑：仓库创建并推送、CI 全绿、多架构（amd64/arm64）镜像发布到 GHCR 且匿名可拉取、GitHub Release `v2.0.0` 发布；发布过程中修复了 QEMU 构建过慢与镜像标签缺少 `v` 前缀两个问题。

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

## 正在进行 / 未完成

| 工作项 | 现状 | 下一动作 |
| --- | --- | --- |
| 独立测试服务器真实部署 + NPM 反代 HTTPS | 未验证 | 在具备 NPM 与域名的环境完成 |
| GHCR 遗留标签 `2.0.0` / `2.0` 清理 | 可选 | 包管理页删除摘要 `cf6040d5…` 的版本（P5 报告 §9.3） |
| 替换占位提交身份 | 待用户提供 | 提供真实 Git 身份后更新 |
| 跨浏览器 / 真机触控 / 无障碍审计 | 未验证 | 后续版本 |
| OpenAPI 规范级 lint / 前端类型自动生成 | 未接入 | 后续接入 CI |
| 旧站真实数据的正式导入 | 已可在容器中导入 | 剩余 23 个占位图标见迁移报告 §13.5 |

## 当前阻塞与风险

- **无 NPM 实例与公网域名**：反代与 HTTPS 未实测。
- **受限网络下 Go 模块代理不可达**：构建镜像需传 `--build-arg GOPROXY=...`（已文档化）。
- **提交身份为占位**：待用户替换。
- **部分站点图标依赖第三方**：23 个站点显示首字母、4 个第三方 Favicon 内容透明。

## 下一步（按优先级）

1. 在独立测试服务器完成真实部署与 NPM 反代 HTTPS 验证。
2. 可选：清理 GHCR 遗留标签、替换提交身份。
3. 规划下一版本增强项（`docs/requirements.md` §2）。

## 最近交付记录

- **[P5 阶段交付报告](deliveries/P5-delivery.md)**（正式归档）
- **[P4 阶段交付报告](deliveries/P4-delivery.md)**（正式归档）
- **[P3 阶段交付报告](deliveries/P3-delivery.md)**（正式归档）
- [旧站数据迁移报告（含 §13 P3 收尾补充）](legacy-data-migration-report.md)
- **[P2 阶段交付报告](deliveries/P2-delivery.md)**（正式归档）
- [P1 阶段交付报告](deliveries/P1-delivery.md)（正式归档）

## 验证与环境说明

- 最近一次验证（2026-10-25 08:00）：
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
- Git 信息：远程 `origin` → `https://github.com/xiaowork-dev/xiaowork-nav.git`；
  分支 `main` 已推送且与远程一致；标签 `v2.0.0` 已推送。提交身份为本地占位（待用户替换）。

> 说明：本仓库当前使用本地占位提交身份（`xiaowork-dev <xiaowork-dev@users.noreply.github.com>`），可在配置 GitHub 远程时替换为用户真实身份。
