# XiaoWork Nav V2.0：重构方案与实施计划

> 文档类型：架构选型 / V2.0 需求范围 / 交付计划  
> 创建日期：2026-10-24  
> 项目状态：**方案阶段，尚未初始化代码仓库或部署服务器**  
> 项目代号：XiaoWork Nav（暂定）  
> 仓库名称建议：`xiaowork-nav`（**仅为建议，尚未创建或验证可用性**）

> **2026-10-24 计划调整说明**：本文档仍为项目总体依据，但**开发顺序与前后端架构已按新要求调整**为"后端优先、API 驱动、严格前后端分离"。
> 第 8 节的分阶段计划已被 [`docs/development-plan-v2.md`](docs/development-plan-v2.md) 取代；架构、API、数据库与前端交接规范见 [`docs/`](docs/README.md)。
> 技术栈、部署目标与安全底线保持本文档原有约定。

---

## 0. 本次重构的核心目标

将现有 HTML/CSS/JavaScript + PHP + JSON 数据文件的个人导航站，重构成轻量、可自托管、可公开开源的产品。

**必须实现：**

1. **一键部署**：用户不需要自己构建前端或安装数据库，通过发布的 Docker 镜像、Docker Compose 和安装脚本启动。
2. **一键更新**：每次正式发布后，部署用户执行一次更新脚本即可升级，并保留数据；后续可选自动更新。
3. **GitHub 自动发版**：通过 GitHub Actions 进行测试、构建多架构镜像、推送 GHCR 和发布版本说明。
4. **可视化管理**：分类、导航项、图标、排序、搜索和外观通过界面维护，无须手改 JSON。
5. **数据安全**：持久化存储、升级前备份、导入导出、迁移、故障回退。
6. **继承旧站体验**：保留截图中的渐变背景、分组导航、搜索框、图标卡片和桌面优先的清爽布局，同时完善手机端。
7. **开源友好**：README、LICENSE、示例环境配置、版本日志、部署手册、故障排查与贡献指南齐全。

**不追求：** 在 V2.0 就加入多租户、社交、复杂权限、支付、推荐算法和微服务。这些会抬高自托管维护成本。

---

## 1. 当前旧站特征（根据已有使用情况与本次截图）

- 首页由搜索框和多个分类组成，如 Daily、SpeedTest、常用、IP&DNS、AI & 智能、VPS/云服务器、邮箱、开发与编程、域名等。
- 每个导航项具有名称、图标、链接、分类、排序信息。
- 旧实现主要使用 HTML/CSS/JS，PHP + `data.json` 做后端同步，运维过程需要手动部署。
- 已有过添加、删除、拖动排序、长按/右键操作、本地与后端同步等功能需求。
- 风格基线：紫蓝渐变、居中搜索、图标网格、分组标题；重构不需要彻底推翻视觉。

**旧站数据迁移注意：** 必须先取得真实旧 `data.json` 及附件/图标组织方式，再编写转换逻辑。当前未取得旧站源代码或 JSON，不得宣称迁移器已支持具体旧格式。

---

## 2. 推荐技术架构

| 层 | 技术 | 原因 |
|---|---|---|
| Web UI | Vue 3 + TypeScript + Vite | 管理界面方便、构建后为静态文件 |
| 样式 | 原生 CSS / CSS 变量，必要时使用轻量 UI 组件 | 高度保留原视觉、减少运行时依赖 |
| API | Go（稳定受支持版本） | 单进程、低内存、适合 Docker |
| 数据库 | SQLite | 单实例场景足够、免配置数据库 |
| 静态资源 | Go `embed` | 构建产物与 API 作为一个应用发布 |
| 容器 | Docker 多阶段构建 + Docker Compose | 一个应用容器部署，无 Node 运行时 |
| 自动发版 | GitHub Actions | 测试、生成镜像和版本发布 |
| 镜像仓库 | GitHub Container Registry (GHCR) | 公共项目可匿名拉取公共镜像 |
| 外部 HTTPS | 用户已有的 Nginx Proxy Manager / 任意反代 | 容器内部走 HTTP，TLS 在反代层终止 |

