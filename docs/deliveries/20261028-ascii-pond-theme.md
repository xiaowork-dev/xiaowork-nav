# ASCII Pond 可选主题交付记录（待验收）

日期：2026-10-28  
分支：`feat/ascii-pond-theme`  
状态：**代码已提交独立分支，尚未执行本地构建、浏览器验收、合并或上线**

## 本次变更
- 增加第三套“🐟 锦鲤池塘”视觉主题，原有深色、浅色均保留。
- 导航首页背景从原有渐变切换为接近 ASCII Pond 上游的墨黑色与暖白/橙色调；不修改布局结构、分类、站点数据和接口。
- 前台顶栏从二档主题按钮改为三选一主题选择器；本地浏览器主题偏好仍存储在 `xn-theme`。
- 在导航首页引入按需加载的 ASCII Pond 动画，切换到其他主题时销毁实例。动画不会拦截鼠标点击；优先避让导航、搜索框和按钮区域。
- 相关静态模块已复制到 `frontend/public/vendor/ascii-pond/`，保留上游 MIT 许可证。

## 技术范围
- 仅修改前端：`frontend/src/composables/useTheme.ts`、`frontend/src/components/HeaderActions.vue`、`frontend/src/components/AsciiPondBackground.vue`、`frontend/src/pages/HomePage.vue`、`frontend/src/styles/pond-theme.css`、`frontend/src/main.ts`。
- 未修改 API 契约、后端、数据库、Docker 编排、升级脚本和管理后台的默认主题枚举（后台默认主题依然是自动/浅色/深色，锦鲤主题是浏览器本地可选外观）。
- 参考与第三方代码：<https://github.com/stqfdyr/ascii-pond>，MIT License，Copyright (c) 2026 stqfdyr。

## 验证情况
- 已检查代码文件、入口、主题逻辑和上游挂载/卸载接口。
- **未执行** `cd frontend && npm run build`。
- **未执行** `cd frontend && npm run test`。
- **未执行** 真实浏览器视觉验收（桌面/手机）。
- **未执行** 真实 Docker 构建与更新测试。
- **未合并至 main；未部署到任何服务器。**

## 待验收与可能风险
1. 运行前端类型检查、构建和现有测试。
2. 在桌面和手机端切换三种主题，确认动画切换、页面布局、侧栏和站点点击正常。
3. 关注大量网站时的绘制性能，必要时减少锦鲤数量或关闭移动端动画。
4. 检查滚动到长页面时的前景文字对比度和锦鲤避让效果。
5. 确认第三方动画脚本在浏览器及生产静态目录里可加载。

## 下一步
完成测试、人工视觉验收后再合并 PR。发布请独立进行版本发布和升级验收，不要直接触碰生产环境。
