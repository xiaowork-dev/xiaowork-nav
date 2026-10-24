# XiaoWork Nav 旧站数据迁移报告

> 文档类型：数据迁移验证报告（P3 收尾）
> 生成日期：2026-10-24
> 数据来源：`data.json`（90,426 字节，用户提供的旧版 XiaoWork Nav 导出文件）
> 状态：**已在独立测试数据库中完成导入并验证**；正式导入方案见第 9 节
> 关联：`docs/samples/legacy-data.sample.json`（脱敏样本）、`docs/api-spec.md`、`docs/openapi/openapi.yaml`

---

## 0. 结论摘要

| 项 | 结果 |
| --- | --- |
| 旧数据结构 | 顶层**数组**，10 个分类 / 110 个站点 |
| 字段识别 | 分类 3 个字段、条目 9 个字段全部识别 |
| 分类顺序 | ✅ 完整保留（`sort_order` 10→100） |
| 站点排列顺序 | ✅ 完整保留（分类内 `sort_order` 10,20,30…） |
| 外部图标（19 个） | ✅ **原样保留**，未替换为默认 Favicon |
| 内嵌 SVG 图标（31 个） | ✅ 原样提取为本地文件（内容逐字节保留） |
| 文本型图标（3 个） | ⚠️ 无法作为图片，已设为无图标并把文本保留到关键词（见 8.1） |
| 无效网址 | 0 个 |
| 重复网址 | 1 个（跨分类，按设计保留两条） |
| 导入结果 | 分类 10 / 站点 110 / 跳过 0 |
| 浏览器验证 | ✅ 10 个分类顺序与数量正确；49 个图标正常渲染，61 个回退为文字图标 |
| 需人工处理 | 6 类，见第 8 节 |

**原始文件未被修改。** 真实数据文件不会提交到公开仓库（见第 2 节）。

---

## 1. 数据来源与安全声明

- 源文件：`data.json`（本地路径，**未复制进仓库**）。
- 本报告中的分类名、站点名与图标地址用于说明迁移结果；真实文件本身不进入版本库。
- `.gitignore` 已新增规则，阻止 `data.json`、`legacy-data.json`、`*.private.json` 被提交。
- 测试使用**脱敏样本** `docs/samples/legacy-data.sample.json`（分类/站点名与 URL 均为示例数据）。
- 所有分析脚本均以只读方式打开源文件，未做任何写入或覆盖。

---

## 2. 旧数据结构分析

### 2.1 顶层结构

```jsonc
[
  {
    "category": "Daily",     // 分类名
    "icon": "📁",            // 分类图标（emoji）
    "items": [ /* 站点数组 */ ]
  }
  // … 共 10 个分类
]
```

顶层是**数组**（不是 `{ "categories": [...] }` 对象），这是现有 `legacy-v1` 导入器此前不支持的结构。

### 2.2 字段使用统计

**分类字段**（10 个分类全部使用）

| 字段 | 出现次数 | 说明 |
| --- | --- | --- |
| `category` | 10 | 分类名（`邮箱 `、`域名 ` 含尾随空格） |
| `icon` | 10 | 分类 emoji（📁 / ✨ / 🌐 / 🤖 / 🐔 / 📧 / 💻 / 🔗） |
| `items` | 10 | 站点数组 |

**条目字段**（共 110 条）

| 字段 | 出现次数 | 说明 |
| --- | --- | --- |
| `name` | 110 | 站点名称 |
| `url` | 110 | 站点地址 |
| `desc` | 110 | 描述 |
| `logo` | 54 | 图标值（URL 或 data URI 或文本） |
| `logoType` | 33 | 值为 `svg`，标记图标为 SVG |
| `si` | 27（8 个非空） | Simple Icons 图标标识，如 `github`、`vercel` |
| `color` | 6 | 主题色，如 `#2D72D9` |
| `tag` | 4 | 标签，如 `Info`、`Art`、`Deploy`、`Q&A` |
| `iconType` | 1 | 值为 `favicon` |

**条目字段组合**（7 种）

| 组合 | 数量 |
| --- | --- |
| `desc,name,url` | 28 |
| `desc,name,si,url` | 22 |
| `desc,logo,logoType,name,url` | 33 |
| `desc,logo,name,url` | 21 |
| `color,desc,name,si,tag,url` | 3 |
| `color,desc,name,si,url` | 2 |
| `color,desc,iconType,name,tag,url` | 1 |

### 2.3 分类清单（源文件顺序）

| 序号 | 分类 | 图标 | 站点数 |
| --- | --- | --- | --- |
| 1 | Daily | 📁 | 17 |
| 2 | SpeedTest | 📁 | 8 |
| 3 | 常用 | ✨ | 7 |
| 4 | IP&DNS | 🌐 | 14 |
| 5 | AI & 智能 | 🤖 | 10 |
| 6 | VPS/云服务器 | 🐔 | 26 |
| 7 | 邮箱（尾随空格） | 📧 | 13 |
| 8 | 开发 & 编程 | 💻 | 3 |
| 9 | 域名（尾随空格） | 🔗 | 11 |
| 10 | 其他导航 | 📁 | 1 |
| | **合计** | | **110** |

### 2.4 图标构成

| 类型 | 数量 | 说明 |
| --- | --- | --- |
| 无图标 | 57 | 条目没有 `logo` 字段或为空 |
| 内嵌 data URI | 31 | 全部为 `data:image/svg+xml,...`（百分号编码 SVG），长度 592–11,635 字符（平均 2,169，9 个超过 2,048） |
| 外部图片 URL | 19 | `.webp` 10、`.png` 6、`.svg` 2、无扩展名 1 |
| 文本型图标 | 3 | `延迟`、`地址`、`ip`（短文本当作图标显示） |

