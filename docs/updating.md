# 更新、备份与回滚指南

> 适用版本：XiaoWork Nav 2.0.x
> 关联：[安装指南](installation.md)、[数据库设计](database-design.md)

---

## 1. 更新

### 1.1 一键更新（推荐）

```bash
cd xiaowork-nav
./scripts/update.sh v2.0.1      # 指定版本
./scripts/update.sh latest      # 最新稳定版
```

可用参数：

| 参数 | 说明 |
| --- | --- |
| `--compose <file>` | 使用 `compose.npm.yaml` 等替代编排文件 |
| `--local` / `--no-pull` | 不拉取镜像，使用本地已存在的镜像（离线/自建场景） |
| `--yes` | 跳过确认提示 |

脚本执行流程：

1. 判断目标标签是否可变：
   - **固定版本**（`vX.Y.Z`）且与当前版本相同 → 直接退出（幂等）；
   - **可变标签**（`latest`、`v2.0` 等）→ **总是向仓库查询并拉取**最新镜像。
2. 可变标签拉取后若镜像摘要未变、且运行中的容器正是该镜像且健康 → 报告
   "无需操作"并退出，**不重建容器**；
3. 读取当前版本与**数据库结构版本**（见 1.2）；
4. 停止应用，对数据目录做**一致性快照**（数据库 + 上传图标 + 备份）；
5. 原子写入新的 `APP_VERSION` 并启动新镜像；
6. 等待 `/healthz`，并核对结构版本未倒退、书签数量未异常变化；
7. 失败时自动回滚（见 §4）。

> 更新不会删除或覆盖你的数据库、图标与设置。

### 1.2 结构版本如何读取

脚本**直接从数据库文件**读取结构版本与行数，命令等价于：

```bash
docker run --rm -v "$XN_DATA_HOST_DIR:/data" <镜像> --db-info --db /data/app.db
# db_path=/data/app.db
# exists=true
# schema_version=2
# categories=10
# bookmarks=110
# uploads=31
```

这样做的原因：

- **不依赖宿主机端口**：NPM 共享网络模式不发布宿主机端口，无法用 HTTP 查询；
- **容器停止时也可读**：更新失败后需要先判断结构版本再决定是否回滚，此时应用
  往往已无法启动；
- **只读**：`--db-info` 只执行 `SELECT`，不会创建表或写入任何数据。

`update.sh` 只用这一种方式判断结构版本；若无法读取（例如镜像损坏），会按
"兼容性未知"处理并采用最保守的回滚路径，而不会改用 HTTP 或猜测。

### 1.3 手动更新

```bash
cd deploy
./../scripts/backup.sh                       # 先备份（强烈建议）
sed -i 's/^APP_VERSION=.*/APP_VERSION=v2.0.1/' .env
docker compose pull
docker compose up -d
curl -fsS http://127.0.0.1:8787/healthz
```

### 1.4 固定版本

生产环境建议固定语义化版本而不是 `latest`：

```dotenv
APP_VERSION=v2.0.1
```

- 完整的 `vX.Y.Z` 被视为**不可变**：`update.sh v2.0.1` 在已是该版本时立即退出。
- `latest`、`v2.0` 等**可变**标签会始终检查更新，回滚时需要显式指定旧版本。

---

## 2. 备份

### 2.1 应用内备份（无需停机）

管理后台 →「数据管理」→「立即备份」。应用使用 SQLite 的 `VACUUM INTO`
生成一致性快照，并把上传图标一并保存到 `/data/backups`。
保留数量由 `XN_BACKUP_KEEP` 控制。

### 2.2 命令行备份（整目录）

```bash
./scripts/backup.sh              # 停止 → 打包 → 启动
./scripts/backup.sh --keep 10    # 同时清理旧快照，保留最近 10 份
./scripts/backup.sh --out /mnt/backups
```

备份内容：

```text
app.db            SQLite 数据库
uploads/          图标文件
backups/          应用内快照
```

> 脚本会短暂停止应用（数秒），以确保 WAL 模式下的数据库被完整捕获。
> 直接热拷贝正在使用的 `app.db` 可能得到损坏的备份。

