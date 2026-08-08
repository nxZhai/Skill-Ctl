# weekly

`weekly` 用于把简要周报笔记整理成正式周报文档，并在用户要求时继续导出 PDF 或准备 Apple Mail 邮件。它优先保护模板格式和历史报告，不直接改模板。

| 项 | 内容 |
| --- | --- |
| 类型 | 周报 |
| 原始 skill | `personal-agent-skills/skills/weekly/SKILL.md` |
| 主要产出 | 新 DOCX 周报、可选 PDF、可选邮件草稿或发送状态 |

## 何时使用

- 用户提供本周工作或研究笔记，需要填入周报模板。
- 项目有固定 DOCX 模板、命名规则、输出目录或邮件规则。
- 用户要求导出 PDF，或准备带 PDF 附件的邮件。

## 工作方式

1. 解析模板路径、输出目录、日期格式、章节、语气和邮件规则。
2. 若存在 `WEEKLY_REPORT_TEMPLATE`，把它作为优先模板。
3. 从模板创建新 DOCX，不修改模板本身。
4. 用简体中文、正式且具体的主管汇报语气扩写笔记。
5. 不编造指标、实验、日期或结论；笔记太稀疏时保守扩展。
6. 创建 DOCX 后检查结构，并尽可能渲染或打开做视觉 QA。

## PDF 与邮件

- 只有用户请求或项目规则允许时才导出 PDF。
- PDF 默认与 DOCX 同名并放在相邻位置。
- 邮件流程只在用户要求后进入；默认附件应是确认后的 PDF，而不是 DOCX。
- 若项目要求发送前确认，应尊重该规则。

## 边界与注意

- 不覆盖历史报告。
- 不硬编码机器特定 home 目录。
- 渲染或视觉 QA 不可用时，应明确说明剩余风险。

## 导航

上一项：[office-report-materials](office-report-materials.html) · [返回首页](../index.html) · 下一项：[remote-server-workspace](remote-server-workspace.html)
