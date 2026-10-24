# XiaoWork Nav V2.0：需求清单

> 文档类型：需求规格
> 更新日期：2026-10-24
> 状态：P1 基线（待契约冻结确认）
> 关联：`development-plan-v2.md`、`api-spec.md`、`openapi/openapi.yaml`

本文档将 V2.0 功能拆分为可追溯到接口/页面的条目。优先级：**P0 首个正式发布必须完成**，P1 为后续增强。

---

## 0. 角色与访问模型

| 角色 | 权限 |
| --- | --- |
| 游客（未登录） | 浏览**公开**分类与网站；使用搜索；切换主题；使用搜索引擎 |
| 管理员（唯一） | 游客全部权限 + 管理后台全部读写 + 系统设置 + 导入导出 + 备份恢复 |

**可见性规则（已确认）：**

- 导航首页允许游客公开访问。
- 游客仅能看到 `is_visible = true` 的分类与导航项。
- 私有分类/网站（`is_visible = false`）仅管理员登录后可见。
- **所有数据修改操作必须登录**，无例外。
- 管理后台必须身份认证。

> 说明：V2.0 以"可见/私有"二元模型实现私有内容，不引入多用户角色体系。

---

## 1. P0 功能需求

### 1.1 前台导航

| 编号 | 需求 | 对应接口/页面 | 优先级 |
| --- | --- | --- | --- |
| F-01 | 首页按分类分组展示图标卡片 | `GET /public/navigation` · 首页 | P0 |
| F-02 | 每个网站显示名称与图标（上传/URL/自动抓取/无） | `Bookmark.icon_type` | P0 |
| F-03 | 实时本地搜索：按名称、描述、关键词过滤 | `GET /public/search` + 前端本地过滤 | P0 |
| F-04 | 多搜索引擎配置与切换 | `/public/settings` `search_engines` | P0 |
| F-05 | 点击站点按配置在新标签打开 | `Bookmark.open_in_new_tab` | P0 |
| F-06 | 分类锚点导航 / 快速跳转 | 首页 | P0 |
| F-07 | 响应式布局（桌面/平板/手机） | 首页 | P0 |
| F-08 | 时钟 / 日期展示（参考旧站视觉） | 首页 | P1 |
| F-09 | 深浅色主题切换并持久化 | `/public/settings` `theme` | P0 |
| F-10 | 站点标题与 Favicon 可配置 | `/public/settings` | P0 |
| F-11 | 分类折叠/展开（可记忆） | 首页 | P1 |

### 1.2 后台管理

| 编号 | 需求 | 对应接口 | 优先级 |
| --- | --- | --- | --- |
| A-01 | 首次运行初始化唯一管理员，初始化后关闭注册 | `GET /auth/status`、`POST /setup` | P0 |
| A-02 | 登录、登出、会话保持 | `POST /auth/login`、`/auth/logout` | P0 |
| A-03 | 修改密码 | `PUT /auth/password` | P0 |
| A-04 | 分类增删改查 | `/admin/categories` | P0 |
| A-05 | 分类排序（拖拽）与显隐开关 | `PUT /admin/categories/order`、`PUT /admin/categories/{id}` | P0 |
| A-06 | 导航项增删改查 | `/admin/bookmarks` | P0 |
| A-07 | 导航项排序、跨分类移动 | `PUT /admin/bookmarks/order`、`POST /admin/bookmarks/{id}/move` | P0 |
| A-08 | 导航项图标：上传 / URL / 抓取 | `/admin/uploads/icon`、`/admin/icons/fetch` | P0 |
| A-09 | 站点设置（标题、Favicon、主题、搜索引擎、布局） | `GET/PUT /admin/settings` | P0 |
| A-10 | 移动端排序替代操作（按钮上移/下移） | 后台页面 | P0 |
| A-11 | 更新信息页（当前版本/最新版本，只读） | `GET /admin/update-check` | P1 |

### 1.3 数据与运维

| 编号 | 需求 | 对应接口/机制 | 优先级 |
| --- | --- | --- | --- |
| D-01 | SQLite 持久化到 `/data`，图标存 `/data/uploads` | 部署配置 | P0 |
| D-02 | JSON 导出（带格式版本与校验） | `GET /admin/export` | P0 |
| D-03 | JSON 导入（预览、冲突策略、不覆盖已有数据） | `POST /admin/import/preview`、`/admin/import` | P0 |
| D-04 | **旧站数据 JSON 导入**（`format: legacy-v1`） | 同上 | P0 |
| D-05 | 备份与恢复（数据库 + 图标，保留历史快照） | `/admin/backups*` | P0 |
| D-06 | 版本化数据库迁移，升级前备份，失败阻止启动 | `internal/migrations` | P0 |
| D-07 | 一键部署 / 一键更新脚本 | `scripts/` | P0 |
| D-08 | GitHub Actions 测试 + 多架构镜像 + GHCR + Release | `.github/workflows` | P0 |

> D-04 说明：**当前不提供旧站源码与 `data.json`**，导入器先按通用 JSON 结构实现，拿到真实样本后再适配具体字段映射，不阻塞其余开发。

---

## 2. P1 增强（V2.1 候选）

- 图标尺寸/网格密度个性化、页面布局模式。
- 快捷键、快捷添加、批量导入链接与分类。
- 多主题与背景图片上传。
- 浏览器书签 HTML 导入、PWA 安装。
- 定时更新检查提醒。

---

## 3. 非功能需求

| 类别 | 要求 |
| --- | --- |
| 安全 | 密码 Argon2id；HttpOnly/Secure/SameSite Cookie；CSRF/Origin 校验；URL 协议白名单；图标抓取防 SSRF；上传校验类型与体积 |
| 性能 | 首页单次请求获取导航数据；列表分页；SQLite WAL |
| 可维护性 | 严格前后端分离；契约唯一；版本化迁移 |
| 可移植性 | 单容器部署；环境变量配置；不写死域名/网络名 |
| 兼容性 | 现代浏览器（Chrome/Edge/Firefox/Safari 最近两个大版本）；移动端可用 |

---

## 4. 明确不做（V2.0）

- 多用户、多租户、复杂角色权限。
- 通过 Docker Socket 的网页内"一键升级服务器"。
- 对接外部 AI 生成标签/内容。
- 复杂全文搜索服务、数据分析平台。

---

## 5. 需求到交付阶段映射

| 需求组 | 主要阶段 |
| --- | --- |
| F-01 ~ F-11 | P2（接口）+ P3（页面）+ P4（体验） |
| A-01 ~ A-10 | P2 + P3 |
| A-11 | P2 + P3（P1 增强） |
| D-01 ~ D-06 | P2 |
| D-07 ~ D-08 | P5 |