快照文件名包含应用版本与时间戳，例如：

```text
backup-v2.0.0-20261025-101500.tar.gz
pre-update-v2.0.0-20261025-101500.tar.gz
```

`restore.sh` 会从文件名推断该快照属于哪个版本（见 §3）。

---

## 3. 恢复

```bash
./scripts/restore.sh /path/to/backup-v2.0.0-20261025-101500.tar.gz
./scripts/restore.sh <快照> --version v2.0.0    # 显式指定快照所属版本
./scripts/restore.sh <快照> --compose compose.npm.yaml --yes
./scripts/restore.sh <快照> --no-start          # 只恢复数据，不启动
```

脚本会：

1. 校验快照是可读的 `tar.gz`；
2. 解析快照所属版本：优先 `--version`，否则从文件名推断
   （`backup-<版本>-<时间戳>.tar.gz`、`pre-update-<版本>-<时间戳>.tar.gz`）；
3. 停止应用，**先把当前数据另存一份**（可反悔）；
4. 解包覆盖；
5. **在启动前**把 `APP_VERSION` 切换到快照所属版本 —— 否则当前配置的新镜像
   会立刻对刚恢复的旧库重新执行迁移；
6. 启动并等待健康检查，打印恢复后的结构版本与行数。

> **版本未知时拒绝启动**：若既没有 `--version`，也无法从文件名推断版本，
> 脚本只恢复数据、保持应用停止，并打印手动启动指引，而**不会**猜测。

也可以使用管理后台的「备份与恢复」界面，从 `/data/backups` 中选择快照恢复
（恢复前同样会自动生成安全快照）。

---

## 4. 回滚

### 4.1 只回滚镜像（未发生数据库迁移）

```bash
cd deploy
sed -i 's/^APP_VERSION=.*/APP_VERSION=v2.0.0/' .env
docker compose up -d
```

适用条件：新版本**没有**升级数据库结构版本。旧镜像可以正常读取旧结构。

### 4.2 迁移后回滚（必须同时恢复数据）

若新版本执行了数据库迁移，**不能只用旧镜像启动**。从 2.0.0 起，应用启动时会
校验结构版本：

```
ERROR database schema is newer than this build; refusing to start
      database_schema=3 build_schema=2
      hint="restore the pre-update backup or deploy a matching image version"
```

这是**有意设计的安全保护**：避免旧代码用错误的表结构读写数据。容器会持续重启
失败，而不会破坏数据。

正确的回滚步骤：

```bash
# 恢复更新前的快照，并把版本切回快照所属版本后再启动
./scripts/restore.sh <pre-update 快照路径> --version v2.0.0
```

`./scripts/update.sh` 在失败回滚时会**自动完成**上述步骤。

### 4.3 更新失败时的自动回滚保证

`update.sh` 在下列任一情况都会触发回滚，并**再次验证**回滚结果：

| 失败场景 | 处理 |
| --- | --- |
| 新镜像拉取失败 | 本地已有该镜像则继续；否则**不改动任何东西**并退出 |
| 新容器启动失败（`compose up -d` 返回非 0） | 回滚 |
| 新容器健康检查失败 | 回滚 |
| 新版本已迁移数据库，但随后启动/健康检查失败 | **先用快照恢复数据**，再切回旧镜像 |
| 无法读取数据库结构版本（HTTP 与文件都失败） | 视为"兼容性未知"，**先用快照恢复数据**再回滚（保守策略） |
| NPM 共享网络模式升级失败 | 同上（不依赖宿主机端口，因此行为一致） |
| 结构版本倒退（新 < 旧） | 回滚 |

回滚流程：

1. 读取当前（可能已被迁移的）结构版本；
2. 若结构版本**变大**、**无法读取**、或**更新前版本未知** → 先用更新前快照恢复数据；
3. 把 `APP_VERSION` 切回旧版本并启动；
4. **再次验证**：等待健康检查、核对结构版本与书签行数，而不是只看
   `docker compose up -d` 的退出码；
