# Personal Agent Skills 索引

这个目录把 `personal-agent-skills` 中的 12 个 skill 整理成一个本地 HTML 工作区。首页用于快速选路，每个详情页介绍一个 skill 的使用场景、核心流程、产出验证和边界。

## 总览

| Skill | 类型 | 适用场景 | 详情 |
| --- | --- | --- | --- |
| `align-research-project` | 研究对齐 | 科研方向、问题、技术路线或实验计划还不清楚时，先对齐理解再执行。 | [查看](skills/align-research-project.html) |
| `arxiv-daily-paper-radar` | 论文雷达 | 从个性化 Daily arXiv 邮件中筛选、排序并总结高价值论文。 | [查看](skills/arxiv-daily-paper-radar.html) |
| `hf-daily-paper-radar` | 论文雷达 | 拉取 Hugging Face Daily Papers，按用户研究兴趣生成中文阅读报告。 | [查看](skills/hf-daily-paper-radar.html) |
| `research-ui-frontend` | 前端视觉规范 | 为研究工具、运营后台和本地开发 UI 套用 Skillctl 风格。 | [查看](skills/research-ui-frontend.html) |
| `organize-html-reports` | HTML 工作区 | 把 Markdown、报告、实验记录和阅读材料组织成 `index.html` 入口的本地网页。 | [查看](skills/organize-html-reports.html) |
| `local-web-ui-ship-loop` | UI 交付循环 | 小范围本地 Web UI 修改、构建、浏览器验证、提交和发布。 | [查看](skills/local-web-ui-ship-loop.html) |
| `init-agents-md` | 项目指令初始化 | 为项目创建或更新 `AGENTS.md`，沉淀项目规则和可复用 workflow。 | [查看](skills/init-agents-md.html) |
| `office-report-materials` | 正式材料 | 扩写、打磨和检查 Word/PPT 学术、项目、专利或中期报告材料。 | [查看](skills/office-report-materials.html) |
| `weekly` | 周报 | 根据简要笔记生成正式周报 DOCX，并按需导出 PDF 或准备邮件。 | [查看](skills/weekly.html) |
| `remote-server-workspace` | 远程工作区 | 通过本地 Codex 会话操作远程 SSH 服务器上的代码、数据和 git 仓库。 | [查看](skills/remote-server-workspace.html) |
| `self` | 自我复盘 | 复盘本地 Codex 历史，提取长期偏好、执行教训和可复用规则并归档。 | [查看](skills/self.html) |
| `hatch-pet` | 视觉资产 | 从概念、参考图或品牌线索创建 Codex 兼容的动画 pet 精灵图。 | [查看](skills/hatch-pet.html) |

## 选用路径

### 研究与论文

- 先用 `align-research-project` 对齐科研问题、方法边界和实验目标。
- 对每日论文源做筛选时，Apple Mail 个性化 arXiv 摘要走 `arxiv-daily-paper-radar`，Hugging Face Daily Papers 走 `hf-daily-paper-radar`。

### 本地 UI 与 HTML 报告

- 视觉风格、组件密度、色彩和排版规则由 `research-ui-frontend` 决定。
- 需要把文档、实验记录、论文笔记变成本地网页入口时，用 `organize-html-reports`。
- 对 React/Vite 或 Go embedded frontend 做小改动并要验证时，用 `local-web-ui-ship-loop`。

### 项目规则与正式材料

- 新项目或需要补全 agent 行为规范时，用 `init-agents-md`。
- 正式 Word/PPT 报告、专利材料、中期材料走 `office-report-materials`。
- 每周工作汇报、模板填充、PDF 导出和邮件附件流程走 `weekly`。

### 环境、复盘与资产

- 远程 SSH 项目、conda 环境、远程 git 和远程文件同步走 `remote-server-workspace`。
- 从历史对话和执行日志提炼长期规则走 `self`。
- 需要生成 Codex app 可用的动画 pet 包时，走 `hatch-pet`。

## 源目录

这些页面介绍的是 `personal-agent-skills/skills/<skill-name>/SKILL.md` 中的工作流。当前 HTML 工作区只保留相对链接，适合直接用 `file://` 打开。
