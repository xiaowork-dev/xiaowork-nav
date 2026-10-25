# P5 补充交付记录：一键更新与数据库回滚缺陷修复（v2.0.1）

> 本文是 **P5 阶段的增量交付记录**，用于归档 P5 验收复审后对"一键更新 / 数据库回滚"
> 的缺陷修复。P5 阶段正式报告仍为 [`P5-delivery.md`](P5-delivery.md)（本记录的内容
> 也已摘要并入该报告 §12）。
>
> 内容基于实际代码、真实 Docker 异常注入测试与 Git 状态编写；未执行或未验证的内容
> 均明确标注。

---

## 1. 基本信息

| 项 | 内容 |
| --- | --- |
| 项目名称 | XiaoWork Nav |
| 项目版本 | **V2.0.1**（后端 `v2.0.1`；数据库结构版本仍为 **2**，无结构变更） |
| 阶段 | **P5 补充**（Docker 部署与运维修复） |
| 目标 | 修复 P5 验收复审发现的一键更新与数据库回滚缺陷，并补充真实验证 |
| 开始日期 | 2026-10-25 |
| 完成日期 | 2026-10-25 |
| 完成状态 | 代码、脚本与文档已完成；后端 `75/75` 通过；**异常注入 12 场景 / 43 断言全部通过**；待提交、推送并发布 v2.0.1 |
| 上位依据 | `AGENTS.md`、[`docs/updating.md`](../updating.md)、[`docs/deployment.md`](../deployment.md) |

---

## 2. 本阶段工作总结

P5 验收结论：Docker 部署、GitHub CI/CD、GHCR 发布与 Release 通过验收；
但**一键更新与数据库回滚**存在缺陷。本记录逐条对应修复。

### 2.1 缺陷一：`latest` 更新逻辑

| 项 | 内容 |
| --- | --- |
| 现象 | `update.sh` 在 `TARGET_VERSION == CURRENT_VERSION` 时直接退出。当两者都是 `latest` 时，后续更新永远不会重新拉取镜像。 |
| 修复 | 可变标签（`latest`、`v2.0` 等）**总是查询并拉取**镜像；固定语义化版本（`vX.Y.Z`）仍为幂等直接退出。 |
| 细节 | 拉取后若镜像摘要未变、且运行中的容器正是该镜像且健康 → 明确报告"无需操作"并**不重建容器**；摘要变化或容器状态不符 → 正常执行快照、切换、健康检查。 |
| 判定规则 | 新增 `is_mutable_tag()`：`latest` 或非 `vX.Y.Z` 形式视为可变；完整 `vX.Y.Z` 视为不可变。 |

### 2.2 缺陷二：NPM 模式的数据库版本检测

| 项 | 内容 |
| --- | --- |
| 现象 | `app_schema_version()` 依赖宿主机 `127.0.0.1:<端口>`；NPM Compose 模式**不发布宿主机端口**，检测必然失败。 |
| 修复 | 新增应用内命令行模式 `--db-info`，**直接读取数据库文件**；结构版本与行数由此获得，不再依赖任何宿主机端口。 |
| 输出 | `db_path` / `exists` / `schema_version` / `categories` / `bookmarks` / `uploads`（`key=value`，无需 `jq`）。 |
| 只读保证 | `--db-info` 只执行 `SELECT`（`store.OpenForInspection` + `migrations.ReadVersion`），不会建表或写入。 |
| 额外收益 | 应用**已停止**（更新失败后）同样可读，因此回滚前可以正确判断结构兼容性。 |
| 安全 | 检测过程不暴露任何端口；NPM 模式仍完全不发布宿主机端口。 |

### 2.3 缺陷三：更新失败后的自动回滚

覆盖要求中列出的全部六类场景：

| # | 场景 | 处理方式 |
| --- | --- | --- |
| 1 | 新镜像拉取失败 | 本地已有该镜像则回退使用本地副本并继续；否则**不改动任何东西**直接退出 |
| 2 | 新容器启动失败 | 回滚 |
| 3 | 新容器健康检查失败 | 回滚 |
| 4 | 已迁移数据库但随后启动/健康检查失败 | **先用更新前快照恢复数据**，再切回旧镜像 |
| 5 | 无法获取数据库结构版本（文件读取也失败） | 视为"兼容性未知"，**先恢复快照**再回滚（保守策略） |
| 6 | NPM 共享网络模式升级失败 | 与独立模式完全一致（检测不依赖端口） |

另外：结构版本**倒退**（新 < 旧）也会触发回滚。

**回滚后再次验证**（不再只看 `docker compose up -d` 的退出码）：

1. 重新等待健康检查（`healthy`/`running`）；
2. 核对结构版本是否回到更新前的值；
3. 核对书签行数是否与更新前一致；
4. 打印更新前快照路径供人工介入。

> 关键设计：**无法可靠判断数据库兼容性时，绝不盲目用旧镜像启动新数据库**。
> 判定为"结构变大 / 无法读取 / 更新前未知"时一律先恢复快照。

### 2.4 缺陷四：手动恢复流程 `restore.sh`