### 架构示意

```text
Browser (desktop / mobile)
          |
        HTTPS
          |
Reverse proxy (NPM / Nginx / Caddy)
          |
   private Docker network
          |
  xiaowork-nav container
      |             |
  Vue static SPA   Go REST API
                    |
                SQLite /data
                    |
              uploaded icons
```

**取舍说明：**

- 默认只有一个应用容器，SQLite 数据保存在挂载的 `/data` 目录，不与镜像绑定。
- SQLite 使用 WAL 模式需规范处理数据库文件与备份；运行时备份使用 SQLite Online Backup API 或 `VACUUM INTO`，不直接热拷贝单个 db 文件。
- Go 服务提供 `/healthz`，用于部署成功判定和健康检查。
- 静态页面由 Go 服务托管，刷新 SPA 路由时返回 `index.html`，API 路径除外。
- 若未来需要多实例、高并发、多用户协作，再评估 PostgreSQL；V2.0 不需要。

---

## 3. V2.0 功能范围

### 3.1 P0：首个正式发布必须完成

**前台导航**

- 按分类展示图标卡片，分类折叠/展开可作为用户设置。
- 本地实时搜索：按名称、描述、关键词过滤；支持快捷使用搜索引擎。
- 支持后台配置多种搜索引擎及默认选项。
- PC 和移动端响应式布局；适配手机端触控与长按。
- 新标签打开规则、站点标题、Favicon、背景、颜色主题可配置。
- 导航项可设置图标：上传本地图标、输入 URL 或使用站点图标抓取机制；抓取应由服务端安全限制网络目标，防止 SSRF。

**后台管理**

- 首次运行的初始化向导：安全设置唯一管理员账号和密码，初始化后关闭公开注册。
- 登录、登出、修改密码、会话管理；使用安全哈希及 HttpOnly/Secure/SameSite Cookie。
- 分类的增删改、排序与显示开关。
- 导航项的增删改、排序、分组移动、关键词、图标和链接管理。
- 拖拽排序；移动端提供排序按钮/触控替代操作。
- 首页展示模式配置、基础站点设置。
- 更新信息页（展示当前版本、最新稳定版本、发行说明链接；V2.0 不直接让 Web 容器控制宿主 Docker）。

**数据与运维**

- SQLite 数据持久化到 `/data`，上传图标存储至 `/data/uploads`。
- JSON 格式的数据导出 / 导入：带格式版本号、校验、冲突处理与导入预览。
- 旧站 JSON 导入器：拿到旧数据样本后适配，确保不覆盖已有数据。
- 备份与恢复（数据库和上传图标一起备份）；自动保留有限历史快照。
- 数据库迁移版本化、可测试，升级前备份，失败时阻止危险的继续启动。
- 提供 `install.sh`、`update.sh`、`backup.sh`、`restore.sh` 和 `compose.yaml`。
- 安装后可根据所选模式使用 NPM 反代 HTTPS，或使用直连端口运行。
- GitHub Actions 自动测试并在正式 Release 时发布 AMD64/ARM64 镜像。

### 3.2 P1：V2.1 可考虑

- 页面个性化布局、图标尺寸、网格密度设置。
- 快捷键、快捷添加、批量导入链接与批量分类。
- 支持公开导航和私人导航两种可见性。
- 多主题与背景图片上传。
- 浏览器书签 HTML 导入、PWA 安装。
- 使用者自愿启用定时更新检查；可接系统通知。

### 3.3 暂不做

- 多用户、多租户、复杂角色权限体系。
- 使用 Docker Socket 的网页内“一键升级服务器”按钮。
- 对接外部 AI 服务生成标签及内容。
- 复杂全文搜索服务和数据分析平台。

---

## 4. 数据模型（建议）

