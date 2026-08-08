<p align="center">
  <img src="assets/skillctl-logo.svg" width="420" alt="Skillctl logo">
</p>

<h1 align="center">⚡ Skillctl</h1>

<p align="center">
  <strong>运行在 macOS 本机的 Agent Skills Web 管理工具。</strong>
</p>

<p align="center">
  <a href="README.md">English</a>
  ·
  <a href="https://github.com/nxZhai/Skill-Ctl/releases">Releases</a>
</p>

<p align="center">
  <a href="https://github.com/nxZhai/Skill-Ctl/releases"><img alt="Release" src="https://img.shields.io/github/v/release/nxZhai/Skill-Ctl?style=flat-square&color=2F6F68"></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-macOS-5B6575?style=flat-square">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React%20%2B%20Vite-19%20%2F%208-646CFF?style=flat-square&logo=react&logoColor=white">
  <img alt="Local first" src="https://img.shields.io/badge/local--first-SQLite-41B3A3?style=flat-square">
</p>

## ✨ 功能特性

- **Git Source 管理**：添加 GitHub 或其他 Git-compatible 仓库作为 skill source，并同步到本机。
- **递归发现 Skills**：扫描仓库中的 `SKILL.md`，记录每个 skill 的来源、路径、描述、标签和启用状态。
- **本地 Skills 盘点**：识别 Claude Code 或 Codex 现有本地 skill 目录，让 Skillctl 反映 agent 实际可见内容。
- **搜索与筛选**：按关键词、Source、Tag、Agent、启用状态筛选 skills。
- **标签与备注**：给仓库和 skill 添加标签、个人备注，便于维护大型 skill 库。
- **批量操作**：多选 skills 后批量添加/删除标签，或批量启用。
- **全局与项目启用**：支持启用到全局 agent 目录，也支持启用到项目级 agent 目录。
- **软链接部署**：通过受管理的 symlink 部署 skills，不复制真实文件。
- **项目 Manifest**：生成和应用项目级 `.skillctl.toml` manifest。
- **调用排行**：从本地 Codex/agent 历史中汇总可观测的 skill 调用情况。
- **Doctor 检查**：检查 Git、source checkout、本地修改、失效软链接和 manifest 引用。
- **单二进制本地应用**：Go 二进制内嵌 Vite 前端，启动本机 token 保护的 HTTP 服务。

## 🧭 核心概念

```text
Git repo 是同步和更新单位
Skill 是浏览、筛选和启用单位
软链接是部署方式
```

Skillctl 会把 source 仓库和启用目标分开管理。仓库克隆到本地数据目录，发现到的 skills 写入 SQLite 索引，启用时再通过受控软链接暴露给各个 agent。

## 🚀 命令

```bash
skillctl ui
skillctl doctor
skillctl rescan [source-id]
skillctl update [--check]
skillctl uninstall
skillctl version
```

`skillctl ui` 会初始化本地配置和数据目录，监听 `127.0.0.1` 的随机端口，生成随机 token，并自动打开浏览器。

## 📦 本地安装

把内嵌前端的 `skillctl` 二进制安装到 `~/.local/bin`：

```bash
./scripts/install-local.sh
```

如果 `~/.local/bin` 还不在 shell PATH 中，把下面这行加入 `~/.zshrc`：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

之后就可以在任意目录启动：

```bash
skillctl ui
```

如需安装到其他位置，传入 `PREFIX`：

```bash
PREFIX=/usr/local ./scripts/install-local.sh
```

## 🔄 更新

检查 GitHub Release 是否有新版本：

```bash
skillctl update --check
```

安装匹配当前 Mac 架构的最新 Release：

```bash
skillctl update
```

替换二进制前，Skillctl 会在 `~/.cache/skillctl/backups/` 创建压缩备份，其中包括配置、状态、受管理的仓库和统一 skills 软链接；并在更新前后对这些数据做哈希比对，如发现变动会报告错误。启动 `skillctl ui` 时，如果有新版本，也会在终端打印提示。

若你正在使用开发源码 checkout，可使用下面的方式拉取源码并重新安装：

```bash
./scripts/update-local.sh
```

## 🗑️ 卸载

```bash
skillctl uninstall
```

卸载会先移除所有由 Skillctl 记录的 Agent skills 软链接，并沿用 UI 的目标校验机制；然后询问是否删除 Skillctl 管理的本地 skill 仓库。选择否会保留仓库和本地状态，便于之后重新安装。

## 🗜️ 打包

生成 macOS `arm64` 和 `amd64` 发布压缩包到 `dist/`：

```bash
./scripts/package-macos.sh
```

如需覆盖压缩包文件名中的版本号，传入 `VERSION`：

```bash
VERSION=0.5.0 ./scripts/package-macos.sh
```

## 🛠️ 开发

先构建前端，以便 Go 将 `web/dist` 嵌入二进制：

```bash
cd web
npm install
npm run build
cd ..
go build -o skillctl ./cmd/skillctl
```

运行检查：

```bash
go test ./...
npm --prefix web run build
```

## 🗂️ 本地数据

Skillctl 使用以下目录：

```text
~/.config/skillctl/
~/.local/share/skillctl/
~/.cache/skillctl/
```

Git 仓库会克隆到 `~/.local/share/skillctl/repos/`。统一的 skill 入口软链接会创建在 `~/.local/share/skillctl/skills/`。

## 🔒 安全边界

Skillctl 只创建软链接，不覆盖普通文件或不属于 Skillctl 的软链接。关闭 skill 时，只删除 SQLite 中记录的 activation 链接，不删除真实 skill 文件。

Git 同步只使用 `fetch --prune` 和 `merge --ff-only`，不会执行 `git reset --hard`。本地 Web UI 只监听 `127.0.0.1`，所有 API 请求都必须携带启动时生成的随机 token。
