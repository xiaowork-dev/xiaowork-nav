# XiaoWork Nav v2.1.3 — ASCII Pond 可选主题正式发布交付报告

## 1. 发布概况

| 字段 | 实际结果 |
| --- | --- |
| 项目 | XiaoWork Nav |
| 版本 | `v2.1.3` |
| 发布日期 | 2026-10-28（Asia/Taipei；GitHub published_at=2026-10-27T22:16:39Z） |
| 发布类型 | 功能增强：新增可选前端主题 |
| 数据库结构 | **未变更**，仍为 schema_version=3 |
| PR | [#1 — ASCII Pond 主题](https://github.com/xiaowork-dev/xiaowork-nav/pull/1)（已合并） |
| 合并源码提交 | `093a78aa320a88c73d872830acddf9531497e7b3` |
| 版本标签 | [v2.1.3](https://github.com/xiaowork-dev/xiaowork-nav/tree/v2.1.3)（指向上述合并提交） |
| GitHub Release | [XiaoWork Nav v2.1.3](https://github.com/xiaowork-dev/xiaowork-nav/releases/tag/v2.1.3)（非草稿、非预发布） |
| 多架构镜像 | `ghcr.io/xiaowork-dev/xiaowork-nav:v2.1.3` |
| Release 工作流 | [40593127336](https://github.com/xiaowork-dev/xiaowork-nav/actions/runs/40593127336)：**completed / success** |

## 2. 本版本交付内容

- 新增“锦鲤池塘”第三套**浏览器本地可选主题**，保留深色、浅色与原有导航双布局。
- 引入 ASCII Pond 原版 JavaScript 动画，按需加载，鼠标搅动水面及空白区域点击投喂。
- 主题切走后销毁动画实例，用户选择写入 `xn-theme`，不扩展后端设置枚举。
- 修复手机顶部主题选择控件溢出、首帧背景闪烁、锦鲤主题下占位字母对比度不足的问题。
- 加入测试覆盖：主题切换持久化、懒加载、快速切换竞态与资源清理。
- Docker CI 增加 `/vendor/ascii-pond/{pond,koi,util}.js` 和 `LICENSE` 资源 HTTP 200、JavaScript MIME 校验。
- 保留第三方 MIT 许可证；没有业务接口、后端逻辑或数据库迁移变更。

完整设计、问题修复、浏览器验收与性能数据见 [ASCII Pond 验收记录](../20261028-ascii-pond-theme.md)。

## 3. 验证结果

| 检查 | 结果 |
| --- | --- |
| PR #1 最终 CI | [40590976454](https://github.com/xiaowork-dev/xiaowork-nav/actions/runs/40590976454) 全部通过 |
| 合并后 main CI | [40592576052](https://github.com/xiaowork-dev/xiaowork-nav/actions/runs/40592576052) 全部通过 |
| 发布工作流前置验证 | 通过 |
| 原生 AMD64 镜像构建与推送 | 通过 |
| 原生 ARM64 镜像构建与推送 | 通过 |
| 多架构 manifest 发布 | 通过 |
| GitHub Release 创建 | 通过 |
| 正式生产服务器更新 | **未执行**，由维护者自行升级 |
| 真实设备人眼视觉与 GPU 帧率 | **尚未执行**，以实际部署后人工验收为准 |

Release 工作流采用版本 tag 触发的同一套发布任务；本次通过在独立一次性分支创建经过验证的标签并以 `workflow_dispatch` 触发原有 `release.yml`，**没有修改正式版本源码或改变发布镜像的来源提交**。此前一次性工作流首个提交因 YAML 格式错误未执行任务，经修复后 [dispatcher](https://github.com/xiaowork-dev/xiaowork-nav/actions/runs/40593107041) 运行成功；失败的首次触发没有创建 tag 或发布镜像。

## 4. 源码与版本控制

- PR #1 已合并到 `main`：`093a78a`。
- Git 标签 `v2.1.3` 标记合并提交；自动生成的正式镜像来自该提交。
- `main` 的后续发布文档更新不改变 `v2.1.3` 已发布源码。
- 正式 release 和多架构镜像已完成；用户现有服务器仍需手动更新。

## 5. 现有服务器一键升级

在**确认位于真实现有安装目录、仓库工作区无未提交改动**后：

```bash
cd /opt/xiaowork-nav
git status --short
git branch --show-current
git pull --ff-only
./scripts/update.sh v2.1.3
```

升级脚本会在更新前进行数据快照，包含故障回滚逻辑。**不要**执行重新安装、`docker compose down -v` 或删除生产数据卷。

如部署使用 `compose.npm.yaml`，使用实际 Compose 模式：

```bash
./scripts/update.sh v2.1.3 --compose compose.npm.yaml
```

更新后验证：

```bash
docker inspect -f '{{.State.Health.Status}}' xiaowork-nav
docker logs --tail=40 xiaowork-nav
```

如有端口绑定、容器名或 NPM 模式等自定义配置，以现有 `deploy/.env` 和实际部署为准。浏览器旧资源缓存可用强制刷新处理。主题需在页面顶部选择“锦鲤池塘”，默认旧主题保持原样。

## 6. 风险与下一步

1. CI/软件渲染已验证，但真实 GPU、iOS Safari、微信浏览器和触摸交互未完整验收。
2. 生产部署与数据量验收尚未执行；先验证站点、后台、搜索及图标正常，再检查锦鲤动画。
3. 如生产更新失败，保留 update.sh 生成的快照和日志并遵循 [回滚指南](../../updating.md)。
4. 本发布报告属于正式版发布后的归档记录，不构成生产部署成功声明。
