# Changelog

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。所有重要变更记录于此。

## [Unreleased]

### Added — P2（Go 后端核心业务）

- **认证**：一次性初始化管理员（Argon2id）、登录/登出、会话（服务端会话 + HttpOnly Cookie）、改密（改密后使全部会话失效）、登录限流（按 IP 与用户名）。
- **安全**：Origin/Referer 校验（CSRF 防护）、请求体大小限制、统一错误信封、URL 协议白名单、SSRF 防护的图标抓取、上传文件类型/体积校验、非 root 容器。
- **分类**：增删改查、排序、公开/私有（`is_visible`）。
- **导航项**：增删改查、排序、跨分类移动、可见性、图标类型、关键词、打开方式。
- **公开接口**：导航树聚合、分类、分类下导航项、搜索、公开设置、上传图标读取。
- **系统设置**：公开/管理视图分离、部分更新、取值校验。
- **图标**：上传（PNG/JPEG/WebP/GIF/ICO/SVG，≤1 MiB）与服务端抓取（防 SSRF）。
- **导入导出**：JSON 导出（含 `format`/`schema_version`/`checksum`）、导入预览、导入执行（`skip`/`overwrite`/`rename`，单事务，默认不覆盖）、旧站 `legacy-v1` 宽松映射。
- **备份恢复**：`VACUUM INTO` 快照（含图标目录）、快照列表、恢复（校验结构版本 + 自动安全快照）。
- **更新检查**：可选 GitHub Release 版本检查（只读）。
- **测试**：51 个后端测试（单元 + 接口），覆盖权限边界、可见性、校验与核心流程。

### Changed — P2

- 新增 `GET /api/v1/public/uploads/{filename}`（契约同步更新）。
- 新增环境变量 `XN_SECURE_COOKIE`（默认 true）、`XN_BACKUP_KEEP`、`XN_UPDATE_REPO`。
- 后端模块扩展：`internal/{models,auth,security,service}`，`store` 增加仓储层。

### Added — P1（项目初始化）

- 项目骨架：Go 后端（配置、SQLite、迁移、HTTP 路由与中间件）、Vue 3 前端骨架。
- Docker 部署工件：`Dockerfile`、`deploy/compose.yaml`、`scripts/deploy.sh`、`scripts/update.sh`。
- 文档体系：开发计划、架构、API 规范、数据库设计、前端交接规范、ADR、进度与交付记录。

### 说明

- 前端页面（P3）尚未实现，当前仅骨架。
- Docker 镜像构建与部署未验证（环境无 Docker 守护进程）。
- 旧站 `data.json` 真实样本未提供，`legacy-v1` 导入器仅按通用规则实现。
