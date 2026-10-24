# XiaoWork Nav V2.0 文档索引

> 最后更新：2026-10-24
> 上位依据：[`../XiaoWork_Nav_V2_重构方案与实施计划.md`](../XiaoWork_Nav_V2_重构方案与实施计划.md)

## 规划与架构

| 文档 | 内容 |
| --- | --- |
| [development-plan-v2.md](development-plan-v2.md) | 调整后的五阶段开发计划（后端优先） |
| [requirements.md](requirements.md) | 需求清单（P0/P1）与访问模型 |
| [legacy-analysis.md](legacy-analysis.md) | 旧站视觉/交互参考与迁移假设 |
| [architecture.md](architecture.md) | 总体架构、前后端模块边界、目录结构 |
| [api-spec.md](api-spec.md) | API 设计规范、端点清单、错误码 |
| [api-contract-review.md](api-contract-review.md) | API 契约一致性检查 |
| [database-design.md](database-design.md) | 数据库设计、迁移与备份策略 |
| [deployment.md](deployment.md) | Docker 部署、更新与排障 |
| [frontend-agent-guide.md](frontend-agent-guide.md) | 面向其他 AI Agent 的前端交接规范 |

## 契约

| 文件 | 内容 |
| --- | --- |
| [openapi/openapi.yaml](openapi/openapi.yaml) | OpenAPI 3.1 机器可读契约（基线） |

## 决策记录

| 文档 | 内容 |
| --- | --- |
| [decisions/README.md](decisions/README.md) | ADR 索引与模板 |
| [decisions/ADR-0001-backend-first.md](decisions/ADR-0001-backend-first.md) | 后端优先、API 驱动 |
| [decisions/ADR-0002-single-container-embed.md](decisions/ADR-0002-single-container-embed.md) | 单容器交付 |
| [decisions/ADR-0003-sqlite-migrations.md](decisions/ADR-0003-sqlite-migrations.md) | SQLite 与迁移备份 |
| [decisions/ADR-0004-openapi-contract.md](decisions/ADR-0004-openapi-contract.md) | OpenAPI 唯一契约 |

## 进度与交付

| 文档 | 内容 |
| --- | --- |
| [../AGENTS.md](../AGENTS.md) | **项目级 Agent 指令与阶段交付规范（必读）** |
| [PROGRESS.md](PROGRESS.md) | 项目进度总览 |
| [deliveries/P1-delivery.md](deliveries/P1-delivery.md) | **P1 阶段交付报告（正式归档）** |
| [deliveries/P2-delivery.md](deliveries/P2-delivery.md) | **P2 阶段交付报告（正式归档）** |
| [deliveries/](deliveries/) | 过程记录与后续阶段交付报告 |
