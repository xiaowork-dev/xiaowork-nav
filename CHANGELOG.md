# Changelog

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。所有重要变更记录于此。

## [Unreleased]

### Added — P3（Vue 3 前端功能与联调）

- **导航首页**：时钟、搜索引擎切换搜索框、分类分组与图标磁贴、主题切换、响应式布局。
- **搜索**：调用后端 `/api/v1/public/search`（防抖），失败回退本地过滤；回车使用所选引擎搜索。
- **初始化与认证**：首次初始化向导、管理员登录/退出、路由守卫（未初始化强制向导；管理路由未登录跳转并携带 `redirect`）。
- **分类管理**：增删改、公开/私有切换、拖拽排序与移动端 ↑↓ 排序、删除二次确认。
- **网站管理**：按分类筛选与关键词搜索、增删改、跨分类移动、公开/私有切换、图标上传/抓取/手填。
- **系统设置**：站点标题/Favicon、默认主题、首页布局、默认新标签、游客访问开关、搜索引擎增删与默认项；修改密码。
- **数据管理**：导出 JSON（浏览器下载）、导入（文件 → 预览 → 冲突策略 → 执行）、备份列表/创建/恢复、版本检查。
- **前端基础设施**：`src/api` 统一 HTTP 出口、Pinia 状态、`useDragSort`/`useTheme`/`useToast`、CSS 变量主题（浅色/深色）。
- **测试**：新增契约一致性测试与数据持久化测试（后端测试数 53）。

### Changed — P3

- 后端路由重构为单一来源 `internal/api/routes.go`，`Handler()` 与契约测试共用。
- 新增测试依赖 `gopkg.in/yaml.v3`。

### Fixed — P3

- 手动填写图标地址时自动同步 `icon_type=url`（此前会被后端清空）。
- 保存系统设置后刷新站点信息，页头标题即时更新。

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

### Changed — P2

- 新增 `GET /api/v1/public/uploads/{filename}`（契约同步更新）。
- 新增环境变量 `XN_SECURE_COOKIE`（默认 true）、`XN_BACKUP_KEEP`、`XN_UPDATE_REPO`。
- 后端模块扩展：`internal/{models,auth,security,service}`，`store` 增加仓储层。

### Added — P1（项目初始化）

- 项目骨架：Go 后端（配置、SQLite、迁移、HTTP 路由与中间件）、Vue 3 前端骨架。
- Docker 部署工件：`Dockerfile`、`deploy/compose.yaml`、`scripts/deploy.sh`、`scripts/update.sh`。
- 文档体系：开发计划、架构、API 规范、数据库设计、前端交接规范、ADR、进度与交付记录。

### 说明

- 移动端视口/真机、跨浏览器、Docker 镜像构建与旧站真实数据导入尚未验证。

