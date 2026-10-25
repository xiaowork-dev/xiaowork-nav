# XiaoWork Nav V2.0：数据库设计与迁移策略

> 文档类型：数据设计
> 更新日期：2026-10-24
> 数据库：SQLite 3
> 关联：`api-spec.md`、`architecture.md`、`decisions/ADR-0003-sqlite-migrations.md`

---

## 1. 设计原则

1. **单实例足够**：单管理员、单容器场景，SQLite 免运维、备份简单。
2. **ID 长期稳定**：分类与导航项使用自增整数主键，导入导出保留 ID 语义（可映射）。
3. **排序可持久化**：所有可排序实体带 `sort_order` 整数字段。
4. **时间统一**：`created_at`/`updated_at` 存 UTC（RFC 3339 文本或 Unix 秒），对外输出 RFC 3339。
5. **约束前置**：唯一性、外键、非空尽量在数据库层约束。
6. **不可逆操作可控**：迁移与恢复都基于版本化快照。

---

## 2. 表结构

### 2.1 `users`

| 列 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | INTEGER | PK AUTOINCREMENT | |
| username | TEXT | NOT NULL UNIQUE | 3–32 字符 |
| password_hash | TEXT | NOT NULL | Argon2id |
| created_at | TEXT | NOT NULL | RFC 3339 UTC |
| updated_at | TEXT | NOT NULL | RFC 3339 UTC |

> V2.0 仅支持单管理员；表结构保留扩展空间，但不实现多用户/角色。

### 2.2 `sessions`

| 列 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | TEXT | PK | 随机会话标识（存入 Cookie） |
| user_id | INTEGER | NOT NULL, FK→users(id) ON DELETE CASCADE | |
| created_at | TEXT | NOT NULL | |
| expires_at | TEXT | NOT NULL | 过期时间 |
| last_seen_at | TEXT | NOT NULL | 滑动续期 |
| user_agent | TEXT | | 审计辅助 |
| ip | TEXT | | 审计辅助（不存敏感信息） |

索引：`idx_sessions_user_id`、`idx_sessions_expires_at`。

### 2.3 `categories`

| 列 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | INTEGER | PK AUTOINCREMENT | |
| name | TEXT | NOT NULL | 1–64 字符 |
| description | TEXT | NOT NULL DEFAULT '' | ≤ 512 |
| sort_order | INTEGER | NOT NULL DEFAULT 0 | 升序 |
| is_visible | INTEGER | NOT NULL DEFAULT 1 | 0/1 |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

索引：`idx_categories_sort_order`、`idx_categories_is_visible`。

### 2.4 `bookmarks`

| 列 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | INTEGER | PK AUTOINCREMENT | |
| category_id | INTEGER | NOT NULL, FK→categories(id) ON DELETE CASCADE | |
| title | TEXT | NOT NULL | 1–128 |
| url | TEXT | NOT NULL | 仅 http/https，≤ 2048 |
| description | TEXT | NOT NULL DEFAULT '' | ≤ 512 |
| icon_type | TEXT | NOT NULL DEFAULT 'none' | `none`/`url`/`upload`/`auto`/`favicon`/`simple`/`text` |
| icon_value | TEXT | NOT NULL DEFAULT '' | 依类型，≤ 2048；`auto` 存站内缓存路径 |
| keywords | TEXT | NOT NULL DEFAULT '' | ≤ 512，逗号分隔 |
| tags | TEXT | NOT NULL DEFAULT '' | ≤ 256（迁移 0002） |
| color | TEXT | NOT NULL DEFAULT '' | #RGB / #RRGGBB（迁移 0002） |
| sort_order | INTEGER | NOT NULL DEFAULT 0 | 分类内升序 |
| is_visible | INTEGER | NOT NULL DEFAULT 1 | |
| open_in_new_tab | INTEGER | NOT NULL DEFAULT 1 | |
| icon_source | TEXT | NOT NULL DEFAULT '' | 自动图标来源：`manual`/`link-icon`/`apple-touch`/`manifest`/`favicon-ico`/`brand-guess`（迁移 0003） |
| icon_status | TEXT | NOT NULL DEFAULT '' | 最近一次自动获取结果：`ok`/`failed`（迁移 0003） |
| icon_error | TEXT | NOT NULL DEFAULT '' | 最近一次失败原因（迁移 0003） |
| icon_checked_at | TEXT | NOT NULL DEFAULT '' | 最近一次自动获取时间，用于刷新策略（迁移 0003） |
| created_at | TEXT | NOT NULL | |
| updated_at | TEXT | NOT NULL | |

索引：`idx_bookmarks_category_id`、`idx_bookmarks_sort_order`、`idx_bookmarks_is_visible`、`idx_bookmarks_icon_status`。

> 删除分类时级联删除其导航项；接口层需在删除前给出影响数量确认（见 `api-spec.md` 6.2）。
> `icon_type` 为 `url`/`upload`/`simple`/`text` 时属于**用户手动设置**，自动图标流程永不覆盖。

### 2.5 `site_settings`

| 列 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| key | TEXT | PK | 设置键 |
| value_json | TEXT | NOT NULL | JSON 序列化值 |
| updated_at | TEXT | NOT NULL | |

> 键值化存储，避免频繁加列；公开/私有设置在服务端过滤（见 `api-spec.md` 附录 B）。

### 2.6 `uploads`

