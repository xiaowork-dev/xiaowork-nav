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
| `--local` | 不拉取镜像，使用本地已存在的镜像（离线/自建场景） |
| `--yes` | 跳过确认提示 |

脚本执行流程：

1. 读取当前版本与**数据库结构版本**（`/api/v1/version` 的 `schema_version`）；
2. 停止应用，对数据目录做**一致性快照**（数据库 + 上传图标 + 备份）；
3. 原子写入新的 `APP_VERSION` 并启动新镜像；
4. 等待 `/healthz`，并核对结构版本未倒退；
5. 失败时：若新版本已执行数据库迁移，先**用快照恢复数据**，再切回旧镜像。

> 更新不会删除或覆盖你的数据库、图标与设置。

### 1.2 手动更新

```bash
cd deploy
./../scripts/backup.sh                       # 先备份（强烈建议）
sed -i 's/^APP_VERSION=.*/APP_VERSION=v2.0.1/' .env
docker compose pull
docker compose up -d
curl -fsS http://127.0.0.1:8787/healthz
```

### 1.3 固定版本

生产环境建议固定语义化版本而不是 `latest`：

```dotenv
APP_VERSION=v2.0.1
```

`latest` 会跟随最新稳定版，回滚时需要显式指定旧版本。

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

---

## 3. 恢复

```bash
./scripts/restore.sh /path/to/backup-2.0.0-20261025-101500.tar.gz
```

脚本会：停止应用 → **先把当前数据另存一份**（可反悔）→ 解包覆盖 →
启动并等待健康检查。

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
# 1) 恢复更新前的快照（脚本会自动保存当前数据）
./scripts/restore.sh <pre-update 快照路径>

# 2) 切回旧版本镜像
cd deploy
sed -i 's/^APP_VERSION=.*/APP_VERSION=v2.0.0/' .env
docker compose up -d
```

`./scripts/update.sh` 在失败回滚时会**自动完成上述两步**。

### 4.3 如何判断是否发生了迁移

比较更新前后的 `schema_version`：

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

## 6. 自动更新（可选）

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

## 7. 常见问题

| 现象 | 处理 |
| --- | --- |
| 更新后容器反复重启，日志提示结构版本过新 | 说明回滚到了旧镜像。按 4.2 恢复快照，或切回匹配的镜像版本 |
| `compose pull` 失败 | 检查网络与 GHCR 访问；或使用 `--local` 与本地镜像 |
| 更新后数据"消失" | 确认 `XN_DATA_HOST_DIR` 未变；`./scripts/restore.sh` 可恢复 |
| 磁盘空间不足 | 清理旧快照（`--keep`）与未使用的镜像（`docker image prune`） |
| 想跳过确认 | 加 `--yes`（请确保已备份） |