5. 打印快照路径供人工介入。

> 无论哪条路径，**更新前的一致性快照都不会被删除**。

### 4.4 如何判断是否发生了迁移

优先用数据库文件（任意模式、容器停止时也可用）：

```bash
docker run --rm -v "$XN_DATA_HOST_DIR:/data" <镜像> --db-info --db /data/app.db
```

或独立模式下：

```bash
curl -fsS http://127.0.0.1:8787/api/v1/version
```

- 相同 → 可直接回滚镜像（4.1）
- 变大 → 必须同时恢复数据（4.2）

`update.sh` 会打印 `database schema version: X -> Y`。

---

## 5. 迁移兼容性说明

- 迁移脚本**只增不改**：已发布的迁移文件不会修改。
- 新增列一律带默认值，旧数据仍可读取。
- 迁移在**单个事务**内执行；失败则回滚并拒绝启动，不会留下半迁移状态。
- 结构版本只增不减：**新版本可以读旧数据，旧版本不能读新数据**。
- 因此升级是安全的，降级必须配合快照恢复。

---

## 6. 异常注入测试

部署脚本的关键失败路径有可提交的自动化测试：

```bash
./scripts/tests/deploy-faults.test.sh
```

该测试会在临时目录中搭建隔离部署，构建一次性镜像，并覆盖：

1. 固定版本已部署 → 幂等退出；
2. `latest` 镜像未变化 → 不重建容器；
3. 目标镜像不可用 → 不改动任何东西；
4. 容器启动失败 → 回滚并验证；
5. 迁移后健康检查失败 → 先恢复快照再回滚；
6. 应用停止时仍能读取结构版本（NPM 模式无宿主机端口）；
7. `restore.sh` 启动前切换到快照版本；
8. 快照版本未知时 `restore.sh` 拒绝启动。

需要 Docker；构建镜像时可设置 `XN_TEST_GOPROXY` 指定 Go 代理。

---

## 7. 自动更新（可选）

默认是手动更新。如需无人值守更新，可在宿主机用 systemd timer 调用同一套脚本：

```ini
# /etc/systemd/system/xiaowork-nav-update.service
[Unit]
Description=XiaoWork Nav automatic update

[Service]
Type=oneshot
WorkingDirectory=/opt/xiaowork-nav
ExecStart=/opt/xiaowork-nav/scripts/update.sh latest --yes
```

```ini
# /etc/systemd/system/xiaowork-nav-update.timer
[Unit]
Description=Check for XiaoWork Nav updates daily

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
systemctl enable --now xiaowork-nav-update.timer
```

建议：仅在测试环境验证过新版本后再启用；生产环境保持手动更新，以便确认发行说明。

> 不建议为了在网页内升级而挂载 `/var/run/docker.sock`，那会让应用接近拥有宿主 Docker 管理权限。

---

## 8. 常见问题

| 现象 | 处理 |
| --- | --- |
| 更新后容器反复重启，日志提示结构版本过新 | 说明回滚到了旧镜像。按 4.2 恢复快照，或切回匹配的镜像版本 |
| `latest` 更新提示"无需操作" | 镜像摘要未变，属正常行为；如确认上游已发布新版本，检查是否推送到了同一标签 |
| `compose pull` 失败 | 检查网络与 GHCR 访问；或使用 `--local` 与本地镜像 |
| 更新后数据"消失" | 确认 `XN_DATA_HOST_DIR` 未变；`./scripts/restore.sh` 可恢复 |
| 升级后容器启动即退出，日志提示目录不可写 | 数据目录属主与容器用户不一致；见[安装指南 §4](installation.md)。可运行 `./scripts/update.sh` 重新修复，或手动 `chown`/设置 `XN_UID` |
| `restore.sh` 提示"refusing to start" | 快照文件名未包含版本。用 `--version vX.Y.Z` 指定后重跑 |
| 磁盘空间不足 | 清理旧快照（`--keep`）与未使用的镜像（`docker image prune`） |
| 想跳过确认 | 加 `--yes`（请确保已备份） |
