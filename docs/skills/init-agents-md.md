# init-agents-md

`init-agents-md` 用于在 Codex App 中创建或更新项目 `AGENTS.md`，相当于一个更谨慎、更项目化的 `/init` 工作流。它会先检查项目，再写出可执行、可继承、可维护的 agent 指令。

| 项 | 内容 |
| --- | --- |
| 类型 | 项目指令初始化 |
| 原始 skill | `personal-agent-skills/skills/init-agents-md/SKILL.md` |
| 主要产出 | 项目 `AGENTS.md`、可选项目本地 skill 建议、portable path 检查结果 |

## 何时使用

- 用户要求初始化 `AGENTS.md`、写项目贡献者/agent 指令或补充项目规则。
- 项目类型需要明确，例如全栈、研究代码、文档报告、skill 仓库、飞书工作流或混合项目。
- 需要把历史偏好转成项目级的可复用规则。

## 工作方式

1. 先运行项目检查脚本和 portable path 检查。
2. 阅读现有 `AGENTS.md`、README、包管理文件、测试配置和目录结构。
3. 只问会阻塞写作的问题。
4. 基于模板、instruction discovery 规则和项目类型写草稿。
5. 根据是否有 GitHub remote 写入 commit/push 完成规则。
6. 推荐真正重复出现的项目 skill 或 subagent。
7. 写入后复查 Markdown 结构和路径可移植性。

## 边界与注意

- 不盲目覆盖已有非空 `AGENTS.md`。
- 不擅自 `git init`、clone、push 或启用外部 skill。
- 不写机器特定用户名或绝对 home 路径进可复用指令。
- 不把原始聊天记录塞进项目指令。

## 导航

上一项：[local-web-ui-ship-loop](local-web-ui-ship-loop.html) · [返回首页](../index.html) · 下一项：[office-report-materials](office-report-materials.html)