```text
users
  id, username, password_hash, created_at, updated_at

categories
  id, name, description, sort_order, is_visible, created_at, updated_at

bookmarks
  id, category_id, title, url, description, icon_type, icon_value,
  keywords, sort_order, is_visible, open_in_new_tab, created_at, updated_at

site_settings
  key, value_json, updated_at

schema_migrations
  version, applied_at
```

原则：所有排序字段可持久化；分类与导航项 ID 长期稳定；安全校验 URL 协议；文本输出转义，防范 XSS；图标与导入数据严格限制格式、容量及外网访问。

---

## 5. 安装模式设计

### A. NPM 反代模式（你的 VPS 首选）

- 应用容器不设置公网 `ports`，仅 `expose: ["8080"]` 用于文档说明。
- 应用与 Nginx Proxy Manager 容器加入同一个用户自定义 Docker 网络。
- NPM 的转发主机填 `xiaowork-nav`、端口 `8080`（确切名称以最终 Compose 服务名/网络别名为准）。
- DNS 指向服务器；TLS 由 NPM 终止；应用自行识别受信任代理传入的 HTTPS 信息。
- **效果：没有应用宿主机发布端口，因此公网 IP+应用端口访问不到应用。** 反代管理后台是否暴露属于 NPM 自身配置，与导航站不同。
- 安装脚本需确认已有网络名称并进行连接；不能假定用户的 NPM 网络恰好叫 `proxy`。

### B. 独立模式（给其他开源用户）

- 安装脚本可选择指定宿主机监听地址和端口。
- 默认建议回环监听 `127.0.0.1:8787` 供同宿主机反代调用；若用户明确要求公网直连，可选择 `0.0.0.0:8787`，同时提醒其需配置 TLS/防火墙。
- 不应将“绑定 127.0.0.1”当作 NPM 容器一定可访问的保证；Docker 桥接网络下需要对应网络连接或另行设计。

安装体验目标：

```bash
# 以下只是未来发行版的使用方式示意，当前仓库/脚本尚不存在
./install.sh
```

脚本应检查 Docker + Compose、磁盘空间、端口/网络冲突，创建权限合理的数据目录，获取**正式发行版**，完成启动和健康检查，最后输出地址及初始化说明。

公开 README 可说明如何从具体的 GitHub Release 下载并**先审核安装脚本**，尽量避免直接信任未审核的 `curl | bash` 执行方式。

---

## 6. 发布与升级流程（核心交付）

### 6.1 GitHub Actions

1. 提交代码至 `main` 后执行 lint、类型检查、单元测试和构建验证。
2. 创建正式版本标签 `v2.0.0`、`v2.0.1` 等后执行完整测试。
3. 构建 Linux AMD64 和 ARM64 的多架构 Docker 镜像。
4. 将镜像推送到公共 GHCR，例如：`ghcr.io/xiaowork-dev/xiaowork-nav:v2.0.0`（仅示例，实际以新建仓库和镜像为准）。
5. 按发布策略更新 `latest`，同时保留每一个不可变的版本号标签。
6. 创建 GitHub Release，附版本说明和升级注意事项。
7. 不应在每一次未完成测试的 `main` 提交上覆盖公开稳定版本。

### 6.2 服务器一键更新

```bash
# 设计中的 CLI，尚未实现
./update.sh
```

`update.sh` 应执行：

1. 检查当前版本及最新稳定 Release；若有预发布版本，不自动选择。
2. 确认镜像可用，读取版本迁移说明及最低兼容要求。
3. **先在线备份数据与图标**，写入带版本、时间戳的快照。
4. 检查磁盘剩余空间；下载目标版本镜像。
5. 原子写入持久部署配置中的 `APP_VERSION=vX.Y.Z`；执行 Docker Compose 更新。
6. 等待 `/healthz` 并进行数据库与核心接口验证。
7. 失败时尽可能恢复上版容器；若升级运行了不可逆数据库迁移，应配合升级前快照恢复，而不是仅切回旧镜像。
8. 输出成功/失败信息、新旧版本、备份位置和回滚说明。

Compose 的基本更新机制是：

```bash
docker compose pull
docker compose up -d
```

