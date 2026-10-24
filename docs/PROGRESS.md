# 项目进度总览

> 最后更新：2026-10-24 17:53（Asia/Shanghai）
> 项目：XiaoWork Nav V2.0
> 总体状态：开发中（P2 完成：Go 后端核心业务；前端页面尚未开始）
>
> 交付规范见 [`../AGENTS.md`](../AGENTS.md)：每阶段必须产出 `docs/deliveries/P<N>-delivery.md`。

## 项目目标与当前阶段

- 核心目标：将旧版个人导航站重构为轻量、可自托管、可公开开源、支持一键部署与一键更新的产品。
- 当前阶段：**P2 Go 后端核心业务开发** —— 已完成。
- 本轮里程碑：认证与会话、分类与导航项 CRUD、排序、搜索、系统设置、图标、导入导出、备份恢复、统一校验与安全机制，配套 51 个后端测试。

## 已完成（仅列已经验证的成果）

| 成果 | 状态 | 验证依据 | 交付记录 |
| --- | --- | --- | --- |
| 需求/契约/数据库设计（P1） | 已完成 | `docs/requirements.md` 等 | [P1 报告](deliveries/P1-delivery.md) |
| 项目初始化与后端基础架构（P1） | 已完成 | `go test` + 运行时冒烟 | [P1 报告](deliveries/P1-delivery.md) |
| 认证与会话（初始化/登录/登出/改密/限流） | 已完成 | 接口测试 + 服务测试 + 二进制冒烟 | [P2 报告](deliveries/P2-delivery.md) |
| 权限边界（管理接口需登录） | 已完成 | 16 个管理端点逐一断言 401 | [P2 报告](deliveries/P2-delivery.md) |
| 分类 CRUD / 排序 / 可见性 | 已完成 | 接口测试 | [P2 报告](deliveries/P2-delivery.md) |
| 导航项 CRUD / 排序 / 移动 / URL 校验 | 已完成 | 接口测试 | [P2 报告](deliveries/P2-delivery.md) |
| 公开导航与搜索的可见性隔离 | 已完成 | 接口测试 | [P2 报告](deliveries/P2-delivery.md) |
| 系统设置（公开/管理视图、校验） | 已完成 | 接口测试 + 服务测试 | [P2 报告](deliveries/P2-delivery.md) |
| 图标上传与抓取（防 SSRF） | 已完成 | 接口测试 + 安全测试 | [P2 报告](deliveries/P2-delivery.md) |
| 导入导出（原生 + legacy） | 已完成 | 接口测试 | [P2 报告](deliveries/P2-delivery.md) |
| 备份与恢复 | 已完成 | 接口测试 | [P2 报告](deliveries/P2-delivery.md) |
| 统一错误处理与 CSRF/Origin 校验 | 已完成 | 接口测试 + 二进制冒烟 | [P2 报告](deliveries/P2-delivery.md) |
| Docker 部署工件（P1） | 部分完成 | `docker compose config` 通过；**镜像构建未执行** | [P1 报告](deliveries/P1-delivery.md) |

## 正在进行 / 未完成

| 工作项 | 现状 | 下一动作 |
| --- | --- | --- |
| 前端页面（首页/后台/管理） | 未开始 | P3 实现 |
| UI 与主题完善 | 未开始 | P4 |
| Docker 镜像构建实测 | 未执行 | P5 在具备 Docker 的环境构建并验证 |
| GitHub 远程仓库与推送 | 未创建 | 基础稳定后配置远程并推送 |
| 旧站真实数据导入验证 | 待样本 | 取得 `data.json` 后补充映射与测试 |
| OpenAPI 规范级 lint | 未接入 | P5/CI 引入 `redocly`/`spectral` |

## 当前阻塞与风险

- **Docker 守护进程不可用**：镜像构建与容器启动未验证。
- **旧站 `data.json` 未提供**：`legacy-v1` 导入器仅按通用映射实现，未经真实数据验证。
- **登录限流可被用于短时锁定管理员**：连续失败后按用户名限流约 5 分钟（已知取舍）。
- **GitHub 仓库信息未定**：远程未创建，暂不能推送。

## 下一步（按优先级）

1. 进入 P3：实现导航首页、初始化/登录、管理后台（分类/网站/设置）、响应式布局。
2. P3 使用 `openapi-typescript` 生成类型，确保契约一致。
3. 提供旧站 `data.json` 样本以完成并验证 legacy 导入。
4. 确认 GitHub 仓库名后配置远程并推送。
5. P5 在具备 Docker 的环境完成镜像构建与一键部署实测。

## 最近交付记录

- **[P2 阶段交付报告](deliveries/P2-delivery.md)**（正式归档）
- [P1 阶段交付报告](deliveries/P1-delivery.md)（正式归档）
- [2026-10-24 P1 项目初始化与后端基础架构](deliveries/20261024-170200-p1-init-backend-skeleton.md)（过程记录）
- [2026-10-24 架构与开发计划调整](deliveries/20261024-163148-v2-planning-adjustment.md)（过程记录）

## 验证与环境说明

- 最近一次验证（2026-10-24 17:53）：
  - `go build ./...`、`go vet ./...` 通过（backend）。
  - `go test -count=1 ./...` 通过：**51 个测试通过 / 0 失败**（api/auth/config/migrations/security/service/store）。
  - 真实二进制冒烟：初始化→登录→建分类/导航项→公开导航/搜索→备份→导出→登出→401/403 全部符合预期。
  - `npx js-yaml docs/openapi/openapi.yaml` 通过；`docker compose config` 两套配置通过。
  - **Docker 镜像构建未执行**（守护进程不可用）。
- 部署环境：无（未部署服务器；未创建 GitHub 远程）。
- Git 信息：本地仓库 `master` 分支；远程未配置。提交身份为本仓库本地占位（待用户替换）。

> 说明：本仓库当前使用本地占位提交身份（`xiaowork-dev <xiaowork-dev@users.noreply.github.com>`），可在配置 GitHub 远程时替换为用户真实身份。