| 项 | 内容 |
| --- | --- |
| 现象 | 恢复数据后直接启动当前配置的镜像；若当前镜像比快照新，会立刻对刚恢复的旧库重新执行迁移。 |
| 修复 | 恢复数据后、**启动之前**，先把 `APP_VERSION` 切换到快照所属版本。 |
| 版本来源 | 优先 `--version vX.Y.Z`；否则从快照文件名推断（`backup-<版本>-<时间戳>.tar.gz`、`pre-update-<版本>-<时间戳>.tar.gz`）。 |
| 安全兜底 | 版本无法确定时**只恢复数据、保持应用停止**，并打印手动启动指引，**不猜测**。 |
| 可反悔 | 恢复前自动对当前数据再做一份安全快照。 |
| 新增参数 | `--version`、`--no-start`、`--compose`、`--yes`。 |

### 2.5 缺陷五：真实验证

新增可提交的异常注入测试 `scripts/tests/deploy-faults.test.sh`，在临时目录搭建隔离部署，
构建一次性镜像（含一个额外迁移的镜像与一个"无法启动"的镜像），覆盖 12 个场景（见 §4.2）。

---

## 3. 代码和文件变更

### 3.1 新增

| 文件 | 作用 |
| --- | --- |
| `backend/internal/migrations/readversion_test.go` | `ReadVersion` 的只读/非破坏性测试 |
| `backend/cmd/server/main_test.go` | `--db-info` 参数解析与输出测试 |
| `scripts/tests/deploy-faults.test.sh` | 部署脚本异常注入测试（12 场景） |
| `docs/decisions/ADR-0005-db-info-inspection.md` | 决策记录：用 `--db-info` 直接读库判断结构版本 |
| `docs/deliveries/20261025-170154-p5-update-rollback-fix.md` | 本记录 |

### 3.2 修改

| 文件 | 变更 |
| --- | --- |
| `backend/cmd/server/main.go` | 新增 `--db-info` 模式与 `hasFlag` / `flagValue` / `countRows` 辅助函数 |
| `backend/internal/migrations/migrations.go` | 新增 `ReadVersion`（不写库、表不存在时返回 0） |
| `backend/internal/store/store.go` | 新增 `OpenForInspection`（只读用途的打开方式） |
| `scripts/lib.sh` | 新增 `image_repo` / `image_ref_for` / `image_ref` / `local_image_id` / `container_image_id` / `is_mutable_tag` / `db_info` / `db_field` / `schema_version` / `snapshot_dir` / `restore_snapshot` / `version_from_snapshot`；`db_info` 兼容 Windows Git Bash 的 MSYS 路径转换；移除基于宿主机端口的检测路径 |
| `scripts/update.sh` | 重写更新判定（可变/固定标签）、拉取、快照、回滚判定与回滚后验证 |
| `scripts/restore.sh` | 重写为"版本推断/指定 → 切换版本 → 启动"，版本未知时拒绝启动 |
| `docs/updating.md` | 新增 §1.2 结构版本读取、§4.3 回滚保证、§6 异常注入测试；更新恢复流程 |
| `docs/deployment.md` | 更新脚本清单与关键设计约束 |
| `docs/installation.md` | 新增排障条目（restore 拒绝启动、NPM 下读结构版本） |
| `README.md` | 更新"Updating"小节与恢复说明 |
| `CHANGELOG.md` | 新增 `[2.0.1]` 条目 |
| `.github/workflows/ci.yml` | shellcheck 同时检查 `scripts/tests/*.sh` |
| `.gitignore` | 忽略测试用的 `deploy/*.local.yaml` |

---

## 4. 测试与验证结果

环境：Windows 11 + Docker Desktop，Docker **29.8.2**、Compose **v5.6.0**（linux/amd64）。
所有测试使用**本地构建的真实镜像**与**临时数据目录**；未触碰任何现有服务器数据。

### 4.1 自动化测试

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 后端构建 | `go build ./...` | ✅ 通过 |
| 后端静态检查 | `go vet ./...` | ✅ 通过 |
| 后端测试 | `go test -count=1 ./...` | ✅ **75 通过 / 0 失败**（新增 4 个用例） |
| 前端类型检查 + 构建 | `npm run build` | ✅ 通过 |
| 脚本语法 | `bash -n scripts/*.sh scripts/tests/*.sh` | ✅ 通过 |
| Compose 配置（独立模式） | `docker compose -f deploy/compose.yaml --env-file deploy/.env.example config --quiet` | ✅ 通过 |
| Compose 配置（NPM 模式） | 同上，`compose.npm.yaml`，并设置 `XN_PROXY_NETWORK` | ✅ 通过（未设置该变量时按设计报错，属预期） |
| OpenAPI 语法 | `npx js-yaml docs/openapi/openapi.yaml` | ✅ 通过（本阶段未改契约） |

### 4.2 真实 Docker 异常注入测试

命令：

```bash
XN_TEST_GOPROXY=https://goproxy.cn,direct ./scripts/tests/deploy-faults.test.sh
```