但正式脚本还必须封装版本切换、备份、健康检查和回滚条件，不能只有上面两行。

### 6.3 更新策略

- **默认：手动一键更新。** 允许用户确认新版本后运行 `./update.sh`。
- **可选：定时检测提醒。** 可以从发布 API 获取版本元数据并显示提示，不执行更新。
- **可选：无人值守自动更新。** 后续通过宿主机受控的 systemd timer 调用同一套更新脚本；只追踪指定稳定通道。
- 不建议为方便在 Web 界面升级而挂载 `/var/run/docker.sock`，这会让应用接近拥有宿主 Docker 管理能力。

### 6.4 Git 交付规则

- 从项目首次配置 Git 和远程 GitHub 仓库开始，每个完成的开发阶段都应提交 commit 并推送（在授权与认证条件满足时）。
- 每次版本更新必须有版本号、CHANGELOG、必要的数据迁移说明、自动化测试结果。
- 若无权限、身份验证失败或远程不可用，须明确在交付文档中写明“未推送成功”，不可假称已同步。

---

## 7. 推荐目录结构

```text
xiaowork-nav/
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   ├── components/
│   │   ├── composables/
│   │   ├── api/
│   │   └── styles/
│   └── package.json
├── backend/
│   ├── cmd/server/
│   ├── internal/
│   │   ├── api/
│   │   ├── auth/
│   │   ├── config/
│   │   ├── storage/
│   │   ├── migrations/
│   │   └── bookmarks/
│   └── go.mod
├── deploy/
│   ├── compose.yaml
│   ├── .env.example
│   └── compose.npm.yaml
├── scripts/
│   ├── install.sh
│   ├── update.sh
│   ├── backup.sh
│   └── restore.sh
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── docs/
│   ├── architecture.md
│   ├── deployment.md
│   ├── update-and-rollback.md
│   ├── data-migration.md
│   └── delivery/
├── Dockerfile
├── README.md
├── CHANGELOG.md
├── CONTRIBUTING.md
├── LICENSE
└── .gitignore
```

> Go embed 的静态前端打包细节需在实际构建时安排好资源路径；上面是职责划分示意，不等于已可直接编译运行的目录。

---

## 8. 分阶段开发与验收

| 阶段 | 交付内容 | 验收标准 |
|---|---|---|
| P0 方案评审 | 需求、架构、数据模型、迭代清单 | 范围确认，不启动真实部署 |
| P1 项目骨架 | Go API、Vue 前端、基础路由、构建流程、Dockerfile | 本地和 Docker 可访问健康检查/首页 |
| P2 业务闭环 | 分类/书签增删改查、排序、搜索、图标 | 页面操作刷新不丢数据，接口有测试 |
| P3 后台与安全 | 初始化管理员、登录、会话、限流、安全审计基础 | 未授权无法修改数据；首次初始化只允许一次 |
| P4 数据迁移 | 导入导出、旧 JSON 转换、备份与恢复 | 导入真实旧站测试数据，不丢分类/图标 |
| P5 部署与更新 | 安装脚本、更新脚本、回滚、NPM 配置 | 全新 Linux VPS 可部署；旧版本可升级且数据保留 |
| P6 开源发布 | README、Actions、GHCR、Release、许可证 | 公开的干净环境可安装，AMD64/ARM64 通过测试 |

**每一阶段交付文档统一模板**（存放在 `docs/delivery/`）：

```md
# 阶段交付报告：P?-阶段名

## 本阶段目标
## 实际完成
## 代码与文件变更
## 测试/验收证据
## 已知问题与风险
## 当前总体进度
## Git commit / GitHub push 状态
## 下一阶段工作
```

执行规范：每一阶段完成后立即生成文档；明确交付清单和状态；首次绑定 GitHub 后每次改版均应提交并推送；失败需报告。

---

## 9. 安全与质量底线

