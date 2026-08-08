# remote-server-workspace

`remote-server-workspace` 用于从本地 Codex 会话操作远程 SSH 服务器上的文件、数据、代码和 git 仓库。它把远程执行留在 SSH 中，把本地 skill、飞书/Lark 集成和对话上下文保留在本地。

| 项 | 内容 |
| --- | --- |
| 类型 | 远程工作区 |
| 原始 skill | `personal-agent-skills/skills/remote-server-workspace/SKILL.md` |
| 主要产出 | 远程命令结果、远程文件改动、远程测试结果、可选本地到远程/远程到飞书同步 |

## 何时使用

- 用户要求进入某个 SSH 主机、远程项目目录或 conda 环境。
- 需要检查、编辑或运行远程代码和数据命令。
- 需要操作远程 git 仓库，或在远程文件和飞书/Lark 之间同步。

## 必要输入

- SSH host alias。
- 远程项目目录。
- conda 环境名，除非任务明确不需要环境。
- 具体操作类型：检查、编辑、运行、git、同步等。
- 是否需要通过本地代理访问外网。

## 工作方式

1. 优先用自带 `remote_session.py`  helper 创建可重复 SSH 会话。
2. 先做只读发现：`pwd`、`ls`、`git status`、目标 `rg` 或 `find`。
3. 小生成文件可在远程直接创建；复杂编辑应复制到本地临时路径后再上传。
4. 修改后用远程 `git diff`、校验和、测试或目标命令验证。
5. 需要飞书/Lark 同步时，远程文件先复制到本地，再用本地 Lark skill 处理。

## 边界与注意

- 缺少必要输入时只问缺失项。
- 删除文件、覆盖大数据、push、使用代理转发等风险操作必须明确确认。
- 不运行 `git reset --hard`、`git clean`、强推、rebase 共享分支或删除分支，除非用户明确要求。
- 远程仓库已有脏改动时，保留并绕开。

## 导航

上一项：[weekly](weekly.html) · [返回首页](../index.html) · 下一项：[self](self.html)