| 列 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | INTEGER | PK AUTOINCREMENT | |
| filename | TEXT | NOT NULL UNIQUE | 存储文件名 |
| original_name | TEXT | | 原始文件名 |
| mime | TEXT | NOT NULL | |
| size | INTEGER | NOT NULL | 字节 |
| source_url | TEXT | NOT NULL DEFAULT '' | 自动抓取的来源 URL；手动上传为空（迁移 0003） |
| origin | TEXT | NOT NULL DEFAULT 'manual' | `manual`（用户上传）/ `auto`（服务端缓存）（迁移 0003） |
| created_at | TEXT | NOT NULL | |

索引：`idx_uploads_source_url`（按来源去重复用缓存）。

> 文件实体存 `/data/uploads`；数据库仅存元数据，便于导出与清理。
> `origin=auto` 的缓存文件只在"替换后无任何网站引用"时才被清理；`manual` 文件永不自动删除。

### 2.7 `schema_migrations`

| 列 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| version | INTEGER | PK | 迁移序号 |
| name | TEXT | NOT NULL | 迁移名 |
| applied_at | TEXT | NOT NULL | |

---

## 3. 关系与完整性

```text
users 1 ── n sessions
categories 1 ── n bookmarks   (ON DELETE CASCADE)
uploads  (独立，被 bookmarks.icon_value 以文件名引用)
site_settings (键值)
schema_migrations (迁移记录)
```

- 启动时执行 `PRAGMA foreign_keys = ON`。
- 连接使用 WAL 模式：`PRAGMA journal_mode=WAL;`，并设置 `busy_timeout`。

---

## 4. 迁移策略

### 4.1 机制

- 迁移文件为版本化 SQL，命名 `NNNN_<name>.up.sql`（如需回滚：`NNNN_<name>.down.sql`）。
- 通过 Go `embed` 打包进二进制，**运行时不依赖外部迁移工具**。
- 启动时：读取 `schema_migrations` → 按序应用未执行的迁移 → 每个迁移在**单事务**中执行 → 记录版本。
- 迁移器先获取写锁，避免多进程并发迁移（单容器场景主要为防御性）。

### 4.2 目录与示例

```text
backend/internal/migrations/
├── 0001_init.up.sql
├── 0001_init.down.sql
├── 0002_icons_and_labels.up.sql
├── 0002_icons_and_labels.down.sql
├── 0003_icon_metadata.up.sql
└── 0003_icon_metadata.down.sql
```

| 版本 | 名称 | 内容 |
| --- | --- | --- |
| 0001 | `init` | 2.1–2.7 全部表与索引 |
| 0002 | `icons_and_labels` | `categories.icon/color`、`bookmarks.color/tags` |
| 0003 | `icon_metadata` | 自动图标元数据（见 §2.4、§2.6）与两个索引 |

### 4.3 规则

1. **只增不改**：已发布迁移不可修改，变更必须新增迁移文件。
2. **向后兼容优先**：新增列需有默认值；删除列先弃用再删。
3. **可逆性分级**：
   - 可逆（加表/加列/加索引）：提供 `down`，支持切回旧镜像。
   - 不可逆（删列/改类型/数据重写）：必须依赖**升级前快照**恢复，而非 `down`。
4. **失败即停**：迁移失败时服务**拒绝启动**并输出明确错误，避免半迁移状态运行。
5. **幂等**：`schema_migrations` 判定避免重复执行；`down` 仅用于开发与可控回滚。

### 4.4 升级流程（与部署脚本联动）

```text
1. update.sh 检查版本与磁盘空间
2. 在线备份（VACUUM INTO / Backup API）→ /data/backups/<version>-<timestamp>.db
3. 备份上传目录（/data/uploads）
4. 拉取新镜像并切换版本
5. 容器启动 → 自动执行迁移
6. 健康检查 /healthz + 核心接口冒烟
7. 失败：可逆迁移→切回旧镜像；不可逆迁移→用升级前快照恢复
```

### 4.5 备份策略

- **在线备份**：使用 SQLite `VACUUM INTO 'file'` 或 Online Backup API，**不直接热拷贝 WAL 数据库单文件**。
- **内容**：数据库快照 + `/data/uploads` + 版本与时间戳元数据。
- **保留**：自动保留最近 N 份（可配置，默认 5），超出按时间淘汰。
- **恢复**：校验快照完整性后替换数据文件，重启服务；恢复前对当前数据再备份一次。

---

## 5. 数据导入导出

- 导出格式（`xiaowork-nav-v1`）：
  ```json
  {
    "format": "xiaowork-nav-v1",
    "schema_version": 1,
    "exported_at": "2026-10-24T08:31:48Z",
    "checksum": "sha256:...",
    "categories": [ /* ... */ ],
    "bookmarks": [ /* ... */ ],
    "settings": { /* ... */ }
  }
  ```
- 导入流程：解析 → 校验格式与校验和 → **预览**（新增/更新/冲突统计）→ 选择冲突策略（`skip`/`overwrite`/`rename`）→ 单事务执行。
- 旧站导入：`format: "legacy-v1"`，由转换器映射旧字段；**必须先获得真实 `data.json` 样本**再实现映射，不得凭空假设。
- 导入**默认不覆盖现有数据**，冲突需显式策略。

---

## 6. 安全与容量

- 所有文本输出经转义；URL 协议白名单。
- 上传图标限制类型与体积（≤ 1 MiB）。
- 数据库与备份文件不进入静态目录。
- SQLite 适合单实例；当出现多实例/高并发需求时再评估 PostgreSQL（V2.0 不做）。