---

## 3. 字段映射表

| 旧站字段 | 新字段 | 规则 | 状态 |
| --- | --- | --- | --- |
| `category` | `Category.name` | 去除首尾空格 | ✅ 完整 |
| 数组下标 | `Category.sort_order` | `(下标+1)×10`，保留原顺序 | ✅ 完整 |
| `icon`（分类 emoji） | — | 当前数据模型没有分类图标字段 | ⚠️ 未映射（见 8.5） |
| `items` 数组下标 | `Bookmark.sort_order` | `(下标+1)×10`，保留原顺序 | ✅ 完整 |
| `name` | `Bookmark.title` | 必填 | ✅ 完整 |
| `url` | `Bookmark.url` | 必填，仅接受 http/https | ✅ 完整（0 个非法） |
| `desc` | `Bookmark.description` | | ✅ 完整 |
| `logo`（http/https） | `icon_type=url` + `icon_value` | **原样保留**，不做任何替换 | ✅ 19/19 |
| `logo`（data URI） | `icon_type=upload` + `icon_value=/api/v1/public/uploads/<file>` | 解码后写入本地图标文件，内容逐字节保留 | ✅ 31/31 |
| `logo`（文本） | `icon_type=none` + 文本并入 `keywords` | 无法作为图片 | ⚠️ 3 项（见 8.1） |
| `logoType` | — | 仅用于确认 SVG，无需存储 | ℹ️ 忽略 |
| `si` | 并入 `Bookmark.keywords` | 非图片引用，保留为可搜索关键词 | ⚠️ 8 项（见 8.2） |
| `tag` | 并入 `Bookmark.keywords` | | ⚠️ 4 项（见 8.2） |
| `color` | — | 当前数据模型没有颜色字段 | ⚠️ 未映射（见 8.5） |
| `iconType=favicon` | `icon_type=none` | **不自动抓取/替换**，避免擅自改变图标 | ⚠️ 1 项（见 8.3） |
| `desc` 为空 | 空字符串 | | ✅ |

> ⚠️ **本节部分规则已被第 13 节（P3 收尾增强）取代**：文字型图标现在映射为 `icon_type=text`（不再降级为无图标），`si` 命中已验证的 Simple Icons 标识时映射为 `icon_type=simple`，`tag` 迁移到独立的 `tags` 字段，`color` 迁移到独立的 `color` 字段，分类 `icon`（emoji）迁移到 `categories.icon`。以第 13 节为准。

---

## 4. `legacy-v1` 兼容性检查与代码改进

### 4.1 改进前的兼容情况

| 检查项 | 结果 |
| --- | --- |
| 顶层数组结构 | ❌ 不支持（原实现只接受 `{ "categories": [...] }`） |
| 分类字段 `category` | ❌ 不支持（只认 `name`/`title`） |
| 分类条目字段 `items` | ❌ 不支持（只认 `links`/`bookmarks`/`sites`） |
| 图标字段 `logo` | ❌ 不支持（只认 `icon`/`iconUrl`/`favicon`） |
| `data:` URI 图标 | ❌ 不支持（会被当作非法 URL 丢弃） |
| `si` / `tag` / `color` / `logoType` | ❌ 未处理 |
| 顺序保留 | ⚠️ 依赖数组顺序，但未显式写入 `sort_order` |

**结论：改进前的导入器无法导入本文件，图标会全部丢失。**

### 4.2 本次改进内容

| 改进 | 位置 |
| --- | --- |
| 支持顶层数组与对象两种旧站结构 | `internal/service/transfer.go` `DecodeBundle` |
| 支持 `category` / `items` 字段别名 | `legacyCategory` |
| 支持 `logo` / `logoType` / `si` / `tag` / `color` / `iconType` | `legacyItem` |
| 分类与条目的 `sort_order` 显式保留顺序 | `normalizeLegacy` + `store.insertCategory`（修复了原先硬编码 0 的问题） |
| data URI 图标解码并转为本地上传 | `internal/service/icons.go` `SaveDataURI` |
| 图标分类统计与映射告警 | `sanitizeIcons` / `legacyWarnings` |
| 冲突策略改为支持查询参数（数组请求体无法内嵌字段） | `internal/api/admin_transfer.go` |
| 预览新增图标与非法网址统计 | `models.ImportPreview` + `openapi.yaml` |
| 修正冲突检测：按分类名解析后再比对 URL | `store.PreviewImport` |
| 安全修正：书签引用未知分类时跳过而非猜测 ID | `store.ImportData` |

契约已同步更新（`/admin/import` 的 `conflict_strategy` 查询参数、`ImportPreview` 新增字段、`LegacyCategoryArray` schema），并新增自动化测试覆盖。

---

## 5. 图标处理策略（重点）

用户要求**保留手动挑选的图标**，本次严格遵守：

1. **外部图片地址（19 个）**：原样写入 `icon_value`，`icon_type=url`。**未替换为任何网站默认 Favicon**，未做任何改写（含 `.webp`、`.png`、`.svg` 与无扩展名地址）。
2. **内嵌 SVG（31 个）**：旧站把 SVG 直接以 `data:image/svg+xml,...` 内嵌在数据里。由于：
   - 现行契约 `icon_value` 上限 2,048 字符（9 个超过该上限）；
   - 书签 URL 校验只接受 http/https；
   
   因此**解码后写入本地图标文件**（`/data/uploads/*.svg`），`icon_type=upload`，`icon_value=/api/v1/public/uploads/<file>`。**图标内容逐字节保留**，只是从"内嵌数据"变成"本地文件"。这是唯一可行的保留方式，且不会把 90KB 的 SVG 塞进每次导航响应。
