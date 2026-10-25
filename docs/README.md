# XiaoWork Nav V2.0 文档索引

> 最后更新：2026-10-24
> 上位依据：[`../XiaoWork_Nav_V2_重构方案与实施计划.md`](../XiaoWork_Nav_V2_重构方案与实施计划.md)

## 规划与架构

| 文档 | 内容 |
| --- | --- |
| [development-plan-v2.md](development-plan-v2.md) | 调整后的五阶段开发计划（后端优先） |
| [development-plan-v2.1.md](development-plan-v2.1.md) | **V2.1.0 实施计划（图标自动获取 / 滚动布局侧栏 / 标签布局）** |
| [requirements.md](requirements.md) | 需求清单（P0/P1）与访问模型 |
| [legacy-analysis.md](legacy-analysis.md) | 旧站视觉/交互参考与迁移假设 |
| [architecture.md](architecture.md) | 总体架构、前后端模块边界、目录结构 |
| [api-spec.md](api-spec.md) | API 设计规范、端点清单、错误码 |
| [api-contract-review.md](api-contract-review.md) | API 契约一致性检查 |
| [database-design.md](database-design.md) | 数据库设计、迁移与备份策略 |
| [legacy-data-migration-report.md](legacy-data-migration-report.md) | **旧站真实数据迁移报告（字段映射、导入结果、异常数据）** |
| [installation.md](installation.md) | **安装指南（前置条件、两种模式、首次使用、排障）** |
| [updating.md](updating.md) | **更新、备份、恢复与回滚指南（含迁移兼容性）** |
| [deployment.md](deployment.md) | 部署总览与关键设计约束 |
| [frontend-agent-guide.md](frontend-agent-guide.md) | 面向其他 AI Agent 的前端交接规范 |

## 测试样本

| 文件 | 内容 |
| --- | --- |
| [samples/legacy-data.sample.json](samples/legacy-data.sample.json) | 脱敏的旧站数据样本（供自动化测试；真实数据不入库） |

## 视觉审查材料

> 以下快照包含真实站点数据，**不纳入版本库**（`.gitignore` 已排除 `docs/screenshots/`），仅保留在本机。

| 文件（本地，未入库） | 内容 |
| --- | --- |
| `docs/screenshots/home-page-snapshot.html` | 导航首页的离线 DOM 快照（真实渲染结果，非截图） |
| `docs/screenshots/admin-bookmarks-snapshot.html` | 后台网站管理列表的离线 DOM 快照 |

## 契约

| 文件 | 内容 |
| --- | --- |
| [openapi/openapi.yaml](openapi/openapi.yaml) | OpenAPI 3.1 机器可读契约（基线） |
| [../SECURITY.md](../SECURITY.md) | 安全策略与部署加固清单 |

## 决策记录

| 文档 | 内容 |
| --- | --- |
| [decisions/README.md](decisions/README.md) | ADR 索引与模板 |
| [decisions/ADR-0001-backend-first.md](decisions/ADR-0001-backend-first.md) | 后端优先、API 驱动 |
| [decisions/ADR-0002-single-container-embed.md](decisions/ADR-0002-single-container-embed.md) | 单容器交付 |
| [decisions/ADR-0003-sqlite-migrations.md](decisions/ADR-0003-sqlite-migrations.md) | SQLite 与迁移备份 |
| [decisions/ADR-0004-openapi-contract.md](decisions/ADR-0004-openapi-contract.md) | OpenAPI 唯一契约 |
| [decisions/ADR-0005-db-info-inspection.md](decisions/ADR-0005-db-info-inspection.md) | 部署脚本通过 `--db-info` 读取数据库状态 |

## 进度与交付

| 文档 | 内容 |
| --- | --- |
| [../AGENTS.md](../AGENTS.md) | **项目级 Agent 指令与阶段交付规范（必读）** |
| [PROGRESS.md](PROGRESS.md) | 项目进度总览 |
| [deliveries/P1-delivery.md](deliveries/P1-delivery.md) | **P1 阶段交付报告（正式归档）** |
| [deliveries/P2-delivery.md](deliveries/P2-delivery.md) | **P2 阶段交付报告（正式归档）** |
| [deliveries/P3-delivery.md](deliveries/P3-delivery.md) | **P3 阶段交付报告（正式归档）** |
| [deliveries/P4-delivery.md](deliveries/P4-delivery.md) | **P4 阶段交付报告（正式归档）** |
| [deliveries/P5-delivery.md](deliveries/P5-delivery.md) | **P5 阶段交付报告（正式归档，含 §12 v2.0.1 补充）** |
| [deliveries/v2.1.0/](deliveries/v2.1.0/) | **V2.1.0 阶段交付报告（P1 起）** |
| [deliveries/20261025-170154-p5-update-rollback-fix.md](deliveries/20261025-170154-p5-update-rollback-fix.md) | **P5 补充交付记录：一键更新与回滚修复（v2.0.1）** |
| [deliveries/](deliveries/) | 过程记录与后续阶段交付报告 |
