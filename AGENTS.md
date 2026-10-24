# AGENTS.md — XiaoWork Nav 开发与交付规范

> 本文件是**项目级 Agent 指令**。任何 AI Agent（Codex、Claude Code、Cursor、其他支持 `AGENTS.md` 的工具）在本仓库工作前必须阅读并遵守。
> 适用范围：**P1、P2、P3 及后续所有阶段**，不限于当前阶段。

---

## 0. 项目速览

- 项目：XiaoWork Nav V2.0 —— 轻量、可自托管、可开源的个人导航站。
- 技术栈：Vue 3 + TypeScript + Vite（前端）/ Go 标准库（后端）/ SQLite（数据）/ Docker Compose（部署）/ GitHub Actions + GHCR（CI/CD）。
- 架构原则：**后端优先、API 驱动、严格前后端分离**；OpenAPI 为唯一契约。
- 关键文档入口：[`docs/README.md`](docs/README.md)、[`docs/development-plan-v2.md`](docs/development-plan-v2.md)、[`docs/architecture.md`](docs/architecture.md)、[`docs/api-spec.md`](docs/api-spec.md)、[`docs/PROGRESS.md`](docs/PROGRESS.md)。

---

## 1. 交付规范（强制）

### 1.1 每个阶段结束必须产出阶段交付报告

每完成一个开发阶段（P1、P2、P3、P4、P5 及后续任何阶段），**必须**在项目目录中生成一份独立、完整、可长期归档的 Markdown 阶段交付报告：

```text
docs/deliveries/P<N>-delivery.md
```

- 该文件是**阶段交付的正式档案**，不是聊天总结，也不能被其他技术文档替代。
- 阶段持续演进时**更新同一份文件**（例如 P2 分多次提交，仍更新 `P2-delivery.md`），保持其内容为"该阶段最终真实状态"。
- 过程中可另建增量记录 `docs/deliveries/YYYYMMDD-HHMMSS-<slug>.md`，但**阶段报告是必需项**。

### 1.2 阶段交付报告必须包含的章节

1. **阶段基本信息**：项目名称与版本、阶段名称与目标、开始/完成日期、当前完成状态。
2. **本阶段工作总结**：实际完成的任务、已实现功能、核心技术方案与架构决策、与原计划的差异及原因。
3. **代码和文件变更**：新增/修改的重要文件、各模块实际完成情况、重要目录结构说明。
4. **测试与验证结果**：已执行测试及实际结果、未通过的测试及原因、未执行的测试及原因、当前环境限制。
5. **项目总体进度**：阶段列表、已完成阶段、当前阶段、未开始阶段及剩余任务。
6. **Git 版本记录**：当前分支、最新提交 Hash、工作区状态、是否配置远程、是否已推送。
7. **遗留问题和风险**：未完成任务、已知问题、尚未验证的功能、进入下一阶段的必要条件。
8. **下一阶段计划**：目标、具体任务、预期交付成果、验收标准。

### 1.3 交付报告的真实性红线（严禁违反）

- **禁止**把未实现、未执行、未验证的内容写成"已完成/已通过"。
- 未执行的测试必须明确标注"**未执行**"并说明原因（如环境限制）。
- 未通过的测试必须如实记录现象与复现方式。
- 报告中的文件、行数、命令与结果必须与仓库实际状态一致。
- **聊天窗口中的总结不能代替交付文档**；每次阶段结束都必须有对应文件。

### 1.4 阶段结束的标准动作

1. 运行该阶段可执行的测试/构建/静态检查，记录真实输出。
2. 编写或更新 `docs/deliveries/P<N>-delivery.md`。
3. 更新 [`docs/PROGRESS.md`](docs/PROGRESS.md)：当前阶段、已完成、进行中、阻塞、下一步、最新交付链接、验证摘要。
4. 如有重要架构决策，新增 `docs/decisions/ADR-NNNN-*.md`。
5. 将**代码、必要配置与交付文档纳入同一次 Git 提交**。
6. 在聊天中给出**简短总结**，并明确写出交付报告的**准确文件路径**、当前总体进度、未完成项与下一步。