3. **文本型图标（3 个）**：旧站用短文本（`延迟`/`地址`/`ip`）当图标。当前模型不支持文本图标，已设为无图标，并把文本保留到 `keywords`（可搜索），**未擅自替换成 Favicon**。
4. **SVG 安全**：SVG **不以 HTML 形式注入页面**。它们通过 `<img src>` 加载，并由后端以 `image/svg+xml` 提供，附带 `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; sandbox` 响应头（见 6.3 实测）。

### 5.1 图标清单（50 项，逐项可核对）

| 分类 | 名称 | icon_type | 原始图标 | 存储值 |
| --- | --- | --- | --- | --- |
| Daily | cloudflare | upload | 内嵌 image/svg+xml（1838 字符） | uploads/6c5d70f7d37c7da16fd94475b263b40d.svg |
| Daily | komari探针 | url | https://mjj.today/i/lUGk4m | https://mjj.today/i/lUGk4m |
| SpeedTest | speedtest | upload | 内嵌 image/svg+xml（592 字符） | uploads/3f3bfbdb0bbcaab34ee28be18337d4e5.svg |
| SpeedTest | cf测速 | upload | 内嵌 image/svg+xml（1838 字符） | uploads/d55677f37864b972462d5c720ac20bc0.svg |
| 常用 | YouTube | upload | 内嵌 image/svg+xml（719 字符） | uploads/9027c3fdae817a3f5e91e928056736aa.svg |
| 常用 | GitHub | upload | 内嵌 image/svg+xml（1316 字符） | uploads/54b40e867e495b4a75712ebfc321f270.svg |
| 常用 | telegram | upload | 内嵌 image/svg+xml（1584 字符） | uploads/e0d7428e9fcd82e52d58f375dde98f90.svg |
| 常用 | Netflix | upload | 内嵌 image/svg+xml（2568 字符） | uploads/9a86c8a50ae754ee6b1b312f6f43f954.svg |
| 常用 | Spotify | upload | 内嵌 image/svg+xml（1277 字符） | uploads/6f5e8b57d553f8d52b956af41e050e20.svg |
| 常用 | Bilibili | url | https://iili.io/KQ4DCGI.png | https://iili.io/KQ4DCGI.png |
| 常用 | 迪士尼+ | upload | 内嵌 image/svg+xml（4061 字符） | uploads/0b0879707ed9406cf8d5641b6adcaa52.svg |
| IP&DNS | iping | url | https://dh.kejilion.pro/assets/img/img-8eb780138070cf.webp | https://dh.kejilion.pro/assets/img/img-8eb780138070cf.webp |
| IP&DNS | IPLEAK | url | https://ipleak.net/static/images/logotitle.png | https://ipleak.net/static/images/logotitle.png |
| IP&DNS | DNSLEAKTEST | url | https://www.dnsleaktest.com/assets/img/logo.png | https://www.dnsleaktest.com/assets/img/logo.png |
| IP&DNS | IPsuper | url | https://ipsuper.com/_nuxt/logo-icon.DYKBoQ1d.png | https://ipsuper.com/_nuxt/logo-icon.DYKBoQ1d.png |
| AI & 智能 | ChatGPT | upload | 内嵌 image/svg+xml（1979 字符） | uploads/d13932d956f2f0ee8baa40c7d5541f1b.svg |
| AI & 智能 | Gemini | upload | 内嵌 image/svg+xml（5397 字符） | uploads/241260fe03acb23ccecd7960541f8c73.svg |
| AI & 智能 | Claude | upload | 内嵌 image/svg+xml（2137 字符） | uploads/615c713cfd75a148b1d49c11cb3cfa06.svg |
| AI & 智能 | DeepSeek | upload | 内嵌 image/svg+xml（2441 字符） | uploads/e48b4771fc28218d7b456fe45ae3890c.svg |
| AI & 智能 | kimi | upload | 内嵌 image/svg+xml（1206 字符） | uploads/df02141b03f9b7392568bc331d0ef53c.svg |
| AI & 智能 | Manus | upload | 内嵌 image/svg+xml（6925 字符） | uploads/25ae3712bcfdfeaf095003d40dbacb51.svg |
| AI & 智能 | Grok | upload | 内嵌 image/svg+xml（904 字符） | uploads/caa2a2ed41de98fa0b52cab14f40b0f8.svg |
| AI & 智能 | Cursor | upload | 内嵌 image/svg+xml（611 字符） | uploads/56ac2208b16cfd3cbb90438b170a7fdd.svg |
| VPS/云服务器 | 搬瓦工 | url | https://dh.kejilion.pro/assets/img/img-adc3bad3b2967d.webp | https://dh.kejilion.pro/assets/img/img-adc3bad3b2967d.webp |
| VPS/云服务器 | Jtti | url | https://dh.kejilion.pro/assets/img/img-6ed881c962fb60.webp | https://dh.kejilion.pro/assets/img/img-6ed881c962fb60.webp |
| VPS/云服务器 | 谷歌云 | upload | 内嵌 image/svg+xml（1269 字符） | uploads/1f947c09b4cd21d3ea8b84074157a83a.svg |
| VPS/云服务器 | 甲骨文云 | url | https://dh.kejilion.pro/assets/img/img-6224e9c2a298f2.webp | https://dh.kejilion.pro/assets/img/img-6224e9c2a298f2.webp |
| VPS/云服务器 | ColoCrossing | url | https://dh.kejilion.pro/assets/img/img-2bcd58e4adc863.webp | https://dh.kejilion.pro/assets/img/img-2bcd58e4adc863.webp |
| VPS/云服务器 | Hosthatch | url | https://dh.kejilion.pro/assets/img/img-cca56260fa4a46.webp | https://dh.kejilion.pro/assets/img/img-cca56260fa4a46.webp |
| VPS/云服务器 | Evoxt | url | https://dh.kejilion.pro/assets/img/img-3e6219f1b16c87.webp | https://dh.kejilion.pro/assets/img/img-3e6219f1b16c87.webp |
| VPS/云服务器 | 华纳云 | url | https://dh.kejilion.pro/assets/img/img-a23a3c1accb7ea.webp | https://dh.kejilion.pro/assets/img/img-a23a3c1accb7ea.webp |
| VPS/云服务器 | 面包云 | url | https://breadcloud.io/templates/lagom2/assets/img/logo/logo_big.6270810.png | https://breadcloud.io/templates/lagom2/assets/img/logo/logo_big.6270810.png |
| VPS/云服务器 | 阿里云 | upload | 内嵌 image/svg+xml（775 字符） | uploads/d54879a8fc96d481e072afde5c918dbe.svg |
| VPS/云服务器 | 腾讯云 | upload | 内嵌 image/svg+xml（2726 字符） | uploads/bca15949f21606e0e006ca2b4073278d.svg |
| VPS/云服务器 | 华为云 | upload | 内嵌 image/svg+xml（1840 字符） | uploads/6ed3379d34348086dec5c5148274bb18.svg |
| VPS/云服务器 | 莱卡云 | url | https://www.lcayun.com/upload/banner/2023-10/11/169700642979539.png | https://www.lcayun.com/upload/banner/2023-10/11/169700642979539.png |
| 邮箱 | Gmail | upload | 内嵌 image/svg+xml（1105 字符） | uploads/0e3c407df4b5f6b47e5d9ea24d6750d1.svg |
| 邮箱 | Outlook | upload | 内嵌 image/svg+xml（11635 字符） | uploads/17931a64fec76d1777c50d482703fcda.svg |
| 邮箱 | protonmail | upload | 内嵌 image/svg+xml（712 字符） | uploads/83d6f4b4e2891883e59740309bd2f5e9.svg |
| 邮箱 | QQ邮箱 | url | https://dh.kejilion.pro/assets/img/inline-b80804abfab5fa.webp | https://dh.kejilion.pro/assets/img/inline-b80804abfab5fa.webp |
| 邮箱 | Yahoo邮箱 | url | https://cdn.worldvectorlogo.com/logos/yahoo-icon.svg | https://cdn.worldvectorlogo.com/logos/yahoo-icon.svg |
| 域名 | cloudflare | upload | 内嵌 image/svg+xml（1838 字符） | uploads/d6f50c5f6df4682fadf200849309aaea.svg |
| 域名 | GoDaddy | upload | 内嵌 image/svg+xml（1418 字符） | uploads/b17ff9eaf805403cfb7381c316510275.svg |
| 域名 | Namecheap | upload | 内嵌 image/svg+xml（879 字符） | uploads/94832d96ff5d49c4d39a91f60b66653c.svg |
| 域名 | Nameslio | upload | 内嵌 image/svg+xml（894 字符） | uploads/0861bae9d6dc00b2888910ea090bb941.svg |
| 域名 | Gname | url | https://dh.kejilion.pro/assets/img/img-512cc5b8c6de74.webp | https://dh.kejilion.pro/assets/img/img-512cc5b8c6de74.webp |
| 域名 | porkhub | upload | 内嵌 image/svg+xml（1660 字符） | uploads/3f1b95e4844644b853e205f446ffaa1d.svg |
| 域名 | marketing | url | https://marketing.sav.com/assets/sav-logo.svg | https://marketing.sav.com/assets/sav-logo.svg |
| 域名 | IONOS | upload | 内嵌 image/svg+xml（2287 字符） | uploads/e175fb423b4ce24fde3be7911e590869.svg |
| 域名 | spaceship | upload | 内嵌 image/svg+xml（823 字符） | uploads/f25baca66d23c55da35165782f2f0000.svg |

