# organize-html-reports

`organize-html-reports` 用于把研究、阅读或实验目录整理成本地 HTML 工作区。它要求 `index.html` 作为入口，所有用户需要阅读的页面都从首页可达，并且链接保持项目相对路径，适合直接用 `file://` 打开。

| 项 | 内容 |
| --- | --- |
| 类型 | HTML 工作区 |
| 原始 skill | `personal-agent-skills/skills/organize-html-reports/SKILL.md` |
| 主要产出 | `index.html`、一组 HTML 报告页、共享 CSS/JS assets、保留的 Markdown 源文件 |

## 何时使用

- 需要把 Markdown 笔记、报告、实验结果、论文阅读记录或分析页面组织成可浏览工作区。
- 需要让数学公式、表格和 Markdown 源保持可复制或可追溯。
- 需要把页面后续同步到飞书或 Lark，但同步只在用户明确要求时做。

## 工作方式

1. 盘点项目中真正面向用户的文档、报告和结果，排除缓存、调试日志、临时文件和重复草稿。
2. 设计信息架构，以 `index.html` 为入口，把页面按 docs、reports、experiments、analyses 等类别分组。
3. 优先用自带 `render_markdown_report.py` 把 Markdown 渲染为 HTML。
4. 应用 `research-ui-frontend` 的研究工具风格。
5. 只在用户要求时执行飞书或 Lark 同步。

## 验证重点

- 从首页打开，而不是只检查文件存在。
- 首页链接到 HTML 页面，不默认链接原始 `.md`。
- 桌面和移动宽度都要检查。
- 表格复制、公式复制和本地 `file://` 链接按页面内容适用性验证。

## 边界与注意

- 保留 Markdown 源文件作为可编辑事实源。
- 不硬编码用户名、home 目录或机器本地绝对路径。
- 不把工作区首页做成营销页。

## 导航

上一项：[research-ui-frontend](research-ui-frontend.html) · [返回首页](../index.html) · 下一项：[local-web-ui-ship-loop](local-web-ui-ship-loop.html)