---

## 2. 架构与代码约束

1. **后端优先**：接口先冻结再实现；前端不先于其依赖的后端接口实现。
2. **严格前后端分离**：前端只通过 `/api/v1` 交互；前端不得访问数据库或依赖后端内部结构；后端不得依赖前端框架。
3. **OpenAPI 唯一契约**：接口变更先改 [`docs/openapi/openapi.yaml`](docs/openapi/openapi.yaml)，再改后端实现与前端类型；破坏性变更升级 API 版本并记 ADR。
4. **后端分层**：`api`（HTTP）→ `service`（业务）→ `store`（数据），禁止反向依赖；业务层不写裸 SQL。
5. **前端**：所有 HTTP 请求集中在 `frontend/src/api/`；组件/页面不得直接 `fetch`。
6. **数据库迁移**：版本化 SQL，`只增不改`；失败必须拒绝启动；不可逆迁移依赖升级前快照。
7. **安全底线**：密码 Argon2id；Cookie `HttpOnly`/`SameSite`/生产 `Secure`；状态变更校验 Origin/CSRF；URL 协议白名单；图标抓取防 SSRF；上传校验类型与体积；容器非 root。

---

## 3. Git 与敏感信息规范

- 提交信息使用祈使句，可加 `feat:`/`fix:`/`docs:`/`chore:` 前缀。
- **严禁提交** `.env`、密钥、令牌、真实数据文件、数据库文件。
- 本地提交随时可进行；**配置 GitHub 远程后，每个正式版本必须提交并推送**。
- 若无权限或远程不可用导致推送失败，必须在交付报告中明确写"未推送成功"，**不得假称已同步**。
- 不要在本仓库待提交的文件中硬写"当前提交 SHA / 已推送成功"；提交 SHA 与推送结果在聊天总结中以 Git 事实为准。

---

## 4. 验证命令（提交前应执行）

```bash
# 后端
cd backend && go build ./... && go vet ./... && go test ./... && cd ..

# 前端
cd frontend && npm run build && cd ..

# OpenAPI 语法
npx js-yaml docs/openapi/openapi.yaml > /dev/null

# 部署工件（需 Docker CLI）
docker compose -f deploy/compose.yaml --env-file deploy/.env.example config --quiet
```

无法执行的检查必须在交付报告中标注"未执行"及原因。

---

## 5. 阶段与目录约定

| 阶段 | 名称 | 交付报告 |
| --- | --- | --- |
| P1 | 需求梳理、契约冻结与项目初始化 | `docs/deliveries/P1-delivery.md` |
| P2 | Go 后端完整实现 | `docs/deliveries/P2-delivery.md` |
| P3 | Vue 3 前端完整实现 | `docs/deliveries/P3-delivery.md` |
| P4 | UI、交互与主题完善 | `docs/deliveries/P4-delivery.md` |
| P5 | 部署、更新与发布 | `docs/deliveries/P5-delivery.md` |

```text
backend/    Go 服务（cmd/server + internal/{api,config,store,migrations,static,version}）
frontend/   Vue 3 前端（src/api 为唯一 HTTP 出口）
deploy/     Compose 与环境变量模板
scripts/    部署与更新脚本
docs/       架构 / API / 数据库 / 部署 / 决策 / 进度 / 交付
```

---

## 6. 完成判据

一个阶段只有同时满足以下条件才算完成：

1. 该阶段交付报告 `docs/deliveries/P<N>-delivery.md` 存在且内容与代码、测试、Git 状态一致。
2. `docs/PROGRESS.md` 已同步更新。
3. 已运行该阶段允许的测试并如实记录结果。
4. 交付文档与代码在同一提交中纳入 Git 版本管理。
5. 聊天总结中给出了交付报告路径、总体进度、未完成项与下一步。