> 共 50 项：`upload` 31 项、`url` 19 项。外部地址一列与源文件完全一致（逐字符相同）。

---

## 6. 测试库导入结果

### 6.1 导入流程

在**独立测试数据库**（全新空库，非任何现有数据）中执行：

1. 初始化管理员；
2. `POST /api/v1/admin/import/preview` 上传原始 `data.json`；
3. `POST /api/v1/admin/import?conflict_strategy=skip` 执行导入；
4. 通过管理接口与公开接口核对结果；
5. 浏览器渲染验证。

### 6.2 预览报告（导入前）

| 指标 | 值 |
| --- | --- |
| 识别格式 | `legacy-v1` |
| 分类：新增 / 冲突 | 10 / 0 |
| 站点：新增 / 冲突 | 110 / 0 |
| 网址非法（导入时跳过） | 0 |
| 图标：原样保留（url） | 19 |
| 图标：转为本地（upload） | 31 |
| 图标：无效丢弃 | 0 |

映射告警（原文）：

```
31 个内嵌 data URI 图标将转换为本地上传文件（图标内容原样保留）
19 个外部图标地址已原样保留，未替换为网站默认 Favicon
3 个文本型图标（如「延迟」）无法作为图片，已设为无图标并把文本保留到关键词
8 项旧站字段 si（图标标识）已合并到关键词
4 项旧站字段 tag 已合并到关键词
6 项旧站字段 color 未映射（当前数据模型没有对应字段）
10 个分类图标（emoji）未映射（当前数据模型没有对应字段）
```

