# Changelog

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。所有重要变更记录于此。

## [Unreleased]

## [2.0.1] - 2026-10-25

修复一键更新与数据库回滚路径上的缺陷（P5 验收复审发现）。**不涉及数据库结构变更，
`schema_version` 仍为 `2`**，可从 2.0.0 直接升级。

### Added

- 新增 `xiaowork-nav --db-info [--db <path>]` 应用内命令行模式：以只读方式输出
  `db_path` / `exists` / `schema_version` / `categories` / `bookmarks` / `uploads`，
  供部署脚本在任意 Compose 模式与容器停止时读取数据库状态。
- 新增异常注入测试 `scripts/tests/deploy-faults.test.sh`（10 个场景，需 Docker）：
  幂等更新、镜像未变化、镜像缺失、容器启动失败、迁移后健康检查失败、无法读取
  数据库版本、容器停止时读取结构版本、恢复时版本切换、快照版本未知时拒绝启动、
  NPM 模式（无宿主机端口）升级失败回滚。

### Fixed

- **`latest` 更新逻辑**：不再因为"目标版本等于当前版本"而直接退出。可变标签
  （`latest`、`v2.0` 等）现在总是查询并拉取最新镜像；镜像摘要未变且容器健康时
  明确报告"无需操作"并且**不重建容器**；固定语义化版本的幂等行为保持不变。
- **NPM 模式数据库版本检测**：不再依赖宿主机 `127.0.0.1` 端口。结构版本改为从
  数据库文件读取（`--db-info`），因此在不发布宿主机端口的 NPM 共享网络模式下、
  以及应用已停止时都能正确判断；HTTP 仅作为独立模式下的兜底。检测过程不会把
  应用端口暴露到公网。
- **更新失败自动回滚**：覆盖镜像拉取失败、容器启动失败、健康检查失败、迁移后
  启动失败、无法读取数据库版本、NPM 模式失败六类场景。当结构版本**变大**、
  **无法读取**或**更新前版本未知**时，先用更新前快照恢复数据，再切回旧镜像，
  避免用旧镜像打开新结构数据库。
- **回滚后再次验证**：不再只依赖 `docker compose up -d` 的退出码，而是重新等待
  健康检查并核对结构版本与书签行数。
- **`restore.sh` 恢复流程**：恢复数据后**先**把 `APP_VERSION` 切换到快照所属
  版本（由 `--version` 指定或从文件名推断）再启动，避免刚恢复的旧库被新镜像
  重新迁移；版本无法确定时只恢复数据、保持停止并打印指引，**不猜测**。

### Changed

- `scripts/lib.sh`：新增 `image_ref_for` / `local_image_id` / `container_image_id`
  / `is_mutable_tag` / `db_info` / `db_field` / `schema_version` / `snapshot_dir` /
  `restore_snapshot` / `version_from_snapshot`；结构版本检测统一走数据库文件，
  不再保留基于宿主机端口的 HTTP 路径。
- `docs/updating.md`、`docs/deployment.md`、`docs/installation.md`：补充新的更新
  判定、`--db-info` 用法、回滚保证与排障条目。

## [2.0.0] - 2026-10-25

首个正式发布：轻量、可自托管、可一键部署与更新的个人导航站。

### Added

**导航前台**
- 分类分组与图标磁贴首页，含时钟与响应式布局（桌面 / 平板 / 手机 390px）。
- 搜索：服务端搜索（名称 / 描述 / 关键词）+ 可配置搜索引擎切换，回车用所选引擎搜索。
- 图标回退链：用户图标（图片地址 / 本地上传 / Simple Icons 标识 / 文字图标）→ 站点
  Favicon → 首字母；自动步骤仅为显示回退，不会改写已保存的图标地址。
- 明暗主题，默认跟随系统，支持本地覆盖。

**管理后台**
- 首次运行初始化向导（一次性创建管理员，之后关闭入口）。
- 分类与网站的增删改查、拖拽排序、移动端 ↑↓ 排序替代、公开/私有可见性。
- 网站管理分页（每页 20/50/100 条）与显式「排序模式」。
- 图标：本地上传、外部地址、已验证的 Simple Icons 标识、文字图标、站点图标抓取。
- 系统设置：站点标题 / Favicon / 主题 / 首页布局 / 搜索引擎 / 游客访问开关；修改密码。
- 数据管理：JSON 导出、导入（含预览与冲突策略）、备份创建与恢复、版本检查。

**数据与迁移**
- SQLite（WAL）持久化，版本化迁移（`schema_migrations`，事务执行，失败拒绝启动）。
- 旧站数据导入：支持顶层数组结构，识别 `category`/`items`/`logo`/`si`/`tag`/`color`/`iconType`；
  内嵌 `data:image/svg+xml` 图标解码为本地上传文件（内容无损）；外部图标地址原样保留。
- 备份快照同时覆盖数据库与上传图标。

**部署与运维**
- 多阶段 `Dockerfile`：前端构建 → Go(embed) → 精简非 root 运行镜像（约 36 MB）。
- 一键安装脚本 `scripts/install.sh`，以及 `deploy.sh` / `update.sh` / `backup.sh` / `restore.sh`。
- Docker Compose 两种模式：独立模式（默认 `127.0.0.1:8787`）与 NPM 共享网络模式（不发布端口）。
- 更新前自动一致性快照；失败时自动回滚；结构版本保护（旧镜像拒绝打开新数据库）。
- GitHub Actions：CI（构建 / 测试 / 镜像构建 / 仓库安全检查）与 Release（多架构镜像推送
  GHCR + GitHub Release）。
- 全部部署参数通过环境变量配置，未写死域名或网络名。

### Changed

- 前端整页渐变背景（不再依赖 `background-attachment: fixed`）；浅色主题增强。
- 磁贴底层始终渲染首字母占位，避免加载中或图标失效时出现空白方块。
- 移动端管理后台列表改为两行布局（名称在上、操作在下），触控目标 ≥38px。

### Fixed

- 导入时分类 `sort_order` 被硬编码为 0。
- 导入冲突检测使用数据包内 ID，可能误判。
- 书签引用未知分类时猜测 ID 可能撞车。
- 导入失败时遗留孤儿图标文件与上传记录。
- HTML 内容可伪装成 `.svg` 图标。
- 内嵌图标转换失败时未清理。
- 手动填写图标地址时 `icon_type` 未同步为 `url`。
- 保存系统设置后页头标题不刷新。

### Security

- Argon2id 密码哈希；服务端会话 + `HttpOnly`/`SameSite` Cookie（生产 `Secure`）。
- 状态变更请求校验 `Origin`/`Referer`；登录按 IP 与用户名限流。
- URL 协议白名单（仅 http/https）；图标抓取防 SSRF（含云元数据地址与重定向限制）。
- 上传大小 / 类型 / 文件名白名单校验；SVG 以沙箱 CSP 提供，绝不以 HTML 注入。
- 容器非 root 运行并启用 `no-new-privileges`。
- 数据库结构版本高于镜像时拒绝启动，避免降级破坏数据。

### Notes

- 升级方式见 [docs/updating.md](docs/updating.md)；安装方式见 [docs/installation.md](docs/installation.md)。
- 旧站导入的字段映射与异常数据说明见 [docs/legacy-data-migration-report.md](docs/legacy-data-migration-report.md)。
