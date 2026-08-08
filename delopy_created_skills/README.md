# 本地沉淀 Skills

本目录用于存放从当前项目历史和对话中沉淀出的可复用 Agent Skills。每个 skill 都应保持聚焦，主要说明“何时使用”和“如何复用”，避免记录一次性任务细节。

## research-ui-frontend

- 作用：可迁移的 research/operations 前端风格启动包，用于任意新项目快速搭建同类界面气质。
- 何时使用：启动、迁移、设计或评审需要严肃技术产品感的前端，例如研究工具、实验面板、本地管理工具、数据密集型 dashboard。
- 覆盖范围：字体与字重、下划线规则、暖中性色浅色主题、安静深色主题、布局密度、标题与工具条层级、短页 footer 贴底、icons、logo/wordmark、按钮、表单、局部紧凑控件与输入按钮等高、inline metadata 编辑器、标题-description-路径卡片层级、状态标记、repo/project 聚合状态 badge、可移除 chip、危险操作确认 modal、drawers、紧凑批量输入行、配置键大小写去重与真实 API 标识保留、响应式检查。

## skillctl-local-backend

- 作用：总结 Skillctl 本地 Go 后端的安全和架构范式。
- 何时使用：修改或评审 `internal/`、`cmd/skillctl/` 中涉及 SQLite、HTTP API、Git source 同步、SKILL.md 扫描、软链接启用、project manifest 或 doctor 检查的代码时使用。
- 覆盖范围：本地 token 保护 API、SQLite schema/migration、非破坏性 Git 同步、多 remote checkout 的 remote/branch 状态展示、ahead 与 behind 语义区分、按 skill 路径标注本地/远程 Git 改动、幂等启用、拒绝覆盖用户文件、只删除受管理软链接、删除 source 时保留 checkout 并校验统一链接目录，以及 Claude/Codex skill 调用统计、JSONL 超长单行读取、rollout 增量扫描、排行快照和持久化缓存。

## skillctl-source-migration

- 作用：沉淀把已手工安装、well-known 来源或 personal repo 中 vendored 的第三方 skills 迁移成 Skillctl Git source 纳管的安全流程。
- 何时使用：需要从 `~/.agents/skills`、`~/.claude/skills`、`~/.codex/skills` 等全局目录梳理原始 skill 目录，或需要把 personal skill 仓库里误放的第三方 skill 删除并改由上游 GitHub 仓库维护时使用。
- 覆盖范围：source 分类、官方 source 确认、自建 skill 私有仓库打包、迁移前审批、旧目录和 lock 文件备份、通过 Skillctl source 逻辑添加仓库、精确删除旧全局目录或 vendored 目录、迁移已有 activation、清理安装锁记录、验证 DB/unified symlink/activation 状态。

## skillctl-local-packaging

- 作用：总结把 Skillctl 仓库构建成本机可执行命令和 macOS 发布压缩包的流程。
- 何时使用：安装、打包、发布或评审 `skillctl` CLI 时，尤其涉及 `web/dist` 嵌入 Go 二进制、`scripts/install-local.sh`、`scripts/package-macos.sh` 或 `dist/` 发布产物。
- 覆盖范围：前端先构建、单文件 Go 二进制安装到 PATH、从 upstream fast-forward 更新后重新安装、macOS arm64/amd64 打包、版本命名、产物忽略规则，以及安装命令和压缩包二进制验证。

## skill-packaging-patterns

- 作用：总结在本仓库中创建和维护 Agent Skill 文件夹的打包规则。
- 何时使用：把项目经验沉淀成新 skill，或检查 `delopy_created_skills/` 下已有 skill 是否结构清晰、触发条件明确、元数据完整时使用。
- 覆盖范围：`SKILL.md` frontmatter、`agents/openai.yaml`、命名规则、渐进披露、资源目录取舍、结构验证。

## conversation-pattern-miner

- 作用：总结对话结束后提炼可复用范式的流程。
- 何时使用：一次或多次项目对话后，判断是否需要新增或更新 skill，并维护本目录中文索引时使用。
- 覆盖范围：候选范式筛选、合并到已有 skill、新建 skill 的触发标准、中文 README 条目要求、验证清单。