### 6.3 导入结果

| 指标 | 值 |
| --- | --- |
| 分类导入 | 10 |
| 站点导入 | 110 |
| 跳过 | 0 |
| 生成的本地图标文件 | 31 个 `.svg` |

**顺序验证**

| 分类 | sort_order | 站点数 |
| --- | --- | --- |
| Daily | 10 | 17 |
| SpeedTest | 20 | 8 |
| 常用 | 30 | 7 |
| IP&DNS | 40 | 14 |
| AI & 智能 | 50 | 10 |
| VPS/云服务器 | 60 | 26 |
| 邮箱 | 70 | 13 |
| 开发 & 编程 | 80 | 3 |
| 域名 | 90 | 11 |
| 其他导航 | 100 | 1 |

分类内站点 `sort_order` 为 10,20,30…（例：Daily 首条 `NodeSeek`=10，末条 `极简探针`=170），与源文件顺序一致。分类名首尾空格已去除（`邮箱 ` → `邮箱`、`域名 ` → `域名`）。

**图标类型分布（导入后）**：`upload` 31、`url` 19、`none` 60（= 57 无图标 + 3 文本图标）。

**上传图标服务验证**

```text
GET /api/v1/public/uploads/bbf6d717f57c583c494d0e9980affc8d.svg
-> 200
   Content-Type: image/svg+xml
   Cache-Control: public, max-age=86400, immutable
   Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; sandbox
   bytes: 1473
```

### 6.4 浏览器渲染验证（Chromium + 真实后端）

| 检查项 | 结果 |
| --- | --- |
| 分类数量与顺序 | ✅ 10 个，顺序与源文件一致 |
| 各分类站点数 | ✅ 17 / 8 / 7 / 14 / 10 / 26 / 13 / 3 / 11 / 1 = 110 |
| 图标元素 | ✅ 31 个本地 SVG + 18 个外部图标成功渲染（`naturalWidth > 0`） |
| 回退显示 | ✅ 61 个（60 个无图标 + 1 个图标不可用）显示为首字母占位 |
| 搜索 | ✅ 搜索"测速"返回 5 条（谷歌测速、Fast、cf测速、Measurementlab、IP+测速），经由 `/api/v1/public/search` |

---

## 7. 数据质量检查

| 检查项 | 结果 |
| --- | --- |
| 站点 URL 合法性 | ✅ 110/110 为 http/https |
| 缺失 `name` / `url` | ✅ 0 |
| 分类名首尾空格 | ⚠️ 2 个（`邮箱 `、`域名 `）→ 已自动去除 |
| 重复站点 URL | ⚠️ 1 组：`https://dash.cloudflare.com/` 同时出现在 `Daily` 与 `域名` |
| 重复 (名称+URL) | ⚠️ 同上（`cloudflare`） |
| 无效分类关联 | ✅ 无（旧站为嵌套结构，不存在孤立引用） |
| 图标地址可达性 | ⚠️ 19 个中 18 个可访问；1 个超时、1 个返回 HTML（见 8.4） |
| 图标字段格式 | ⚠️ 3 个为文本、1 个为 `iconType=favicon` |

**关于重复 URL**：`cloudflare` 在两个分类中各出现一次。这属于旧站的有意安排（`Daily` 里是快捷入口，`域名` 里是域名服务），**导入时按分类分别保留**（因冲突检测按"分类内 URL"判定），未做去重。如果希望只保留一条，请在正式导入前告知。

---

## 8. 异常数据与需人工处理的问题

### 8.1 文本型图标（3 项，无法显示为图片）

| 分类 | 名称 | 原值 |
| --- | --- | --- |
| Daily | 各地延迟域名 | `延迟` |
| Daily | 地址生成器 | `地址` |
| IP&DNS | IP 检测 | `ip` |

处理：`icon_type=none`，文本已并入 `keywords`（可搜索）。**未替换成 Favicon。**
建议：如需图标，请提供图片地址或在后台"网站管理"中上传图标。

### 8.2 `si`（8 项）与 `tag`（4 项）

旧站用 `si` 指定 Simple Icons 图标（如 `github`、`vercel`、`stackoverflow`），当前模型没有该字段，已并入 `keywords`。

| 分类 | 名称 | si | tag |
| --- | --- | --- | --- |
| Daily | 大文件下载 | speed | |
| Daily | 节点一键脚本 | yongge | |
| AI & 智能 | Midjourney | artstation | Art |
| 邮箱 | 88完美邮箱 | 88完美邮箱 | |
| 邮箱 | 微软邮箱管理 | 微软 | |
| 开发 & 编程 | GitHub | github | |
| 开发 & 编程 | Vercel | vercel | Deploy |
| 开发 & 编程 | Stack Overflow | stackoverflow | Q&A |
| IP&DNS | IPinfo | | Info |

建议（**未自动执行**）：若希望恢复这些图标，可将 `si` 映射到 Simple Icons CDN，例如
`https://cdn.simpleicons.org/<slug>`（如 `https://cdn.simpleicons.org/github`）。
这属于新增图标来源，需要你确认后再实施；`开发 & 编程` 三个站点目前没有图标。

### 8.3 `iconType: favicon`（1 项）

| 分类 | 名称 |
| --- | --- |
| IP&DNS | IPinfo |

处理：`icon_type=none`。按要求**未自动抓取或替换**。后台可用"抓取站点图标"按钮手动获取。

### 8.4 图标地址问题（2 项）

