# V2.1.0 / P1 阶段交付报告：需求调研与架构调整

> 本文件是 **V2.1.0 第 P1 阶段的正式交付文档**，可长期归档。
> 内容基于对当前仓库代码的实际阅读；未验证内容均明确标注。
> 关联：[V2.1.0 实施计划](../../development-plan-v2.1.md)、[AGENTS.md](../../../AGENTS.md)

---

## 1. 阶段目标

在**不破坏现有业务**的前提下，为 V2.1.0 的三个核心功能完成：

1. 现状调研（图标系统、前端布局、数据库、部署脚本）；
2. 架构与接口设计；
3. 必要的数据库迁移设计；
4. 实施计划与阶段划分；
5. 风险与测试策略。

## 2. 完成的工作

| 工作 | 结果 |
| --- | --- |
| 检查 Git 分支 / 工作区 / 远程 | `main` = `51465a1`，与 `origin/main` 一致，工作区干净，标签 `v2.0.0`/`v2.0.1` 存在且未改动 |
| 后端图标系统调研 | 完成（见 §4.1） |
| 前端导航页调研 | 完成（见 §4.2） |
| 数据库与迁移机制调研 | 完成（见 §4.3） |
| 部署脚本与权限问题调研 | 完成（见 §4.4） |
| 架构与接口设计 | 完成，写入 [`docs/development-plan-v2.1.md`](../../development-plan-v2.1.md) |
| 迁移设计 | 完成：`0003_icon_metadata`（只增列，见计划 §3.5） |
| 阶段划分 | 完成：P2 图标 / P3 滚动布局 / P4 标签布局 / P5 集成发布 |

## 3. 新增和修改的文件

| 文件 | 变更 |
| --- | --- |
| `docs/development-plan-v2.1.md` | 新增：V2.1.0 实施计划（调研结论、设计、迁移、接口、测试、风险） |
| `docs/deliveries/v2.1.0/P1-architecture.md` | 新增：本报告 |
| `docs/PROGRESS.md` | 更新：当前阶段与下一步 |

> 本阶段**未修改任何代码**（纯调研与设计），符合"先设计、后实现"的后端优先原则。

## 4. 实现的具体功能（调研结论）

### 4.1 后端图标系统

- `bookmarks.icon_type` 已含 `auto`（"reserved for server-side fetching"），但**当前无效**：
  `validateIcon` 对 `auto` 强制清空 `icon_value`（`service/catalog.go:592`）。
- `service.IconService.Fetch` 已实现"解析 `<link rel=icon>` → 回退 `/favicon.ico` → 落盘 →
  返回 `upload` 引用"，但：无 manifest 解析、无 apple-touch 优先、无质量判定、
  无按来源去重、**不落库**（仅返回给前端）、无来源/失败原因记录。
- 安全已具备：`security.SafeHTTPClient` 在 dialer 的 `Control` 钩子中校验**解析后 IP**
  （可防 DNS 重绑定），CIDR 黑名单含 `169.254.0.0/16`（云元数据），重定向最多 3 跳且逐跳校验 scheme。
  缺口：未拒绝 userinfo、未限制端口、未按 Content-Type 提前放弃、无单次抓取总预算。
- `uploads` 表 + `/data/uploads` + `/api/v1/public/uploads/{filename}`（`immutable` 缓存 + SVG 沙箱 CSP）已可承载缓存；
  **无引用计数**，因此自动清理必须显式按引用判断。
- **无任何异步任务基础设施**（仅有 `RunMaintenance` 定时器）→ 批量补全需新增受控任务管理器。

### 4.2 前端导航页

- `HomePage.vue` 单文件渲染 `AppHeader` + 时钟 + 搜索 + 分类列表；数据来自一次
  `GET /api/v1/public/navigation`（`categories[]` 内嵌 `bookmarks[]`）。
- 主题：`useTheme.ts` 模块级单例 + `localStorage['xn-theme']` + `<html data-theme>`；
  **前端唯一的 localStorage 键**，可直接照此模式实现布局偏好。
- `settings.homepage_layout` 类型为 `'grid' | 'list'`，但**首页从未读取**，实际是"死配置"。
- `IconImage.vue` 已实现显示回退链（用户图标 → `origin/favicon.ico` → 首字母），
  且 `auto` 类型在有值时可直接显示 → 自动图标落盘后可**零改动**复用该组件。
- **前端无测试运行器**，门禁仅为 `vue-tsc --noEmit` + `vite build`。

### 4.3 数据库与迁移

- `//go:embed sql/*.sql`，命名 `NNNN_name.up.sql`，只读 `.up.sql`，事务执行，失败拒绝启动；
  `CheckCompatible` 保证旧镜像不打开新结构库。
- 当前 **max schema version = 2**，下一个为 `0003`。
- 契约由 `api/routes.go` + `contract_test.go` 强制与 `docs/openapi/openapi.yaml` 一致 →
  任何新增路由必须同步契约。