结果：**12 个场景 / 43 个断言全部通过，0 失败**（脚本末尾输出
`All deployment fault-injection tests passed.`）。

| # | 场景 | 关键断言 | 结果 |
| --- | --- | --- | --- |
| 1 | 应用运行时读取数据库文件 | `exists=true`、`schema_version=2` | ✅ |
| 2 | 固定版本已部署 | 输出"already on the pinned version" | ✅ |
| 3 | `latest` 镜像未变化 | 报告"无需操作"且**容器 ID 不变** | ✅ |
| 4 | 目标镜像不可用 | 拒绝继续，容器与数据未变，应用仍可访问 | ✅ |
| 5 | 镜像拉取失败（本地无副本） | 报告拉取失败，容器与 `APP_VERSION` 未变 | ✅ |
| 6 | 容器启动失败（结构未变） | 检测失败 → 回滚 → 验证；结构版本与数据不变 | ✅ |
| 7 | 迁移后健康检查失败 | 检测到"结构已推进" → **先恢复快照** → 回滚并验证结构/书签数回到更新前 | ✅ |
| 8 | 新镜像无法报告结构版本 | 走保守路径（"cannot determine the database schema version"）→ 恢复快照 → 回滚并验证 | ✅ |
| 9 | 应用**停止**时读取结构版本 | 停止容器后仍读到 `schema_version=2` 与行数（NPM 模式无宿主机端口） | ✅ |
| 10 | `restore.sh` 指定/推断版本 | 启动前切换 `APP_VERSION` 到快照版本，恢复后数据完整 | ✅ |
| 11 | `restore.sh` 版本未知 | 输出"refusing to start"，保持应用停止 | ✅ |
| 12 | NPM 模式（无宿主机端口）升级失败 | 检测迁移 → 恢复快照 → 回滚，回滚后容器 `healthy` | ✅ |

### 4.3 未执行 / 未验证

| 项 | 原因 |
| --- | --- |
| 真实 Nginx Proxy Manager 反代与 HTTPS 端到端 | 环境无 NPM 实例与公网域名；NPM 模式仅以"不发布宿主机端口 + 外部网络"的等价编排验证 |
| 真实 GHCR 拉取升级（`docker pull` 真实新版本） | 发布 v2.0.1 前，`v2.0.1` 镜像尚不存在；脚本的拉取路径已用"不可达仓库"与"本地镜像"两条路径覆盖 |
| 长期运行与并发压力 | 未做 |
| Linux 原生环境运行测试脚本 | 本机为 Windows + Docker Desktop；脚本已对 MSYS 路径转换做兼容（见 `db_info`） |

> **声明：本记录未将任何未执行的测试标记为已通过。**

---

## 5. 项目总体进度

| 阶段 | 名称 | 状态 |
| --- | --- | --- |
| P1 | 需求梳理、契约冻结与项目初始化 | ✅ 已完成 |
| P2 | Go 后端完整实现 | ✅ 已完成 |
| P3 | Vue 3 前端完整实现 | ✅ 已完成 |
| P4 | UI、交互与主题完善 | ✅ 已完成 |
| P5 | 部署、更新与发布 | ✅ 已完成（v2.0.0 已发布） |
| **P5 补充** | **一键更新与回滚缺陷修复（v2.0.1）** | ✅ 修复完成并通过真实验证；待发布 v2.0.1 |

---

## 6. Git 版本记录

| 项 | 内容 |
| --- | --- |
| 当前分支 | `main` |
| 本次提交前的最近提交 | `c1cadd4` — `docs(p5): record the GitHub publication results` |
| 本次提交内容 | 上述代码、脚本、测试、文档（同一提交纳入版本管理） |
| 远程 | `origin` → `https://github.com/xiaowork-dev/xiaowork-nav.git` |
| 推送状态 | 见聊天总结与 `git log`（本文件不硬写提交 SHA 与推送结果） |
| 历史标签 | `v2.0.0` **保持不变**（未移动、未重建） |

---

## 7. 遗留问题和风险

| 问题 | 影响 | 处理 |
| --- | --- | --- |
| 真实 NPM 反代 + HTTPS 端到端未实测 | 反代链路未验证 | 在具备 NPM 与域名的环境完成 |
| `--db-info` 需本地存在目标镜像 | 无本地镜像时结构版本读取失败 | 按"未知"走保守回滚；更新流程本身总会先拉取镜像 |
| GHCR 遗留标签 `2.0.0` / `2.0` | 与规范标签并存 | 可选清理（P5 报告 §9.3） |
| 提交身份为本地占位 | 提交作者信息不真实 | 用户提供真实身份后替换 |

---

## 8. 下一阶段计划

1. 提交并推送 `main`，创建并推送标签 `v2.0.1`。
2. 核验 CI、Release 流水线、GHCR 多架构镜像与匿名拉取。
3. 用已发布的 `v2.0.1` 镜像做一次真实的 `latest`/固定版本升级验证。
4. 在独立测试服务器完成 NPM 反代 HTTPS 部署验证。
