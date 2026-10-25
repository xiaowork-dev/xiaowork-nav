# 架构决策记录（ADR）

本目录记录 XiaoWork Nav V2.0 的重要架构决策。每条决策一个文件，编号连续，**一经接受不修改内容**；若变更，新增一条 ADR 并标注取代关系。

| 编号 | 标题 | 状态 |
| --- | --- | --- |
| [ADR-0001](ADR-0001-backend-first.md) | 后端优先、API 驱动、契约先行 | 已接受 |
| [ADR-0002](ADR-0002-single-container-embed.md) | 单容器交付，Go embed 托管前端 | 已接受 |
| [ADR-0003](ADR-0003-sqlite-migrations.md) | SQLite + 版本化迁移 + 升级前备份 | 已接受 |
| [ADR-0004](ADR-0004-openapi-contract.md) | OpenAPI 作为前后端唯一契约 | 已接受 |
| [ADR-0005](ADR-0005-db-info-inspection.md) | 部署脚本通过 `--db-info` 直接读取数据库状态 | 已接受 |
| [ADR-0006](ADR-0006-icon-auto-fetch.md) | 服务端自动获取并缓存网站图标 | 已接受 |

## 模板

```md
# ADR-NNNN：标题

- 状态：提议 / 已接受 / 已废弃 / 被 ADR-XXXX 取代
- 日期：YYYY-MM-DD

## 背景
## 决策
## 理由
## 影响
## 备选方案
```
