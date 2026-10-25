# Changelog

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。所有重要变更记录于此。

## [Unreleased]

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