| 分类 | 名称 | 图标地址 | 现象 | 当前表现 |
| --- | --- | --- | --- | --- |
| Daily | komari探针 | `https://mjj.today/i/lUGk4m` | 返回 `text/html`（是图床网页，不是图片直链） | 渲染失败 → 显示首字母回退 |
| VPS/云服务器 | 莱卡云 | `https://www.lcayun.com/upload/banner/2023-10/11/169700642979539.png` | 自动检查时 8 秒超时；浏览器中可正常加载 | 浏览器可显示，但服务器响应较慢 |

处理：**原地址保持不动**（未覆盖）。前端 `IconImage` 在图片加载失败时回退为标题首字母，不会破坏布局，也不会改写数据。
建议：把 `komari探针` 的图标换成图片直链；关注 `莱卡云` 图标的加载速度。

### 8.5 未映射字段（当前模型无对应字段）

| 字段 | 数量 | 内容示例 | 建议 |
| --- | --- | --- | --- |
| 分类 `icon`（emoji） | 10 | 📁 ✨ 🌐 🤖 🐔 📧 💻 🔗 | 若需要，可为分类增加图标字段（需改契约与数据库） |
| `color` | 6 | `#000000`、`#2D72D9`、`#ffffff`、`#F58025` | 同上；涉及 IP138、IPinfo、Midjourney、GitHub、Vercel、Stack Overflow |

这些字段已记录在案，**未写入任何替代字段**，等待你的决定。

### 8.6 其他

- `logoType`（33 项，值均为 `svg`）：仅用于识别 SVG，无需存储，已忽略。
- 57 个站点本来就没有图标，保持无图标（前台显示首字母占位）。

---

## 9. 正式导入方案（避免覆盖现有数据）

测试库导入已验证通过。**尚未对任何正式/现有数据执行导入**（当前项目没有生产实例）。

### 9.1 前置条件

1. 已部署的实例可访问，且已创建管理员。
2. **导入前先备份**：管理后台 →「数据管理」→「立即备份」，或服务器执行 `./scripts/update.sh` 前的一致性快照机制。
3. 目标库建议为空或只含少量数据（导入不会删除任何已有数据）。

### 9.2 推荐方式：管理后台（可视化，有预览）

1. 登录后台 →「数据管理」→「导入数据」；
2. 选择 `data.json`；
3. 查看预览：应显示 **分类 10 / 站点 110 / 冲突 0**、图标"原样保留 19 · 转为本地 31"；
4. 冲突策略保持 **"跳过已存在的条目（默认）"**；
5. 点击「确认导入」；
6. 到「分类管理」「网站管理」核对顺序与图标，并在首页确认显示效果。

> 默认策略 `skip` **不会覆盖已有数据**：同名分类会复用，同分类内同 URL 的站点会跳过。

### 9.3 备选方式：命令行

```bash
# 1) 预览（不写库）
curl -sS -b cookies.txt -H 'Content-Type: application/json' \
  --data-binary @data.json \
  http://127.0.0.1:8080/api/v1/admin/import/preview | jq .

# 2) 导入（默认不覆盖）
curl -sS -b cookies.txt -H 'Content-Type: application/json' \
  --data-binary @data.json \
  'http://127.0.0.1:8080/api/v1/admin/import?conflict_strategy=skip' | jq .
```

如需覆盖已存在的同名条目，把策略改为 `overwrite`；如需两者都保留，用 `rename`。

### 9.4 回滚

- 导入在**单个数据库事务**内执行：失败时整体回滚，不会留下半成品。
- 若导入后发现内容不满意：管理后台「数据管理」→「备份与恢复」→ 选择导入前的快照 →「恢复」。

---

## 10. 本次未验证事项

| 未验证项 | 原因 |
| --- | --- |
| 正式/生产实例导入 | 当前无部署实例；仅完成独立测试库验证 |
| 移动端与跨浏览器图标显示 | 环境仅有 Chromium，无设备模拟 |
| 31 个内嵌 SVG 的视觉逐一确认 | 已确认全部可加载（`naturalWidth > 0`），未逐一目视比对内容 |
| `si` → Simple Icons CDN 方案 | 属于新增图标来源，等待用户确认，未实施 |
| 图标的长期可用性 | 外部图标依赖第三方站点，本次仅做一次可达性抽查 |

---

## 11. 附录：本次使用的产物

| 产物 | 位置 | 说明 |
| --- | --- | --- |
| 脱敏样本 | `docs/samples/legacy-data.sample.json` | 供自动化测试使用，可提交仓库 |
| 自动化测试 | `backend/internal/service/transfer_test.go` | 覆盖数组结构解析、顺序保留、图标映射、data URI 转换、幂等导入 |
| 导入实现 | `backend/internal/service/transfer.go`、`icons.go` | legacy 解析与图标转换 |
| 契约更新 | `docs/openapi/openapi.yaml` | `/admin/import` 查询参数、`ImportPreview` 新字段、`LegacyCategoryArray` |
| 原始数据 | 本地 `data.json`（**未入库**） | 受 `.gitignore` 保护 |

---

## 12. 需要你确认的事项

1. **文本图标 3 项**（`各地延迟域名`、`地址生成器`、`IP 检测`）：是否需要补充图片图标？
2. **`si` 图标 8 项**：是否允许映射到 Simple Icons CDN（`https://cdn.simpleicons.org/<slug>`）？其中 `开发 & 编程` 的三个站点目前完全没有图标。
3. **`iconType=favicon` 的 `IPinfo`**：是否允许抓取其站点图标？
4. **`komari探针` 图标**：是否替换为图片直链（当前地址是网页而非图片）？
5. **重复的 `cloudflare`**：`Daily` 与 `域名` 各一条，是否保持两条？
6. **`color`（6 项）与分类 emoji（10 项）**：是否需要新增对应字段以完整保留？

---