### 4.4 部署脚本与首次部署权限问题

- 镜像以非 root（`uid=100(app)`）运行；`compose.yaml` **未设置 `user:`**。
- 数据目录由宿主机绑定挂载，若宿主机目录属主不是 uid 100（例如以 root 执行 `install.sh`
  创建了 `./data`），容器内 `mkdir /data/uploads` 即报 `permission denied` —— 与线上现象一致。
- 方案（见计划 §6.1）：`user: "${XN_UID:-100}:${XN_GID:-101}"` +
  `install.sh` 优先 `chown 100:101`，无法 chown 时自动把 `XN_UID/XN_GID` 设为当前用户 uid/gid +
  镜像内 entrypoint 做可写性校验并给出可复制的修复命令。**不使用 777，也不长期以 root 运行。**

## 5. 自动化测试结果

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 后端构建 | `go build ./...` | ✅ 通过 |
| 后端静态检查 | `go vet ./...` | ✅ 通过 |
| 后端测试 | `go test -count=1 ./...` | ✅ **75 通过 / 0 失败** |
| 前端类型检查 + 构建 | `npm run build` | ✅ 通过 |
| 脚本语法 | `bash -n scripts/*.sh scripts/tests/*.sh` | ✅ 通过 |
| OpenAPI YAML 语法 | `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过 |

> 说明：以上为 P1 **基线回归**（确认起点健康）。本阶段未新增代码，故未新增测试。

## 6. 浏览器验证结果

本阶段为调研与设计阶段，**未执行浏览器验证**（无界面改动）。真实浏览器验收安排在 P2–P5。

## 7. 未执行或未通过的测试

| 项 | 原因 |
| --- | --- |
| 前端单元测试 | 项目当前无测试运行器；计划在 P2 引入 Vitest（见计划 §8.2） |
| 浏览器截图验收 | 无界面改动，安排在后续阶段 |
| Docker 部署 / 升级 / 回滚实测 | 安排在 P2（图标）与 P5（升级回滚） |
| 真实线上环境验证 | **明确不做**：需求禁止直接操作生产环境 |

## 8. 当前项目进度

| 阶段 | 状态 |
| --- | --- |
| v2.0.0 / v2.0.1 发布 | ✅ 已完成 |
| **V2.1.0 / P1 调研与架构** | ✅ **本阶段完成** |
| V2.1.0 / P2 图标自动获取 | ⏳ 待开始 |
| V2.1.0 / P3 滚动布局增强 | ⏳ 待开始 |
| V2.1.0 / P4 标签布局与切换 | ⏳ 待开始 |
| V2.1.0 / P5 集成测试与发布 | ⏳ 待开始 |

## 9. Git Commit Hash 与推送状态

| 项 | 内容 |
| --- | --- |
| 本阶段起点 | `51465a1`（`docs(p5): record the v2.0.1 release and verification`） |
| 本阶段提交 | 见聊天总结与 `git log`（本文件不硬写自身提交 SHA） |
| 推送 | 本阶段文档与代码一并推送 `origin/main` |
| 标签 | 未创建新标签（v2.1.0 在 P5 发布） |

## 10. 遗留问题

| 问题 | 影响 | 计划 |
| --- | --- | --- |
| 前端无测试运行器 | 回归依赖人工 | P2 引入 Vitest（纯逻辑） |
| `homepage_layout` 旧值 `grid/list` | 语义与新布局不一致 | 读取时归一化（`grid→scroll`、`list→tabs`），不重写历史数据 |
| `auto` 图标类型历史为空值 | 旧数据无自动图标 | 批量补全按需生成，不影响旧数据 |
| 线上数据目录属主问题 | 新用户首次部署可能失败 | P5 修复 install 脚本 + entrypoint 校验 |
| 批量任务状态不落库 | 重启后进度丢失 | 明确取舍并文档化；前端可重新发起 |

## 11. 下一阶段工作（V2.1-P2）

1. 迁移 `0003_icon_metadata`（bookmarks 4 列 + uploads 2 列 + 索引）。
2. `security.ValidateFetchURL`（userinfo / 端口限制）与抓取总预算。
3. 图标发现与质量判定：`link rel` 排序、apple-touch、manifest、favicon.ico、品牌推测。
4. 缓存去重与安全清理（按 `source_url` 复用；仅无引用时删除 auto 文件）。
5. `IconJobManager` 异步批量（单任务、3 并发、限频、进度、取消）。
6. 新增/变更 5 个管理接口并同步 OpenAPI 契约与权限。
7. 后台图标管理 UI（单站自动获取/重新获取、批量补全、来源、预览、失败原因）。
8. 后端测试（解析、优先级、失败、缓存、批量、SSRF、迁移）与前端 Vitest 接入。
9. 阶段交付文档 `deliveries/v2.1.0/P2-icons.md`，更新 `docs/PROGRESS.md`，提交并推送。
