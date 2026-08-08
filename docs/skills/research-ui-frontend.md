# research-ui-frontend

`research-ui-frontend` 是 Skillctl 风格的前端视觉规范 skill。它适合研究工具、实验仪表盘、本地开发工具、评估浏览器和运维控制台，强调安静、紧凑、技术感明确的产品界面。

| 项 | 内容 |
| --- | --- |
| 类型 | 前端视觉规范 |
| 原始 skill | `personal-agent-skills/skills/research-ui-frontend/SKILL.md` |
| 主要产出 | 配色、排版、组件密度、交互状态和响应式检查规则 |

## 何时使用

- 新建或迁移研究/运维型前端页面。
- 需要统一颜色、字体、间距、卡片、列表、按钮、输入框、徽标和图标规则。
- 需要检查现有页面是否符合 Skillctl 的研究工具风格。

## 视觉原则

- 主体使用温和中性色浅色主题，暗色模式保持安静的炭黑表面。
- 字体上，UI 用 Inter，路径、哈希和技术元数据用 JetBrains Mono，少量展示标题可用 Instrument Serif。
- 半径默认保持 3 到 6px。
- 除中性色外，一屏最多使用三个额外色系。
- 用列表、表格和紧凑对象卡承载信息，避免营销式 hero、装饰渐变和大面积空白。

## 工作方式

1. 确定用户要查看或操作的核心对象。
2. 围绕这些对象选择列表、表格、卡片、详情面板、过滤器和状态区。
3. 优先套用 style kit 的 CSS token，而不是随手写一次性颜色。
4. 让标题、描述、路径、状态、数量和动作保持稳定层级。
5. 检查桌面和移动宽度，确认长文本、按钮和技术路径不溢出。

## 边界与注意

- 这个 skill 只负责设计语言，不负责构建、提交和发布。
- 如果任务还涉及实际 UI 修改和验证，应配合 `local-web-ui-ship-loop`。
- 不把研究工具做成营销落地页。

## 导航

上一项：[hf-daily-paper-radar](hf-daily-paper-radar.html) · [返回首页](../index.html) · 下一项：[organize-html-reports](organize-html-reports.html)