## 13. P3 收尾补充：迁移质量增强（2026-10-24）

> 本节是 P3 阶段收尾的补充记录，不替代 P3 阶段报告。
> 目标：恢复旧版图标显示能力、扩展数据模型、加固迁移安全，并用真实数据重新验证。

### 13.1 本次新增工作

#### A. 图标显示链（前端）

`IconImage` 组件实现分级回退，**所有自动步骤都只是显示回退，绝不改写已存储的图标地址**：

1. 用户设置的图标：`url` / `upload` / `auto` / `simple`（CDN）/ `text`（文字）
2. 站点自身的 Favicon：`<origin>/favicon.ico`
3. 标题首字母占位

| 类型 | 含义 | 是否用户显式设置 |
| --- | --- | --- |
| `url` | 用户提供的外部图片地址 | ✅ |
| `upload` | 本地上传（含迁移的 31 个内嵌 SVG） | ✅ |
| `simple` | 已验证的 Simple Icons 标识 | ✅ |
| `text` | 文字图标（如「延迟」） | ✅ |
| `favicon` | 显示时取站点 Favicon | ✅ |
| `none` | 无图标 → 显示时依次尝试 Favicon、首字母 | — |

#### B. Simple Icons 语义独立化

- 内置**官方 slug 白名单**（3,494 个，来源 Simple Icons `slugs.md`，2026-10-24 抓取），
  存于 `backend/internal/service/data/simpleicons.txt`，通过 `go:embed` 打包。
- 只有**命中白名单**的 `si` 值才映射为 `icon_type=simple`；其余（如 `微软`、`yongge`、`speed`）
  **不会**被拼成 CDN URL，而是保留到关键词并计入告警。
- 实测：旧站 8 个非空 `si` 中 4 个有效（`github`、`vercel`、`stackoverflow`、`artstation`），
  4 个无效（`speed`、`yongge`、`88完美邮箱`、`微软`）。CDN 探测结果与白名单一致。

#### C. 数据模型扩展（版本化迁移 `0002_icons_and_labels`）

| 表 | 新增列 | 用途 |
| --- | --- | --- |
| `categories` | `icon` | 分类 emoji / 极短文字 |
| `categories` | `color` | 可选强调色 |
| `bookmarks` | `color` | 可选强调色 |
| `bookmarks` | `tags` | 标签（旧站 `tag`） |

- 使用 `ALTER TABLE ... ADD COLUMN ... DEFAULT ''`，**非破坏性**，不重建数据库。
- 同步更新：后端模型、`store` 读写、OpenAPI 契约（含 `icon_type` 枚举扩展）、前端类型与表单、测试。
- 迁移器支持多版本；重复启动不会重复执行。

#### D. 安全加固

| 项 | 措施 | 验证 |
| --- | --- | --- |
| SVG 不被当作 HTML 执行 | 以 `image/svg+xml` + `X-Content-Type-Options: nosniff` + `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; sandbox` 提供；前端仅用 `<img>` 引用，不做内联注入 | `TestSVGUploadIsServedAsImageWithSandbox` |
| 伪装成 SVG 的 HTML | 内容嗅探改为要求 **同时** 满足 `.svg` 扩展名与 `<svg` 根元素 | `TestUploadRejectsHTMLMasqueradingAsSVG` |
| 上传限制 | ≤1 MiB、类型白名单、文件名白名单、无路径穿越 | `TestUploadRejectsUnsupportedAndOversizedFiles`、`TestPublicUploadRejectsPathTraversal` |
| 公开读取需授权边界 | 上传目录只通过受控路由提供；未知文件 404 | `TestPublicUploadRejectsPathTraversal` |
| 导入失败可回滚 | 单事务写入；书签引用未知分类时跳过而非猜测 | `TestImportRollsBackAndRemovesUploadsOnFailure` |
| 导入失败不留孤儿文件 | 转换出的图标文件与上传元数据在事务失败时一并清理 | `TestImportRollsBackAndRemovesUploadsOnFailure` |
| 备份同时覆盖数据库与图标 | 快照 = 数据库副本 + `<id>.files/` 图标目录；恢复时一起还原 | `TestBackupAndRestoreIncludesUploads` |

### 13.2 真实数据重新导入结果（独立测试库）

在**全新空库**中重新导入真实 `data.json`（未触碰任何生产环境）：

| 核对项 | 结果 |
| --- | --- |
| 分类数量 | ✅ 10 / 10 |
| 网站数量 | ✅ 110 / 110 |
| 分类顺序 | ✅ 与源文件逐项一致（`Daily → SpeedTest → 常用 → IP&DNS → AI & 智能 → VPS/云服务器 → 邮箱 → 开发 & 编程 → 域名 → 其他导航`） |
| 分类内网站顺序 | ✅ `sort_order` 10,20,30… 与源顺序一致 |
| 手动图标地址（19 个） | ✅ **19/19 逐字符未变**（0 个被改动） |
| 内嵌 SVG（31 个） | ✅ **31/31 字节级无损**（解码后与存储文件逐字节相同） |
| 无效网址 | 0（与首次一致） |
| 导入结果 | 分类 10 / 网站 110 / 跳过 0 |

**图标类型分布（改进后）**

| 类型 | 数量 | 说明 |
| --- | --- | --- |
| `upload` | 31 | 内嵌 SVG 转本地（内容无损） |
| `url` | 19 | 外部地址原样保留 |
| `simple` | 4 | `si` 命中白名单 |
| `text` | 3 | 文字型图标 |
| `favicon` | 1 | 旧站标记 favicon |
| `none` | 52 | 无图标 → 前台尝试站点 Favicon |