- 密码使用 Argon2id 或经过合理配置的 bcrypt 方案哈希，不在日志、配置、代码中明文写默认管理员密码。
- 初次安装通过一次性初始化流程创建管理员，安装完成后关闭初始化入口。
- 认证 Cookie 设置 `HttpOnly`、生产 HTTPS 下 `Secure`、合理 `SameSite`，状态变更接口进行 CSRF/Origin 防护。
- API 参数校验、请求大小限制、登录失败限流与权限检查。
- 只允许 `http:`、`https:` 等已批准的书签 URL 协议，禁止 `javascript:` 链接。
- 如果后端抓取 favicon，必须防 SSRF（禁止内网/环回/元数据 IP、DNS 重绑定、任意重定向等）。
- 数据文件和备份不经过 Web 公共静态目录访问，`.env` 不提交到仓库。
- 上传文件进行 MIME、扩展名、内容与体积校验，避免任意文件执行。
- 容器使用非 root 运行（如可行）、最小权限、合理健康检查和日志配置。
- 自动发版前执行依赖扫描、基础测试与发布审核。
- 重要变更验证 SQLite 数据迁移；每次升级先备份；备份还应定期试恢复。
- `LICENSE` 根据开源目标在正式公开之前选择（可考虑 MIT / Apache-2.0）。

---

## 10. 需要的外部输入（到对应阶段再提供）

1. 原导航站的完整代码和真实 `data.json`，用于功能对照及数据迁移。
2. 新 GitHub 公开仓库的最终名称与位置。
3. 已部署 NPM 的实际 Docker 网络名称及计划使用的子域名（**不要在聊天中公开密码或私人密钥**）。
4. 是否允许游客访问首页，以及是否存在仅管理员可见的分类（默认 V2.0：游客可浏览公开导航、只有管理员可修改）。

这些不是启动架构设计的前置阻塞项，但在 P4、P5、P6 前必须确认，避免推测旧数据结构或擅自操作服务器。

---

## 11. Codex 执行总指令（建议直接作为项目规则）

> 你正在开发 XiaoWork Nav V2.0。以本文档为需求与架构基线：Vue 3 + TypeScript + Vite 前端、Go 单进程后端、SQLite 持久化、单容器 Docker 部署、GitHub Actions + GHCR 发版。禁止未沟通就加入微服务、多余数据库或 Docker Socket Web 管理权限。  
> 必须严格按 P1 → P6 顺序完成；每阶段完成都必须运行测试、记录文件变更和验证结果、输出 `docs/delivery/` 交付报告。项目启用 Git 远程后，每次有效改版都必须提交并推送 GitHub；失败则明确说明。不要声称未实际执行的测试、部署或推送已成功。  
> 部署必须同时支持 NPM 共享网络（应用无公网发布端口）与通用独立模式。更新脚本必须在修改应用前进行一致性备份，并校验新版本健康；为不可逆迁移准备成对的数据快照恢复路径。镜像必须来自正式 GitHub Release，并保留固定语义化版本标签。  
> 在开始编码前先输出详细的 P1 任务拆分、验收标准及预期交付物，确认与本文档一致，再执行当前阶段；每次仅声明真实完成的阶段成果。

---

## 12. 本次方案阶段交付记录

- **已完成：** 项目重构目标定义、技术选型、功能范围、部署/升级机制、数据与安全策略、开发阶段安排。
- **未完成：** 源代码生成、GitHub 新仓库创建、CI/CD 实测、旧站数据转换器、Docker 镜像发布、服务器部署。
- **当前阶段：** P0 方案草案。
- **下一阶段建议：** 检查旧站源码与数据，锁定 V2.0 页面交互细节和路由 API 设计，再由 Codex 创建 P1 项目骨架。

## 参考资料（官方）

- Docker Compose pull: https://docs.docker.com/reference/cli/docker/compose/pull/
- Docker Compose up: https://docs.docker.com/reference/cli/docker/compose/up/
- GitHub Container Registry: https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry
- GitHub Actions 发布镜像: https://docs.github.com/en/actions/tutorials/publish-packages/publish-docker-images
- SQLite Online Backup API: https://www.sqlite.org/backup.html