（改进前：`none` 60 / `upload` 31 / `url` 19。）

### 13.3 图标显示效果（浏览器实测，Chromium + 真实后端）

| 指标 | 改进前 | 改进后 |
| --- | --- | --- |
| 显示图片的网站 | 49 | **85** |
| 显示文字图标 | 0 | **3** |
| 首字母占位 | 61 | **22** |
| 有图标的网站合计 | 49 | **88** |

改进后的图片构成：本地 SVG 31 · 站点 Favicon 32 · 外部地址 18 · Simple Icons 4。

**回退链实际生效示例**：`komari探针` 的原图标地址是图床网页（返回 `text/html`，无法作为图片），
系统按设计回退到其站点 Favicon 并成功显示，**原地址未被改写**。

**仍有 22 个网站显示首字母**（其站点自身未提供可访问的 Favicon），清单见 13.5。

### 13.4 视觉审查材料

⚠️ **未能提供真实浏览器截图**：本环境的浏览器工具要求桌面窗口可见，无法截屏（多次尝试均返回
`Screenshot needs a visible tab`）。作为替代，提供了两份**离线 DOM 快照**（真实页面渲染结果的静态副本，
非截图）。

> ⚠️ 这两份快照**包含真实站点名称与地址**，因此**不纳入 Git 版本库**（已在 `.gitignore` 中排除
> `docs/screenshots/`）。它们保留在本机，供你直接双击打开审查。

| 文件（本地，未入库） | 内容 | 说明 |
| --- | --- | --- |
| `docs/screenshots/home-page-snapshot.html` | 导航首页（10 分类 / 110 磁贴） | 已内联样式与 31 个本地 SVG（base64），可直接双击打开 |
| `docs/screenshots/admin-bookmarks-snapshot.html` | 后台「网站管理」列表（110 行，含图标类型与标签） | 同上 |

> 快照为真实页面 DOM 的静态副本，已移除脚本；远程图标（Favicon / 外部地址）需要联网才能显示。

### 13.5 仍需人工处理的网站

**A. 显示首字母占位的 22 个网站**（站点自身无可用 Favicon）

| 分类 | 网站 |
| --- | --- |
| Daily | NodeSeek、烧饼论坛、节点一键脚本 |
| SpeedTest | Fast、LibreSpeed、Measurementlab、IP+测速 |
| IP&DNS | Scamalytics、DNSCheck |
| AI & 智能 | 通义千问 |
| VPS/云服务器 | vmiss、Vmrack、bitsflow、nosla、ReckNerd、Zouter、虫网 |
| 邮箱 | 88完美邮箱、临时邮箱、nodeseek邮箱、微软邮箱管理 |
| 其他导航 | 科技lion导航站 |

> 处理方式：在后台「网站管理」中为这些站点设置图标（上传 / 图片地址 / Simple Icons 标识 / 文字图标）。
> 其中「开发 & 编程」的 GitHub/Vercel/Stack Overflow 已通过 Simple Icons 显示，无需处理。

**B. 之前提出的 6 项确认，本次已处理 4 项**

| # | 事项 | 状态 |
| --- | --- | --- |
| 1 | 文本图标 3 项 | ✅ 已保留为 `icon_type=text` |
| 2 | `si` 8 项 | ✅ 4 项有效→Simple Icons；4 项无效→保留在关键词（不再猜测 CDN） |
| 3 | `iconType=favicon` 的 IPinfo | ✅ 已设为 `icon_type=favicon`（显示时取站点 Favicon，不抓取、不覆盖） |
| 4 | `komari探针` 图标是网页而非图片 | 🟡 原地址保留；已自动回退到站点 Favicon 正常显示。如需精确还原，请提供图片直链 |
| 5 | 重复的 `cloudflare`（Daily + 域名） | 🟡 仍保留两条（未擅自去重），如需合并请告知 |
| 6 | `color` 6 项与分类 emoji 10 项 | ✅ 已新增字段并完整迁移 |

**C. 其他**

- `莱卡云` 图标服务器响应较慢（自动检查曾 8 秒超时），浏览器中可正常显示，建议关注。

### 13.6 测试与构建结果

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 后端构建 | `go build ./...` | ✅ 通过 |
| 后端静态检查 | `go vet ./...` | ✅ 通过 |
| 后端测试 | `go test -count=1 ./...` | ✅ **70 通过 / 0 失败**（上次 58） |
| 前端类型检查 + 构建 | `npm run build` | ✅ 通过 |
| 契约一致性 | `TestRoutesMatchOpenAPIContract` | ✅ 通过 |
| 迁移新增列 | `TestExtendedColumnsExist` | ✅ 通过 |
| 真实数据核对 | `verify.js`（计数/顺序/图标/字节比对） | ✅ 全部通过 |

### 13.7 Git 状态

| 项 | 内容 |
| --- | --- |
| 分支 | `master` |
| 上次提交 | `e6d3d44` — `feat(import): support the real legacy data format and verify migration` |
| 本次提交 | 见聊天总结（本文件随该提交一起入库） |
| 工作区 | 干净 |
| 远程 / 推送 | 未配置 / 未推送 |

### 13.8 本次未验证事项

| 未验证项 | 原因 |
| --- | --- |
| 真实浏览器截图 | 浏览器工具要求桌面窗口可见，本环境不可用（已用离线 DOM 快照替代） |
| 移动端与跨浏览器图标显示 | 环境仅有 Chromium，无设备模拟 |
| 正式/生产环境导入 | 按要求未操作生产环境 |
| 22 个占位网站的人工图标补录 | 需人工决策 |
| Docker 镜像构建与容器内运行 | Docker 守护进程不可用 |

