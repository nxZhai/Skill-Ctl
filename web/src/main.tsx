import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { marked } from "marked";
import DOMPurify from "dompurify";
import yaml from "js-yaml";
import "@fontsource/inter/400.css";
import "@fontsource/inter/500.css";
import "@fontsource/inter/600.css";
import "@fontsource/inter/700.css";
import "@fontsource/instrument-serif/400.css";
import "@fontsource/instrument-serif/400-italic.css";
import "@fontsource/jetbrains-mono/400.css";
import "@fontsource/jetbrains-mono/500.css";
import "./styles.css";

type AgentConfig = { user_dir: string; project_dir: string };
type ProjectRef = { alias: string; path: string };
type AppConfig = { agents: Record<string, AgentConfig>; projects: ProjectRef[]; repos_dir: string; skills_dir: string };
type Source = {
  id: string;
  url: string;
  branch: string;
  checkout_path: string;
  local_sha?: string;
  remote_sha?: string;
  last_fetch_at?: string;
  last_commit_at?: string;
  note?: string;
  pinned?: boolean;
  created_at: string;
  skill_count: number;
  status: string;
  message?: string;
  behind?: number;
  ahead?: number;
  remotes?: SourceRemote[];
  local_source?: boolean;
  local_path?: string;
  local_branch?: string;
};
type SourceRemote = {
  name: string;
  url: string;
  branch?: string;
  sha?: string;
  behind?: number;
  ahead?: number;
};
type Activation = {
  id: number;
  skill_id: string;
  agent: string;
  scope: "user" | "project";
  project_root?: string;
  link_path: string;
  created_at: string;
};
type Skill = {
  id: string;
  source_id: string;
  relative_path: string;
  name: string;
  description?: string;
  note?: string;
  content_sha?: string;
  discovered_at: string;
  tags?: string[];
  activations?: Activation[];
  local_changed?: boolean;
  remote_changed?: boolean;
};
type SkillUsageCounts = { claude: number; codex: number; total: number };
type SkillUsageItem = {
  skill_id: string;
  source_id: string;
  relative_path: string;
  name: string;
  counts: SkillUsageCounts;
};
type SkillUsageSummary = { generated_at: string; counts: Record<string, SkillUsageItem> };
type SkillUsageRanking = { generated_at: string; range: UsageRange; items: SkillUsageItem[] };
type SkillUsageSnapshot = { available: boolean; ranking: SkillUsageRanking };
type RenameSourceResult = { source: Source; skills: Skill[] };
type UsageRange = "day" | "week" | "month" | "all";
type LocalSkill = {
  id: string;
  agent: string;
  agent_root: string;
  root_key: string;
  root: string;
  relative_path: string;
  name: string;
  description?: string;
  content_sha?: string;
  is_symlink: boolean;
  symlink_path?: string;
  real_path?: string;
};
type SkillTreeEntry = { name: string; path: string; kind: "dir" | "file"; children?: SkillTreeEntry[] };
type SkillTree = { root: string; entries: SkillTreeEntry[] };
type DoctorCheck = { name: string; ok: boolean; message?: string; path?: string };
type FsListing = { path: string; parent: string; entries: { name: string; path: string }[] };
type Lang = "en" | "zh";
type ViewMode = "sources" | "local" | "usage";
type LocalSkillFilter = "all" | "symlink" | "direct";
type SourceBulkProgressKind = "check" | "sync";
type BulkProgressKind = SourceBulkProgressKind | "usage";
type BulkProgressStatus = "pending" | "running" | "done" | "error";
type BulkProgressItem = { id: string; label?: string; status: BulkProgressStatus; message?: string; needsSync?: boolean };
type BulkProgress = { id: number; kind: BulkProgressKind; items: BulkProgressItem[]; completed: boolean };
type Theme = "light" | "dark";
type ToastKind = "ok" | "error";
type Toast = { id: number; message: string; kind: ToastKind };
type PopupHistoryItem = Toast & { createdAt: string };
type PopupHistoryBallPosition = { side: "left" | "right"; y: number };

const popupHistoryLimit = 100;
const popupHistoryStorageKey = "skillctl-popup-history";
const popupHistoryBallStorageKey = "skillctl-popup-history-ball";
const usageRanges: UsageRange[] = ["day", "week", "month", "all"];

const messages = {
  en: {
    add: "Add",
    addAgent: "Add agent",
    addNote: "Add note",
    addProject: "Add project",
    addSource: "Add source",
    addTags: "Add tags",
    agent: "Agent",
    agentGlobalEnabled: "{agent}: Global",
    agentRepoEnabled: "All skills enabled for {agent}",
    agentSkillsDirectories: "Agent skills directories",
    appStorageDirectories: "Storage directories",
    appStorageDirectoriesDescription: "Default local folders for cloned repositories and unified skill links. Existing repo checkouts are not moved automatically.",
    agents: "Agents",
    allChecksPassed: "All checks passed",
    allTags: "All tags",
    alias: "Alias",
    appTitle: "Agent Skills Manager",
    branch: "Branch",
    bulkCheckTitle: "Checking repositories",
    bulkSyncTitle: "Syncing repositories",
    browse: "Browse...",
    cancel: "Cancel",
    check: "Check",
    checkAll: "Check all",
    chinese: "中文",
    clear: "Clear",
    close: "Close",
    closeCountdown: "Close ({seconds})",
    closeLinks: "Unlink",
    directory: "Directory",
    disable: "Disable",
    doctor: "Doctor",
    editNote: "Edit note",
    enable: "Enable",
    enabled: "Enabled",
    english: "EN",
    collapseDirectory: "Collapse {name}",
    customTags: "Custom tags",
    deleteSource: "Delete {name}",
    deleteSourceConfirmAction: "Delete repository",
    deleteSourceDescription: "\"{name}\" will be removed from Skillctl.",
    deleteSourceKeepDetails: "Files remain available at:",
    deleteSourceRemoveDetails: "Repository records, discovered skills, tags, activations, and managed symlinks.",
    deleteSourceTitle: "Delete repository?",
    deleteSourceWillKeep: "Local checkout will be kept",
    deleteSourceWillRemove: "Skillctl will remove",
    expandDirectory: "Expand {name}",
    filterByTag: "Filter by tag",
    gitUrl: "Git URL",
    global: "Global",
    goHome: "Go to home",
    loading: "Loading...",
    loadingDots: "Loading…",
    localSkillFilterAll: "All",
    localSkillFilterDirect: "Direct",
    localSkillFilterSymlink: "Symlink",
    localSkillPath: "Local path",
    localSkills: "Local Skills",
    localSkillsDescription: "Discover existing Claude Code or Codex skills directly from their local skill folders.",
    localSkillsEmpty: "No local skills found for this agent.",
    localSkillsFilterEmpty: "No matching local skills found for this agent.",
    localSkillsRoot: "Agent root",
    localSkillsStats: "{count} skills",
    name: "Name",
    noNote: "No note yet.",
    noSourcesBody: "Add a Git repository to discover local skills.",
    noSourcesTitle: "No sources yet",
    noSkillsSelected: "No skills selected. Select at least one tag to show skills.",
    noSubfolders: "No sub-folders here.",
    noTags: "No tags yet.",
    noneConfigured: "none configured",
    notEnabled: "Not enabled",
    noUsageRanking: "No observed skill usage in this range.",
    refreshUsage: "Refresh usage",
    bulkUsageTitle: "Refreshing usage",
    usageAutoRefreshDone: "Usage ranking refreshed",
    usageAutoRefreshIssues: "Usage ranking refreshed with {count} issue(s)",
    usageLastUpdated: "Last updated {time}",
    noteWordLimit: "{count} / 50 words",
    skillNotePlaceholder: "Add a personal note for this skill",
    observedUsage: "Observed usage",
    observedUsageDescription: "Counts explicit skill mentions and Codex-selected skills from local history logs.",
    observedUsageForAgent: "{agent}: {count}",
    pinSource: "Pin {name}",
    pinSkill: "Pin {name}",
    popupHistory: "Popup history",
    popupHistoryCount: "{count} records",
    popupHistoryEmpty: "No popup history yet.",
    popupHistoryError: "Error",
    popupHistoryOpen: "Open popup history",
    popupHistorySuccess: "Success",
    primaryNav: "Primary navigation",
    progressCount: "{done}/{total} done",
    progressDone: "Done",
    progressFailed: "Failed",
    progressPending: "Pending",
    progressRunning: "Running",
    progressWaitingSync: "{count} repos waiting to sync",
    project: "Project",
    projectEnabledCount: "{count} project activations",
    projectDir: "Project dir",
    projectDirDescription: "Where each agent reads skills. UserDir = global; ProjectDir = relative path inside a project.",
    projectPathPlaceholder: "~/projects/acme-api",
    projectScope: "Project",
    projectShortcuts: "Project shortcuts",
    projectShortcutsDescription: "Give a project directory a short alias so you can quickly pick it when enabling skills for a project.",
    remove: "Remove",
    removeTags: "Remove tags",
    remoteRepo: "Remote Repo",
    renameSource: "Rename local name",
    repoTags: "Repository tags",
    savedProject: "Saved project…",
    save: "Save",
    search: "Search",
    searchAll: "Search skills and repos",
    searchEmpty: "No matching skills or repos.",
    searchPlaceholder: "Search by skill, repo, tag, path...",
    searchRepos: "Repositories",
    searchResults: "{count} results",
    searchSkills: "Skills",
    selectAllSkills: "Select all skills in {name}",
    selectProjectDirectory: "Select project directory",
    selectSkill: "Select {name}",
    selectThisFolder: "Select this folder",
    settings: "Settings",
    reposDir: "Repos dir",
    skillsDir: "Skills dir",
    skillContents: "Skill contents",
    skillContentsDescription: "Directory structure for this skill. Click a folder or file to open it locally.",
    skillContentsUnavailable: "Directory structure unavailable.",
    skillMarkdown: "SKILL.md",
    skillSymlink: "Symlink",
    skillSymlinkPath: "Symlink path",
    localCommit: "Local",
    localSource: "Local source",
    localRepo: "Local Repo",
    lastUpdated: "Updated {time}",
    remoteCommit: "Remote",
    realSkillPath: "Real path",
    sourceId: "Local name",
    sources: "Sources",
    sourcesDescription: "Git repositories are the sync/update unit. Browse, tag, and enable skills per repo.",
    sourcesInventoryStats: "{repos} Repos · {skills} Skills",
    sourcesEnabledStats: "{global} Global Enabled · {project} Project Enabled",
    sourceView: "Sources",
    statusIssue: "Issue",
    statusLocalChanges: "Local changes",
    statusLocalSource: "Local source",
    statusOk: "OK",
    statusSyncFailed: "Sync failed",
    statusUpdateAvailable: "Update available",
    statusUpToDate: "Up to date",
    rescanIndex: "Rescan index",
    sync: "Sync",
    syncAll: "Sync all",
    switchTheme: "Switch theme",
    tags: "Tags",
    tagPlaceholder: "tags: writing, backend",
    tagPlaceholderSingle: "add tags: writing, backend",
    timeDaysAgo: "{count}d ago",
    timeHoursAgo: "{count}h ago",
    timeJustNow: "just now",
    timeMinutesAgo: "{count}m ago",
    upOneLevel: "../ (up one level)",
    unpinSource: "Unpin {name}",
    unpinSkill: "Unpin {name}",
    userDirGlobal: "User dir (global)",
    viewLocalSkills: "View local skills",
    viewSources: "View sources",
    viewUsage: "View usage",
    working: "Working",
    notePlaceholder: "What are the skills in this repo about?",
    noSkillsForTag: "No skills match this tag.",
    noSkillsDiscovered: "No skills discovered. Try Sync.",
    selectedCount: "{count} selected",
    sourceBranchBehind: "{branch} · {behind} behind",
    sourceRemoteAhead: "{count} local commits ahead",
    sourceRemoteAheadOne: "1 local commit ahead",
    sourceRemoteBehind: "{count} behind",
    sourceSkillCount: "{count} skills",
    sourceSummary: "{branch} · {behind} behind · {count} skills",
    issuesFound: "{count} issues found",
    switchLanguage: "Switch to Chinese",
    activationScope: "Activation scope",
    usageAll: "All time",
    usageDay: "1 day",
    usageMonth: "1 month",
    usageRanking: "Usage Ranking",
    usageRefreshHint: "Click refresh to scan local history.",
    usageRepo: "Repo",
    usageTotal: "{count} total",
    usageWeek: "1 week",
  },
  zh: {
    add: "添加",
    addAgent: "添加智能体",
    addNote: "添加备注",
    addProject: "添加项目",
    addSource: "添加源",
    addTags: "添加标签",
    agent: "智能体",
    agentGlobalEnabled: "{agent}：全局启用",
    agentRepoEnabled: "{agent} 已启用该 repo 的全部 skills",
    agentSkillsDirectories: "智能体技能目录",
    appStorageDirectories: "存储目录",
    appStorageDirectoriesDescription: "设置所有仓库 clone 和统一 skill 链接的默认本地目录。已有 repo checkout 不会自动移动。",
    agents: "智能体",
    allChecksPassed: "所有检查通过",
    allTags: "全部标签",
    alias: "别名",
    appTitle: "智能体技能管理器",
    branch: "分支",
    bulkCheckTitle: "正在检查仓库",
    bulkSyncTitle: "正在同步仓库",
    browse: "浏览...",
    cancel: "取消",
    check: "检查",
    checkAll: "全部检查",
    chinese: "中文",
    clear: "清空",
    close: "关闭",
    closeCountdown: "关闭（{seconds}）",
    closeLinks: "断开",
    directory: "目录",
    disable: "禁用",
    doctor: "诊断",
    editNote: "编辑备注",
    enable: "启用",
    enabled: "已启用",
    english: "EN",
    collapseDirectory: "收起 {name}",
    customTags: "自定义标签",
    deleteSource: "删除 {name}",
    deleteSourceConfirmAction: "删除 Repo",
    deleteSourceDescription: "“{name}”将从 Skillctl 中移除。",
    deleteSourceKeepDetails: "本地文件仍保留在：",
    deleteSourceRemoveDetails: "Repo 记录、已发现的 skills、标签、启用记录和由 Skillctl 管理的软链接。",
    deleteSourceTitle: "确认删除 Repo？",
    deleteSourceWillKeep: "本地 checkout 会保留",
    deleteSourceWillRemove: "Skillctl 将移除",
    expandDirectory: "展开 {name}",
    filterByTag: "按标签筛选",
    gitUrl: "Git 地址",
    global: "全局",
    goHome: "返回首页",
    loading: "加载中...",
    loadingDots: "加载中…",
    localSkillFilterAll: "全部",
    localSkillFilterDirect: "Direct",
    localSkillFilterSymlink: "Symlink",
    localSkillPath: "本地地址",
    localSkills: "本地 Skills",
    localSkillsDescription: "直接从 Claude Code 或 Codex 的本地 skill 目录中识别已有 skills。",
    localSkillsEmpty: "当前 agent 没有发现本地 skills。",
    localSkillsFilterEmpty: "当前 agent 没有匹配的本地 skills。",
    localSkillsRoot: "Agent 根目录",
    localSkillsStats: "{count} 个 skills",
    name: "名称",
    noNote: "暂无备注。",
    noSourcesBody: "添加一个 Git 仓库来发现本地技能。",
    noSourcesTitle: "暂无源",
    noSkillsSelected: "未选择任何技能。请至少选择一个标签来显示技能。",
    noSubfolders: "这里没有子文件夹。",
    noTags: "暂无标签。",
    noneConfigured: "未配置",
    notEnabled: "未启用",
    noUsageRanking: "该时间范围内没有可观测的 skill 调用记录。",
    refreshUsage: "更新调用统计",
    bulkUsageTitle: "正在更新调用统计",
    usageAutoRefreshDone: "调用统计已更新",
    usageAutoRefreshIssues: "调用统计已更新，{count} 个范围失败",
    usageLastUpdated: "上次更新 {time}",
    noteWordLimit: "{count} / 50 词",
    skillNotePlaceholder: "给这个 skill 添加个性化备注",
    observedUsage: "可观测调用",
    observedUsageDescription: "从本地历史日志中统计显式 skill 提及，以及 Codex 实际选择的 skills。",
    observedUsageForAgent: "{agent}：{count}",
    pinSource: "置顶 {name}",
    pinSkill: "固定 {name}",
    popupHistory: "弹窗历史记录",
    popupHistoryCount: "{count} 条记录",
    popupHistoryEmpty: "暂无弹窗历史记录。",
    popupHistoryError: "错误",
    popupHistoryOpen: "打开弹窗历史记录",
    popupHistorySuccess: "成功",
    primaryNav: "主导航",
    progressCount: "已完成 {done}/{total}",
    progressDone: "完成",
    progressFailed: "失败",
    progressPending: "等待中",
    progressRunning: "处理中",
    progressWaitingSync: "{count} 个 repo 等待更新同步",
    project: "项目",
    projectEnabledCount: "{count} 个项目启用",
    projectDir: "项目目录",
    projectDirDescription: "每个智能体读取技能的位置。UserDir 是全局目录；ProjectDir 是项目内的相对路径。",
    projectPathPlaceholder: "~/projects/acme-api",
    projectScope: "项目",
    projectShortcuts: "项目快捷方式",
    projectShortcutsDescription: "给项目目录设置短别名，启用项目技能时可以快速选择。",
    remove: "移除",
    removeTags: "移除标签",
    remoteRepo: "Remote Repo",
    renameSource: "修改本地名称",
    repoTags: "仓库标签",
    savedProject: "已保存项目…",
    save: "保存",
    search: "搜索",
    searchAll: "搜索 skills 和 repos",
    searchEmpty: "没有匹配的 skill 或 repo。",
    searchPlaceholder: "按 skill、repo、tag、路径搜索...",
    searchRepos: "仓库",
    searchResults: "{count} 条结果",
    searchSkills: "Skills",
    selectAllSkills: "选择 {name} 中的所有技能",
    selectProjectDirectory: "选择项目目录",
    selectSkill: "选择 {name}",
    selectThisFolder: "选择此文件夹",
    settings: "设置",
    reposDir: "Repos 目录",
    skillsDir: "Skills 目录",
    skillContents: "技能目录",
    skillContentsDescription: "该 skill 的目录结构。点击文件夹或文件可在本地打开。",
    skillContentsUnavailable: "目录结构不可用。",
    skillMarkdown: "SKILL.md",
    skillSymlink: "软链接",
    skillSymlinkPath: "软链接位置",
    localCommit: "本地",
    localSource: "本地来源",
    localRepo: "Local Repo",
    lastUpdated: "最近更新 {time}",
    remoteCommit: "远程",
    realSkillPath: "真实地址",
    sourceId: "本地名称",
    sources: "源",
    sourcesDescription: "Git 仓库是同步/更新单元。可按仓库浏览、标记并启用技能。",
    sourcesInventoryStats: "{repos} 个 Repos · {skills} 个 Skills",
    sourcesEnabledStats: "{global} 个 Global Enabled · {project} 个 Project Enabled",
    sourceView: "Sources",
    statusIssue: "问题",
    statusLocalChanges: "本地修改",
    statusLocalSource: "本地来源",
    statusOk: "正常",
    statusSyncFailed: "同步失败",
    statusUpdateAvailable: "可更新",
    statusUpToDate: "已最新",
    rescanIndex: "刷新索引",
    sync: "同步",
    syncAll: "全部同步",
    switchTheme: "切换主题",
    tags: "标签",
    tagPlaceholder: "标签：writing, backend",
    tagPlaceholderSingle: "添加标签：writing, backend",
    timeDaysAgo: "{count} 天前",
    timeHoursAgo: "{count} 小时前",
    timeJustNow: "刚刚",
    timeMinutesAgo: "{count} 分钟前",
    upOneLevel: "../（上一级）",
    unpinSource: "取消置顶 {name}",
    unpinSkill: "取消固定 {name}",
    userDirGlobal: "用户目录（全局）",
    viewLocalSkills: "查看本地 Skills",
    viewSources: "查看 Sources",
    viewUsage: "查看调用排行",
    working: "处理中",
    notePlaceholder: "这个仓库里的技能主要用于什么？",
    noSkillsForTag: "没有匹配该标签的技能。",
    noSkillsDiscovered: "尚未发现技能。请尝试同步。",
    selectedCount: "已选择 {count} 个",
    sourceBranchBehind: "{branch} · 落后 {behind}",
    sourceRemoteAhead: "领先 {count} 个本地提交",
    sourceRemoteAheadOne: "领先 1 个本地提交",
    sourceRemoteBehind: "落后 {count}",
    sourceSkillCount: "{count} 个技能",
    sourceSummary: "{branch} · 落后 {behind} · {count} 个技能",
    issuesFound: "发现 {count} 个问题",
    switchLanguage: "切换到英文",
    activationScope: "启用范围",
    usageAll: "全部时间",
    usageDay: "1 天",
    usageMonth: "1 个月",
    usageRanking: "调用次数排行榜",
    usageRefreshHint: "点击更新按钮后扫描本地历史记录。",
    usageRepo: "来源 Repo",
    usageTotal: "共 {count} 次",
    usageWeek: "1 周",
  },
} as const;

type TextKey = keyof typeof messages.en;
type Translate = (key: TextKey, vars?: Record<string, string | number>) => string;

function initialLang(): Lang {
  return localStorage.getItem("skillctl-lang") === "zh" ? "zh" : "en";
}

function initialTheme(): Theme {
  return localStorage.getItem("skillctl-theme") === "dark" ? "dark" : "light";
}

function initialPopupHistory(): PopupHistoryItem[] {
  try {
    const value = localStorage.getItem(popupHistoryStorageKey);
    if (!value) return [];
    const parsed = JSON.parse(value);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item) => {
      return item
        && typeof item.id === "number"
        && typeof item.message === "string"
        && (item.kind === "ok" || item.kind === "error")
        && typeof item.createdAt === "string";
    }).slice(0, popupHistoryLimit);
  } catch {
    return [];
  }
}

function popupBallFallbackY() {
  return clampPopupBallY(Math.round(window.innerHeight * 0.52));
}

function clampPopupBallY(y: number) {
  const margin = 12;
  const size = 56;
  const max = Math.max(margin, window.innerHeight - size - margin);
  return Math.min(max, Math.max(margin, y));
}

function initialPopupHistoryBallPosition(): PopupHistoryBallPosition {
  try {
    const value = localStorage.getItem(popupHistoryBallStorageKey);
    if (!value) return { side: "right", y: popupBallFallbackY() };
    const parsed = JSON.parse(value);
    return {
      side: parsed?.side === "left" ? "left" : "right",
      y: clampPopupBallY(typeof parsed?.y === "number" ? parsed.y : popupBallFallbackY()),
    };
  } catch {
    return { side: "right", y: popupBallFallbackY() };
  }
}

function translate(lang: Lang, key: TextKey, vars?: Record<string, string | number>) {
  let text: string = messages[lang][key];
  if (!vars) return text;
  for (const [name, value] of Object.entries(vars)) {
    text = text.split(`{${name}}`).join(String(value));
  }
  return text;
}

function scopeText(scope: string, t: Translate) {
  return scope === "project" ? t("project") : t("global");
}

function shortSha(sha?: string) {
  return sha ? sha.slice(0, 7) : "";
}

function formatRepoTime(value: string | undefined, lang: Lang, t: Translate) {
  if (!value) return "";
  const time = Date.parse(value);
  if (Number.isNaN(time)) return value;
  const diff = Date.now() - time;
  const minute = 60 * 1000;
  const hour = 60 * minute;
  const day = 24 * hour;
  if (diff >= 0 && diff < minute) return t("timeJustNow");
  if (diff >= 0 && diff < hour) return t("timeMinutesAgo", { count: Math.max(1, Math.floor(diff / minute)) });
  if (diff >= 0 && diff < day) return t("timeHoursAgo", { count: Math.floor(diff / hour) });
  if (diff >= 0 && diff < 7 * day) return t("timeDaysAgo", { count: Math.floor(diff / day) });
  return new Intl.DateTimeFormat(lang === "zh" ? "zh-CN" : "en-US", { year: "numeric", month: "short", day: "numeric" }).format(new Date(time));
}

function formatUsageTime(value: string, lang: Lang) {
  const time = Date.parse(value);
  if (Number.isNaN(time)) return value;
  return new Intl.DateTimeFormat(lang === "zh" ? "zh-CN" : "en-US", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(time));
}

function githubWebURL(raw: string) {
  const trimmed = raw.trim();
  const sshMatch = trimmed.match(/^git@github\.com:([^/]+)\/(.+?)(?:\.git)?$/);
  if (sshMatch) return `https://github.com/${sshMatch[1]}/${sshMatch[2]}`;
  const httpsMatch = trimmed.match(/^https:\/\/github\.com\/([^/]+)\/(.+?)(?:\.git)?$/);
  if (httpsMatch) return `https://github.com/${httpsMatch[1]}/${httpsMatch[2]}`;
  return trimmed;
}

function middleEllipsisParts(value: string, headChars = 32, tailChars = 22) {
  if (value.length <= headChars + tailChars + 3) return null;
  return {
    head: value.slice(0, headChars),
    tail: value.slice(value.length - tailChars),
  };
}

function MiddleEllipsisText({ value }: { value: string }) {
  const parts = middleEllipsisParts(value);
  if (!parts) {
    return (
      <span className="middleEllipsisText">
        <span className="middleEllipsisFull">{value}</span>
      </span>
    );
  }
  return (
    <span className="middleEllipsisText" aria-label={value}>
      <span className="middleEllipsisHead">{parts.head}</span>
      <span className="middleEllipsisDots" aria-hidden="true">...</span>
      <span className="middleEllipsisTail">{parts.tail}</span>
    </span>
  );
}

function repoNameFromURL(raw: string) {
  const trimmed = raw.trim().replace(/\/+$/, "");
  if (!trimmed) return "";
  const path = trimmed.includes(":") && !trimmed.includes("://") ? trimmed.split(":").pop() || "" : trimmed;
  const last = path.split("/").pop() || "";
  return last.replace(/\.git$/i, "").trim();
}

function cleanLocalRepoName(value: string) {
  return value.trim().replace(/[^a-zA-Z0-9._-]+/g, "-").replace(/^-+|-+$/g, "");
}

const sourceNoteWordLimit = 50;

function sourceNoteWordCount(value: string) {
  let count = 0;
  let inWord = false;
  for (const char of value) {
    if (/[\p{Script=Han}\p{Script=Hiragana}\p{Script=Katakana}\p{Script=Hangul}]/u.test(char)) {
      count += 1;
      inWord = false;
    } else if (/[\p{L}\p{N}]/u.test(char)) {
      if (!inWord) count += 1;
      inWord = true;
    } else {
      inWord = false;
    }
  }
  return count;
}

function canonicalAgentName(agent: string) {
  const normalized = agent.trim().toLowerCase();
  if (normalized === "claude-code") return "CLAUDE-Code";
  if (normalized === "codex") return "codex";
  return agent.trim();
}

function agentNames(agents: Record<string, AgentConfig>) {
  const byKey = new Map<string, { name: string; canonical: string; priority: number }>();
  for (const rawName of Object.keys(agents)) {
    const name = rawName.trim();
    if (!name) continue;
    const canonical = canonicalAgentName(name);
    const key = name.toLowerCase();
    const priority = name === canonical ? 2 : 1;
    const existing = byKey.get(key);
    if (!existing || priority > existing.priority) {
      byKey.set(key, { name, canonical, priority });
    }
  }
  return Array.from(byKey.values()).sort((a, b) => {
    const preferred = ["CLAUDE-Code", "codex"];
    const ai = preferred.indexOf(a.canonical);
    const bi = preferred.indexOf(b.canonical);
    if (ai !== -1 || bi !== -1) return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi);
    return a.canonical.localeCompare(b.canonical);
  }).map(({ name }) => name);
}

function activationAgentLabel(agent: string) {
  const canonical = canonicalAgentName(agent);
  return canonical === "CLAUDE-Code" ? "Claude-Code" : canonical;
}

function activationAgentClass(agent: string) {
  return canonicalAgentName(agent).toLowerCase().replace(/[^a-z0-9]+/g, "-");
}

function sortActivationAgents(agents: string[]) {
  const preferred = ["CLAUDE-Code", "codex"];
  return unique(agents.map(canonicalAgentName)).sort((a, b) => {
    const ai = preferred.indexOf(a);
    const bi = preferred.indexOf(b);
    if (ai !== -1 || bi !== -1) return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi);
    return a.localeCompare(b);
  });
}

function projectActivationCount(activations: Activation[]) {
  const projects = new Set<string>();
  for (const activation of activations) {
    if (activation.scope === "project") {
      projects.add(activation.project_root?.trim() || `activation-${activation.id}`);
    }
  }
  return projects.size;
}

function skillActivationSummary(skills: Skill[]) {
  let global = 0;
  let project = 0;
  for (const skill of skills) {
    const activations = skill.activations || [];
    if (activations.some((activation) => activation.scope !== "project")) global += 1;
    if (activations.some((activation) => activation.scope === "project")) project += 1;
  }
  return { skills: skills.length, global, project };
}

function sourceNeedsSync(source: Source) {
	if (source.local_source) return false;
  if (source.remotes?.length) return source.remotes.some((remote) => (remote.behind || 0) > 0);
  return (source.behind || 0) > 0 || (Boolean(source.local_sha) && Boolean(source.remote_sha) && source.local_sha !== source.remote_sha);
}

function sourceRemoteRows(source: Source): SourceRemote[] {
  if (source.remotes?.length) return source.remotes;
  return [{
    name: "origin",
    url: source.url,
    branch: source.branch,
    sha: source.remote_sha,
    ahead: source.ahead,
    behind: source.behind,
  }];
}

function sourceRemoteBranchMeta(remote: SourceRemote, fallbackBranch: string, t: Translate) {
  const parts = [remote.branch || fallbackBranch].filter(Boolean);
  if ((remote.ahead || 0) > 0) parts.push(remote.ahead === 1 ? t("sourceRemoteAheadOne") : t("sourceRemoteAhead", { count: remote.ahead || 0 }));
  if ((remote.behind || 0) > 0) parts.push(t("sourceRemoteBehind", { count: remote.behind || 0 }));
  return parts.join(" · ");
}

function initialPinnedSkills(): Record<string, string[]> {
  try {
    const value = localStorage.getItem("skillctl-pinned-skills");
    if (!value) return {};
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed as Record<string, string[]> : {};
  } catch {
    return {};
  }
}

const token = new URLSearchParams(window.location.search).get("token") || localStorage.getItem("skillctl-token") || "";
if (token) localStorage.setItem("skillctl-token", token);
const homeHref = token ? `/?token=${encodeURIComponent(token)}` : "/";

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      "X-Skillctl-Token": token,
      ...(init?.headers || {}),
    },
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data as T;
}

function updateBulkItem(current: BulkProgress | null, progressId: number, sourceId: string, patch: Partial<BulkProgressItem>) {
  if (!current || current.id !== progressId) return current;
  return {
    ...current,
    items: current.items.map((item) => (item.id === sourceId ? { ...item, ...patch } : item)),
  };
}

function usageRangeLabel(range: UsageRange, t: Translate) {
  switch (range) {
    case "day":
      return t("usageDay");
    case "month":
      return t("usageMonth");
    case "all":
      return t("usageAll");
    default:
      return t("usageWeek");
  }
}

function usageRankingToSummary(data: SkillUsageRanking): SkillUsageSummary {
  return {
    generated_at: data.generated_at,
    counts: Object.fromEntries(data.items.map((item) => [item.skill_id, item])),
  };
}

function App() {
  const [agents, setAgents] = useState<Record<string, AgentConfig>>({});
  const [projects, setProjects] = useState<ProjectRef[]>([]);
  const [reposDir, setReposDir] = useState("");
  const [skillsDir, setSkillsDir] = useState("");
  const [sources, setSources] = useState<Source[]>([]);
  const [skills, setSkills] = useState<Skill[]>([]);
  const [usageSummary, setUsageSummary] = useState<SkillUsageSummary | null>(null);
  const [usageRankings, setUsageRankings] = useState<Partial<Record<UsageRange, SkillUsageRanking>>>({});
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [popupHistory, setPopupHistory] = useState<PopupHistoryItem[]>(() => initialPopupHistory());
  const [popupHistoryOpen, setPopupHistoryOpen] = useState(false);
  const [bulkProgress, setBulkProgress] = useState<BulkProgress | null>(null);
  const [bulkRunning, setBulkRunning] = useState(false);
  const [doctor, setDoctor] = useState<DoctorCheck[] | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [activeSearchSkillId, setActiveSearchSkillId] = useState<string | null>(null);
  const [focusedSourceId, setFocusedSourceId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [busyLabel, setBusyLabel] = useState<string | null>(null);
  const [lang, setLang] = useState<Lang>(() => initialLang());
  const [theme, setTheme] = useState<Theme>(() => initialTheme());
  const [viewMode, setViewMode] = useState<ViewMode>("sources");
  const toastSeq = useRef(0);
  const bulkSeq = useRef(0);
  const bulkRunningRef = useRef(false);
  const usageStartupRefreshRef = useRef(false);
  const t: Translate = (key, vars) => translate(lang, key, vars);

  function toggleLang() {
    setLang((current) => {
      const next = current === "en" ? "zh" : "en";
      localStorage.setItem("skillctl-lang", next);
      return next;
    });
  }

  function toggleTheme() {
    setTheme((current) => {
      const next = current === "light" ? "dark" : "light";
      localStorage.setItem("skillctl-theme", next);
      return next;
    });
  }

  function dismissToast(id: number) {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }

  function pushToast(message: string, kind: Toast["kind"]) {
    const id = ++toastSeq.current;
    const toast = { id, message, kind };
    setToasts((prev) => [...prev, toast]);
    setPopupHistory((prev) => {
      const next = [{ ...toast, createdAt: new Date().toISOString() }, ...prev].slice(0, popupHistoryLimit);
      localStorage.setItem(popupHistoryStorageKey, JSON.stringify(next));
      return next;
    });
  }

  async function refresh() {
    const [cfg, sourceItems, skillItems] = await Promise.all([
      api<AppConfig>("/api/config"),
      api<Source[]>("/api/sources"),
      api<Skill[]>("/api/skills"),
    ]);
    setAgents(cfg.agents || {});
    setProjects(cfg.projects || []);
    setReposDir(cfg.repos_dir || "");
    setSkillsDir(cfg.skills_dir || "");
    setSources(sourceItems || []);
    setSkills(skillItems || []);
  }

  function applyUsageRanking(data: SkillUsageRanking) {
    setUsageRankings((prev) => ({ ...prev, [data.range]: data }));
    setUsageSummary(usageRankingToSummary(data));
  }

  async function run(label: string, fn: () => Promise<unknown>) {
    setLoading(true);
    setBusyLabel(label);
    try {
      await fn();
      await refresh();
      pushToast(`${label} completed`, "ok");
    } catch (err) {
      pushToast(err instanceof Error ? err.message : String(err), "error");
    } finally {
      setLoading(false);
      setBusyLabel(null);
    }
  }

  async function refreshUsageOnStartup() {
    if (usageStartupRefreshRef.current || bulkRunningRef.current) return;
    usageStartupRefreshRef.current = true;
    const progressId = ++bulkSeq.current;
    const items = usageRanges.map((range) => ({
      id: range,
      label: usageRangeLabel(range, t),
      status: "pending" as BulkProgressStatus,
    }));
    bulkRunningRef.current = true;
    setBulkProgress({ id: progressId, kind: "usage", items, completed: false });
    setBulkRunning(true);
    let failures = 0;
    try {
      for (const range of usageRanges) {
        setBulkProgress((current) => updateBulkItem(current, progressId, range, { status: "running", message: undefined }));
        try {
          const data = await api<SkillUsageRanking>(`/api/usage/ranking?range=${encodeURIComponent(range)}`);
          applyUsageRanking(data);
          setBulkProgress((current) => updateBulkItem(current, progressId, range, { status: "done", message: undefined }));
        } catch (err) {
          failures += 1;
          const message = err instanceof Error ? err.message : String(err);
          setBulkProgress((current) => updateBulkItem(current, progressId, range, { status: "error", message }));
        }
      }
      setBulkProgress((current) => (current?.id === progressId ? { ...current, completed: true } : current));
      pushToast(failures ? t("usageAutoRefreshIssues", { count: failures }) : t("usageAutoRefreshDone"), failures ? "error" : "ok");
    } finally {
      bulkRunningRef.current = false;
      setBulkRunning(false);
    }
  }

  async function runAllSources(kind: SourceBulkProgressKind) {
    if (bulkRunningRef.current) return;
    const label = kind === "check" ? "Check all sources" : "Sync all sources";
    const items = sources.map((source) => ({ id: source.id, status: "pending" as BulkProgressStatus }));
    const progressId = ++bulkSeq.current;
    bulkRunningRef.current = true;
    setBulkProgress({ id: progressId, kind, items, completed: false });
    setBulkRunning(true);
    let failures = 0;
    try {
      for (const source of sources) {
        setBulkProgress((current) => updateBulkItem(current, progressId, source.id, { status: "running", message: undefined }));
        try {
          if (kind === "check") {
            const updated = await api<Source>(`/api/sources/${encodeURIComponent(source.id)}/check`, { method: "POST" });
            setBulkProgress((current) => updateBulkItem(current, progressId, source.id, { needsSync: sourceNeedsSync(updated) }));
          } else {
            await api<{ source: Source; skills: Skill[] }>(`/api/sources/${encodeURIComponent(source.id)}/sync`, { method: "POST" });
          }
          setBulkProgress((current) => updateBulkItem(current, progressId, source.id, { status: "done", message: undefined }));
        } catch (err) {
          failures += 1;
          const message = err instanceof Error ? err.message : String(err);
          setBulkProgress((current) => updateBulkItem(current, progressId, source.id, { status: "error", message }));
        }
      }
      await refresh();
      setBulkProgress((current) => (current?.id === progressId ? { ...current, completed: true } : current));
      pushToast(failures ? `${label} completed with ${failures} issue${failures === 1 ? "" : "s"}` : `${label} completed`, failures ? "error" : "ok");
    } finally {
      bulkRunningRef.current = false;
      setBulkRunning(false);
    }
  }

  async function renameSource(oldId: string, newId: string): Promise<RenameSourceResult> {
    const label = `Rename ${oldId}`;
    setLoading(true);
    setBusyLabel(label);
    try {
      const result = await api<RenameSourceResult>(`/api/sources/${encodeURIComponent(oldId)}/rename`, {
        method: "POST",
        body: JSON.stringify({ id: newId }),
      });
      setSources((prev) => prev.map((source) => (source.id === oldId ? result.source : source)));
      setSkills((prev) => [...prev.filter((skill) => skill.source_id !== oldId), ...(result.skills || [])]);
      await refresh();
      pushToast(`${label} completed`, "ok");
      return result;
    } catch (err) {
      pushToast(err instanceof Error ? err.message : String(err), "error");
      throw err;
    } finally {
      setLoading(false);
      setBusyLabel(null);
    }
  }

  async function deleteSource(id: string): Promise<void> {
    const label = t("deleteSource", { name: id });
    setLoading(true);
    setBusyLabel(label);
    try {
      await api(`/api/sources/${encodeURIComponent(id)}`, { method: "DELETE" });
      await refresh();
      pushToast(`${label} completed`, "ok");
    } catch (err) {
      pushToast(err instanceof Error ? err.message : String(err), "error");
      throw err;
    } finally {
      setLoading(false);
      setBusyLabel(null);
    }
  }

  useEffect(() => {
    refresh()
      .catch((err) => pushToast(err instanceof Error ? err.message : String(err), "error"))
      .finally(() => {
        void refreshUsageOnStartup();
      });
  }, []);

  async function runDoctor() {
    await run("Doctor", async () => {
      setDoctor(await api<DoctorCheck[]>("/api/doctor"));
    });
  }

  const focusSource = useCallback((sourceId: string) => {
    setViewMode("sources");
    setSearchOpen(false);
    setFocusedSourceId(sourceId);
  }, []);

  const activeSearchSkill = activeSearchSkillId ? skills.find((skill) => skill.id === activeSearchSkillId) || null : null;

  return (
    <div className="app" data-theme={theme}>
      <header className="topbar">
        <a className="brandHome" href={homeHref} aria-label={t("goHome")}>
          <h1>{t("appTitle")}</h1>
        </a>
        <nav aria-label={t("primaryNav")}>
          <button className="searchToggle" type="button" aria-label={t("searchAll")} title={t("search")} onClick={() => setSearchOpen(true)}>
            <svg className="materialSearchIcon" viewBox="0 -960 960 960" aria-hidden="true">
              <path d="M784-120 532-372q-30 24-69 38t-83 14q-109 0-184.5-75.5T120-580q0-109 75.5-184.5T380-840q109 0 184.5 75.5T640-580q0 44-14 83t-38 69l252 252-56 56ZM380-400q75 0 127.5-52.5T560-580q0-75-52.5-127.5T380-760q-75 0-127.5 52.5T200-580q0 75 52.5 127.5T380-400Z" />
            </svg>
          </button>
          <button className="langToggle" type="button" aria-label={t("switchLanguage")} onClick={toggleLang}>
            <svg className="materialTranslateIcon" xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" aria-hidden="true">
              <path d="m476-80 182-480h84L924-80h-84l-43-122H603L560-80h-84ZM160-200l-56-56 202-202q-35-35-63.5-80T190-640h84q20 39 40 68t48 58q33-33 68.5-92.5T484-720H40v-80h280v-80h80v80h280v80H564q-21 72-63 148t-83 116l96 98-30 82-122-125-202 201Zm468-72h144l-72-204-72 204Z" />
            </svg>
          </button>
          <button
            className="themeToggle"
            type="button"
            aria-label={t("switchTheme")}
            title={theme === "light" ? "Dark mode" : "Light mode"}
            onClick={toggleTheme}
          >
            {theme === "light" ? (
              <svg viewBox="0 0 20 20" aria-hidden="true">
                <path d="M15.6 12.9A6.5 6.5 0 0 1 7.1 4.4a6.8 6.8 0 1 0 8.5 8.5Z" fill="currentColor" />
              </svg>
            ) : (
              <svg viewBox="0 0 20 20" aria-hidden="true">
                <circle cx="10" cy="10" r="3.2" fill="currentColor" />
                <path d="M10 1.8v2.1M10 16.1v2.1M18.2 10h-2.1M3.9 10H1.8M15.8 4.2l-1.5 1.5M5.7 14.3l-1.5 1.5M15.8 15.8l-1.5-1.5M5.7 5.7 4.2 4.2" fill="none" stroke="currentColor" strokeLinecap="round" strokeWidth="1.7" />
              </svg>
            )}
          </button>
          <button className="doctorToggle" type="button" aria-label={t("doctor")} title={t("doctor")} onClick={runDoctor}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M11 2v2" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
              <path d="M5 2v2" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
              <path d="M5 3H4a2 2 0 0 0-2 2v4a6 6 0 0 0 12 0V5a2 2 0 0 0-2-2h-1" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
              <path d="M8 15a6 6 0 0 0 12 0v-3" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
              <circle cx="20" cy="10" r="2" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
            </svg>
          </button>
          <button className="settingsToggle" type="button" aria-label={t("settings")} title={t("settings")} onClick={() => setSettingsOpen(true)}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 15.4a3.4 3.4 0 1 0 0-6.8 3.4 3.4 0 0 0 0 6.8Z" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
              <path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06a2.05 2.05 0 0 1-2.9 2.9l-.06-.06a1.7 1.7 0 0 0-1.88-.34 1.7 1.7 0 0 0-1.03 1.56V21a2.05 2.05 0 0 1-4.1 0v-.09a1.7 1.7 0 0 0-1.03-1.56 1.7 1.7 0 0 0-1.88.34l-.06.06a2.05 2.05 0 0 1-2.9-2.9l.06-.06A1.7 1.7 0 0 0 4.4 15a1.7 1.7 0 0 0-1.56-1.03H2.75a2.05 2.05 0 0 1 0-4.1h.09A1.7 1.7 0 0 0 4.4 8.84a1.7 1.7 0 0 0-.34-1.88L4 6.9A2.05 2.05 0 0 1 6.9 4l.06.06a1.7 1.7 0 0 0 1.88.34 1.7 1.7 0 0 0 1.03-1.56V2.75a2.05 2.05 0 0 1 4.1 0v.09A1.7 1.7 0 0 0 15 4.4a1.7 1.7 0 0 0 1.88-.34L16.94 4a2.05 2.05 0 0 1 2.9 2.9l-.06.06a1.7 1.7 0 0 0-.34 1.88 1.7 1.7 0 0 0 1.56 1.03h.09a2.05 2.05 0 0 1 0 4.1H21a1.7 1.7 0 0 0-1.6 1.03Z" fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
            </svg>
          </button>
        </nav>
      </header>

      {loading && <div className="topProgress" role="progressbar" aria-label={t("working")} />}
      {settingsOpen && (
        <SettingsModal
          t={t}
          agents={agents}
          projects={projects}
          reposDir={reposDir}
          skillsDir={skillsDir}
          onClose={() => setSettingsOpen(false)}
          onSaved={(cfg) => {
            setAgents(cfg.agents);
            setProjects(cfg.projects);
            setReposDir(cfg.repos_dir);
            setSkillsDir(cfg.skills_dir);
          }}
          run={run}
        />
      )}
      {searchOpen && (
        <SearchModal
          t={t}
          sources={sources}
          skills={skills}
          onClose={() => setSearchOpen(false)}
          onOpenSource={focusSource}
          onOpenSkill={(skillId) => {
            setSearchOpen(false);
            setActiveSearchSkillId(skillId);
          }}
        />
      )}
      {activeSearchSkill && (
        <SkillModal
          t={t}
          skill={activeSearchSkill}
          agents={agents}
          run={run}
          onClose={() => setActiveSearchSkillId(null)}
        />
      )}

      <main>
        <div className="viewTabs" role="tablist" aria-label={t("primaryNav")}>
          <button
            type="button"
            role="tab"
            className={viewMode === "sources" ? "active" : ""}
            aria-selected={viewMode === "sources"}
            onClick={() => setViewMode("sources")}
          >
            {t("viewSources")}
          </button>
          <button
            type="button"
            role="tab"
            className={viewMode === "local" ? "active" : ""}
            aria-selected={viewMode === "local"}
            onClick={() => setViewMode("local")}
          >
            {t("viewLocalSkills")}
          </button>
          <button
            type="button"
            role="tab"
            className={viewMode === "usage" ? "active" : ""}
            aria-selected={viewMode === "usage"}
            onClick={() => setViewMode("usage")}
          >
            {t("viewUsage")}
          </button>
        </div>
        {viewMode === "sources" && (
          <SourcesPage t={t} lang={lang} sources={sources} skills={skills} usageSummary={usageSummary} agents={agents} projects={projects} run={run} runAllSources={runAllSources} renameSource={renameSource} deleteSource={deleteSource} focusedSourceId={focusedSourceId} onSourceFocused={() => setFocusedSourceId(null)} busyLabel={busyLabel} bulkRunning={bulkRunning} />
        )}
        {viewMode === "local" && (
          <LocalSkillsPage t={t} run={run} />
        )}
        {viewMode === "usage" && (
          <UsageRankingPage
            t={t}
            lang={lang}
            skills={skills}
            agents={agents}
            run={run}
            refreshedRankings={usageRankings}
            onUsageLoaded={applyUsageRanking}
          />
        )}
      </main>

      <footer className="appFooter">SKILLCTL v0.5.0 @ Nicy</footer>

      <div className="overlayStack">
        {doctor && <DoctorPanel t={t} checks={doctor} onClose={() => setDoctor(null)} />}
        {bulkProgress && <BulkProgressPanel progress={bulkProgress} t={t} onClose={() => setBulkProgress(null)} />}
        <div className="toastStack" role="status" aria-live="polite">
          {toasts.map((toast) => (
            <ToastItem key={toast.id} toast={toast} t={t} onDismiss={() => dismissToast(toast.id)} />
          ))}
        </div>
      </div>
      <PopupHistoryBall t={t} count={popupHistory.length} onOpen={() => setPopupHistoryOpen(true)} />
      {popupHistoryOpen && <PopupHistoryModal t={t} lang={lang} items={popupHistory} onClose={() => setPopupHistoryOpen(false)} />}
    </div>
  );
}

function Spinner() {
  return <span className="buttonIconSlot spinnerSlot" aria-hidden="true"><span className="spinner" /></span>;
}

type ButtonIconName =
  | "add"
  | "browse"
  | "cancel"
  | "check"
  | "clear"
  | "close"
  | "disable"
  | "edit"
  | "enable"
  | "folder"
  | "pin"
  | "remove"
  | "repoAdd"
  | "save"
  | "select"
  | "sync"
  | "tagAdd"
  | "tagRemove"
  | "unlink"
  | "up";

function ButtonIcon({ name }: { name: ButtonIconName }) {
  switch (name) {
    case "add":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="M10 4.5v11M4.5 10h11" />
        </svg>
      );
    case "browse":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="M3.6 6.4h5.1l1.3 1.7h6.4v6.4a1.6 1.6 0 0 1-1.6 1.6H5.2a1.6 1.6 0 0 1-1.6-1.6V6.4Z" />
          <path d="M3.6 7.9V5.6A1.6 1.6 0 0 1 5.2 4h3l1.3 1.7h5.3a1.6 1.6 0 0 1 1.6 1.6v.6" />
        </svg>
      );
    case "cancel":
    case "close":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="m5.6 5.6 8.8 8.8M14.4 5.6l-8.8 8.8" />
        </svg>
      );
    case "check":
      return (
        <svg className="buttonIcon materialButtonIcon" viewBox="0 -960 960 960" aria-hidden="true">
          <path d="m424-296 282-282-56-56-226 226-114-114-56 56 170 170Zm56 216q-83 0-156-31.5T197-197q-54-54-85.5-127T80-480q0-83 31.5-156T197-763q54-54 127-85.5T480-880q83 0 156 31.5T763-763q54 54 85.5 127T880-480q0 83-31.5 156T763-197q-54 54-127 85.5T480-80Zm0-80q134 0 227-93t93-227q0-134-93-227t-227-93q-134 0-227 93t-93 227q0 134 93 227t227 93Zm0-320Z" />
        </svg>
      );
    case "clear":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="M4.4 13.1 10.9 6.6a2 2 0 0 1 2.8 0l1.7 1.7a2 2 0 0 1 0 2.8l-4.6 4.6H5.6l-1.2-1.2a1 1 0 0 1 0-1.4Z" />
          <path d="m8.2 9.3 4.1 4.1M4.8 16h10.4" />
        </svg>
      );
    case "disable":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <circle cx="10" cy="10" r="6.1" />
          <path d="m5.8 14.2 8.4-8.4" />
        </svg>
      );
    case "edit":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="m5 13.9-.8 2.2 2.2-.8 8.3-8.3-1.4-1.4-8.3 8.3Z" />
          <path d="m12.4 6.5 1.4 1.4" />
        </svg>
      );
    case "enable":
    case "select":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <circle cx="10" cy="10" r="6.4" />
          <path d="m6.9 10.2 2 2 4.3-4.5" />
        </svg>
      );
    case "folder":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="M3.7 6.2h4.6l1.2 1.5h6.8v7.1a1.5 1.5 0 0 1-1.5 1.5H5.2a1.5 1.5 0 0 1-1.5-1.5V6.2Z" />
          <path d="M3.7 7.6V5.7a1.5 1.5 0 0 1 1.5-1.5h2.9l1.2 1.5h5.5a1.5 1.5 0 0 1 1.5 1.5v.4" />
        </svg>
      );
    case "pin":
      return (
        <svg className="buttonIcon" viewBox="0 0 16 16" aria-hidden="true">
          <path fill="currentColor" d="M5.8 2.1h4.4l-.45 4.2 2.05 2.05v1.1H8.6V14H7.4V9.45H4.2v-1.1L6.25 6.3 5.8 2.1Zm1.35 1.2.36 3.38-1.57 1.57h4.12L8.49 6.68l.36-3.38h-1.7Z" />
        </svg>
      );
    case "remove":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="M5.6 7.1h8.8M8.4 7.1v7.1M11.6 7.1v7.1M7.1 7.1l.5 9h4.8l.5-9M8.2 5.1h3.6M8.9 3.9h2.2l.7 1.2H8.2l.7-1.2Z" />
        </svg>
      );
    case "repoAdd":
      return (
        <svg className="buttonIcon materialButtonIcon" viewBox="0 -960 960 960" aria-hidden="true">
          <path d="M560-320h80v-80h80v-80h-80v-80h-80v80h-80v80h80v80ZM160-160q-33 0-56.5-23.5T80-240v-480q0-33 23.5-56.5T160-800h240l80 80h320q33 0 56.5 23.5T880-640v400q0 33-23.5 56.5T800-160H160Zm0-80h640v-400H447l-80-80H160v480Zm0 0v-480 480Z" />
        </svg>
      );
    case "save":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="M5 4.4h8.1L15 6.3v9.3H5V4.4Z" />
          <path d="M7.2 4.4v4h5.1v-4M7.4 15.6v-4.1h5.2v4.1" />
        </svg>
      );
    case "sync":
      return (
        <svg className="buttonIcon materialButtonIcon" viewBox="0 -960 960 960" aria-hidden="true">
          <path d="M280-120 80-320l200-200 57 56-104 104h607v80H233l104 104-57 56Zm400-320-57-56 104-104H120v-80h607L623-784l57-56 200 200-200 200Z" />
        </svg>
      );
    case "tagAdd":
    case "tagRemove":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="M4.4 5.2v4.5l5.7 5.7 5.3-5.3-5.7-5.7H5.2a.8.8 0 0 0-.8.8Z" />
          <circle cx="7.2" cy="7.2" r="0.8" />
          {name === "tagAdd" ? <path d="M11.3 9.7v4M9.3 11.7h4" /> : <path d="M9.3 11.7h4" />}
        </svg>
      );
    case "unlink":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="M7.4 12.6 6.2 14a2.7 2.7 0 1 1-3.8-3.8L5 7.6a2.7 2.7 0 0 1 3.8 0M12.6 7.4 13.8 6a2.7 2.7 0 1 1 3.8 3.8L15 12.4a2.7 2.7 0 0 1-3.8 0M5.4 5.4l9.2 9.2" />
        </svg>
      );
    case "up":
      return (
        <svg className="buttonIcon" viewBox="0 0 20 20" aria-hidden="true">
          <path d="m5.2 10 4.8-4.8 4.8 4.8M10 5.2v9.6" />
        </svg>
      );
  }
}

function IconButtonContent({ children, icon }: { children: React.ReactNode; icon: ButtonIconName }) {
  return (
    <span className="buttonContent">
      <span className="buttonIconSlot">
        <ButtonIcon name={icon} />
      </span>
      <span className="buttonText">{children}</span>
    </span>
  );
}

function DeleteSourceModal({
  source,
  busy,
  onClose,
  onConfirm,
  t,
}: {
  source: Source;
  busy: boolean;
  onClose: () => void;
  onConfirm: () => void;
  t: Translate;
}) {
  const titleId = `delete-source-${source.id}-title`;
  const descriptionId = `delete-source-${source.id}-description`;

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape" && !busy) onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [busy, onClose]);

  return (
    <div className="modalOverlay" onClick={() => { if (!busy) onClose(); }}>
      <div
        className="modal deleteSourceModal"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descriptionId}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modalHead deleteSourceModalHead">
          <div className="deleteSourceHeading">
            <span className="deleteSourceHeadingIcon" aria-hidden="true">
              <ButtonIcon name="remove" />
            </span>
            <div>
              <h2 id={titleId}>{t("deleteSourceTitle")}</h2>
              <p id={descriptionId}>{t("deleteSourceDescription", { name: source.id })}</p>
            </div>
          </div>
          <button type="button" disabled={busy} onClick={onClose}>
            <IconButtonContent icon="close">{t("close")}</IconButtonContent>
          </button>
        </div>
        <div className="modalBody deleteSourceModalBody">
          <div className="deleteSourceImpact remove">
            <span className="deleteSourceImpactIcon" aria-hidden="true">
              <ButtonIcon name="remove" />
            </span>
            <span className="deleteSourceImpactText">
              <strong>{t("deleteSourceWillRemove")}</strong>
              <span>{t("deleteSourceRemoveDetails")}</span>
            </span>
          </div>
          <div className="deleteSourceImpact keep">
            <span className="deleteSourceImpactIcon" aria-hidden="true">
              <ButtonIcon name="folder" />
            </span>
            <span className="deleteSourceImpactText">
              <strong>{t("deleteSourceWillKeep")}</strong>
              <span>{t("deleteSourceKeepDetails")}</span>
              <span className="mono deleteSourcePath" title={source.checkout_path}>{source.checkout_path}</span>
            </span>
          </div>
        </div>
        <div className="deleteSourceModalActions">
          <button type="button" autoFocus disabled={busy} onClick={onClose}>
            {t("cancel")}
          </button>
          <button className="danger deleteSourceConfirmButton" type="button" disabled={busy} onClick={onConfirm}>
            {busy && <Spinner />}
            <IconButtonContent icon="remove">{t("deleteSourceConfirmAction")}</IconButtonContent>
          </button>
        </div>
      </div>
    </div>
  );
}

function SearchModal({
  t,
  sources,
  skills,
  onClose,
  onOpenSource,
  onOpenSkill,
}: {
  t: Translate;
  sources: Source[];
  skills: Skill[];
  onClose: () => void;
  onOpenSource: (sourceId: string) => void;
  onOpenSkill: (skillId: string) => void;
}) {
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const normalizedQuery = query.trim().toLowerCase();
  const sourceMatches = useMemo(() => {
    if (!normalizedQuery) return sources.slice(0, 8);
    return sources
      .filter((source) => [
        source.id,
        source.url,
        source.branch,
        source.checkout_path,
        source.note || "",
        source.status,
      ].some((value) => value.toLowerCase().includes(normalizedQuery)))
      .slice(0, 12);
  }, [normalizedQuery, sources]);
  const skillMatches = useMemo(() => {
    const list = normalizedQuery
      ? skills.filter((skill) => [
        skill.name,
        skill.id,
        skill.source_id,
        skill.relative_path,
        skill.description || "",
        ...(skill.tags || []),
      ].some((value) => value.toLowerCase().includes(normalizedQuery)))
      : skills.slice(0, 12);
    return list.slice(0, 24);
  }, [normalizedQuery, skills]);
  const resultCount = sourceMatches.length + skillMatches.length;

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="modalOverlay" onClick={onClose}>
      <div className="modal searchModal" role="dialog" aria-modal="true" aria-label={t("searchAll")} onClick={(event) => event.stopPropagation()}>
        <div className="modalHead searchModalHead">
          <div>
            <h2>{t("searchAll")}</h2>
            <p>{t("searchResults", { count: resultCount })}</p>
          </div>
          <button type="button" onClick={onClose}>
            <IconButtonContent icon="close">{t("close")}</IconButtonContent>
          </button>
        </div>
        <div className="modalBody searchModalBody">
          <label className="searchField">
            {t("search")}
            <span className="searchInputWrap">
              <svg className="searchInputIcon" viewBox="0 -960 960 960" aria-hidden="true">
                <path d="M784-120 532-372q-30 24-69 38t-83 14q-109 0-184.5-75.5T120-580q0-109 75.5-184.5T380-840q109 0 184.5 75.5T640-580q0 44-14 83t-38 69l252 252-56 56ZM380-400q75 0 127.5-52.5T560-580q0-75-52.5-127.5T380-760q-75 0-127.5 52.5T200-580q0 75 52.5 127.5T380-400Z" />
              </svg>
              <input
                ref={inputRef}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder={t("searchPlaceholder")}
              />
            </span>
          </label>

          {resultCount === 0 ? (
            <div className="searchEmpty">{t("searchEmpty")}</div>
          ) : (
            <div className="searchResultsGrid">
              <section className="searchResultSection">
                <div className="searchResultHead">
                  <h3>{t("searchRepos")}</h3>
                  <span>{sourceMatches.length}</span>
                </div>
                <div className="searchResultList">
                  {sourceMatches.length === 0 && <span className="muted small">{t("searchEmpty")}</span>}
                  {sourceMatches.map((source) => (
                    <button className="searchResultItem" type="button" key={source.id} onClick={() => onOpenSource(source.id)}>
                      <span className="searchResultKind">{t("sources")}</span>
                      <span className="searchResultMain">
                        <strong>{source.id}</strong>
                        <span className="mono">{source.url}</span>
                      </span>
                      <span className="searchResultMeta">{t("sourceSkillCount", { count: source.skill_count })}</span>
                    </button>
                  ))}
                </div>
              </section>

              <section className="searchResultSection">
                <div className="searchResultHead">
                  <h3>{t("searchSkills")}</h3>
                  <span>{skillMatches.length}</span>
                </div>
                <div className="searchResultList">
                  {skillMatches.length === 0 && <span className="muted small">{t("searchEmpty")}</span>}
                  {skillMatches.map((skill) => (
                    <button className="searchResultItem" type="button" key={skill.id} onClick={() => onOpenSkill(skill.id)}>
                      <span className="searchResultKind">{t("skillMarkdown")}</span>
                      <span className="searchResultMain">
                        <strong>{skill.name}</strong>
                        <span className="mono">{skill.source_id} · {skill.relative_path}</span>
                      </span>
                      {skill.tags?.length ? <span className="searchResultMeta">{skill.tags.slice(0, 3).join(", ")}</span> : null}
                    </button>
                  ))}
                </div>
              </section>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function AgentIcon({ agent }: { agent: string }) {
  const canonical = canonicalAgentName(agent);
  if (canonical === "CLAUDE-Code") {
    return (
      <svg viewBox="0 0 16 16" aria-hidden="true">
        <path d="m3.127 10.604 3.135-1.76.053-.153-.053-.085H6.11l-.525-.032-1.791-.048-1.554-.065-1.505-.08-.38-.081L0 7.832l.036-.234.32-.214.455.04 1.009.069 1.513.105 1.097.064 1.626.17h.259l.036-.105-.089-.065-.068-.064-1.566-1.062-1.695-1.121-.887-.646-.48-.327-.243-.306-.104-.67.435-.48.585.04.15.04.593.456 1.267.981 1.654 1.218.242.202.097-.068.012-.049-.109-.181-.9-1.626-.96-1.655-.428-.686-.113-.411a2 2 0 0 1-.068-.484l.496-.674L4.446 0l.662.089.279.242.411.94.666 1.48 1.033 2.014.302.597.162.553.06.17h.105v-.097l.085-1.134.157-1.392.154-1.792.052-.504.25-.605.497-.327.387.186.319.456-.045.294-.19 1.23-.37 1.93-.243 1.29h.142l.161-.16.654-.868 1.097-1.372.484-.545.565-.601.363-.287h.686l.505.751-.226.775-.707.895-.585.759-.839 1.13-.524.904.048.072.125-.012 1.897-.403 1.024-.186 1.223-.21.553.258.06.263-.218.536-1.307.323-1.533.307-2.284.54-.028.02.032.04 1.029.098.44.024h1.077l2.005.15.525.346.315.424-.053.323-.807.411-3.631-.863-.872-.218h-.12v.073l.726.71 1.331 1.202 1.667 1.55.084.383-.214.302-.226-.032-1.464-1.101-.565-.497-1.28-1.077h-.084v.113l.295.432 1.557 2.34.08.718-.112.234-.404.141-.444-.08-.911-1.28-.94-1.44-.759-1.291-.093.053-.448 4.821-.21.246-.484.186-.403-.307-.214-.496.214-.98.258-1.28.21-1.016.19-1.263.112-.42-.008-.028-.092.012-.953 1.307-1.448 1.957-1.146 1.227-.274.109-.477-.247.045-.44.266-.39 1.586-2.018.956-1.25.617-.723-.004-.105h-.036l-4.212 2.736-.75.096-.324-.302.04-.496.154-.162 1.267-.871z" />
      </svg>
    );
  }
  return (
    <svg viewBox="0 0 16 16" aria-hidden="true">
      <path d="M14.949 6.547a3.94 3.94 0 0 0-.348-3.273 4.11 4.11 0 0 0-4.4-1.934A4.1 4.1 0 0 0 8.423.2 4.15 4.15 0 0 0 6.305.086a4.1 4.1 0 0 0-1.891.948 4.04 4.04 0 0 0-1.158 1.753 4.1 4.1 0 0 0-1.563.679A4 4 0 0 0 .554 4.72a3.99 3.99 0 0 0 .502 4.731 3.94 3.94 0 0 0 .346 3.274 4.11 4.11 0 0 0 4.402 1.933c.382.425.852.764 1.377.995.526.231 1.095.35 1.67.346 1.78.002 3.358-1.132 3.901-2.804a4.1 4.1 0 0 0 1.563-.68 4 4 0 0 0 1.14-1.253 3.99 3.99 0 0 0-.506-4.716m-6.097 8.406a3.05 3.05 0 0 1-1.945-.694l.096-.054 3.23-1.838a.53.53 0 0 0 .265-.455v-4.49l1.366.778q.02.011.025.035v3.722c-.003 1.653-1.361 2.992-3.037 2.996m-6.53-2.75a2.95 2.95 0 0 1-.36-2.01l.095.057L5.29 12.09a.53.53 0 0 0 .527 0l3.949-2.246v1.555a.05.05 0 0 1-.022.041L6.473 13.3c-1.454.826-3.311.335-4.15-1.098m-.85-6.94A3.02 3.02 0 0 1 3.07 3.949v3.785a.51.51 0 0 0 .262.451l3.93 2.237-1.366.779a.05.05 0 0 1-.048 0L2.585 9.342a2.98 2.98 0 0 1-1.113-4.094zm11.216 2.571L8.747 5.576l1.362-.776a.05.05 0 0 1 .048 0l3.265 1.86a3 3 0 0 1 1.173 1.207 2.96 2.96 0 0 1-.27 3.2 3.05 3.05 0 0 1-1.36.997V8.279a.52.52 0 0 0-.276-.445m1.36-2.015-.097-.057-3.226-1.855a.53.53 0 0 0-.53 0L6.249 6.153V4.598a.04.04 0 0 1 .019-.04L9.533 2.7a3.07 3.07 0 0 1 3.257.139c.474.325.843.778 1.066 1.303.223.526.289 1.103.191 1.664zM5.503 8.575 4.139 7.8a.05.05 0 0 1-.026-.037V4.049c0-.57.166-1.127.476-1.607s.752-.864 1.275-1.105a3.08 3.08 0 0 1 3.234.41l-.096.054-3.23 1.838a.53.53 0 0 0-.265.455zm.742-1.577 1.758-1 1.762 1v2l-1.755 1-1.762-1z" />
    </svg>
  );
}

function SkillActivationMarkers({ activations, t }: { activations: Activation[]; t: Translate }) {
  const globalAgents = sortActivationAgents(activations.filter((activation) => activation.scope !== "project").map((activation) => activation.agent));
  const projectCount = projectActivationCount(activations);
  if (!globalAgents.length && projectCount === 0) return null;
  return (
    <span className="activationMarkers" aria-label={t("enabled")}>
      {globalAgents.map((agent) => (
        <span
          className={`agentMarker ${activationAgentClass(agent)}`}
          key={agent}
          title={t("agentGlobalEnabled", { agent: activationAgentLabel(agent) })}
          aria-label={t("agentGlobalEnabled", { agent: activationAgentLabel(agent) })}
        >
          <AgentIcon agent={agent} />
        </span>
      ))}
      {projectCount > 0 && (
        <span
          className="projectActivationMarker"
          title={t("projectEnabledCount", { count: projectCount })}
          aria-label={t("projectEnabledCount", { count: projectCount })}
        >
          <span className="projectActivationDot" aria-hidden="true" />
          <span className="projectActivationCount">{projectCount}</span>
        </span>
      )}
    </span>
  );
}

function sourceFullyEnabledAgents(sourceSkills: Skill[]) {
  if (sourceSkills.length === 0) return [];
  const candidates = new Set<string>();
  for (const skill of sourceSkills) {
    for (const activation of skill.activations || []) {
      const agent = canonicalAgentName(activation.agent);
      if (agent) candidates.add(agent);
    }
  }
  const enabledAgents = Array.from(candidates).filter((agent) =>
    sourceSkills.every((skill) => (skill.activations || []).some((activation) => canonicalAgentName(activation.agent) === agent)),
  );
  return sortActivationAgents(enabledAgents);
}

function SourceActivationMarkers({ agents, t }: { agents: string[]; t: Translate }) {
  if (agents.length === 0) return null;
  return (
    <span className="sourceActivationMarkers" aria-label={t("enabled")}>
      {agents.map((agent) => (
        <span
          className={`agentMarker sourceAgentMarker ${activationAgentClass(agent)}`}
          key={agent}
          title={t("agentRepoEnabled", { agent: activationAgentLabel(agent) })}
          aria-label={t("agentRepoEnabled", { agent: activationAgentLabel(agent) })}
        >
          <AgentIcon agent={agent} />
        </span>
      ))}
    </span>
  );
}

function SkillUsageMarkers({ usage, t }: { usage?: SkillUsageItem; t: Translate }) {
  if (!usage || usage.counts.total <= 0) return null;
  const items = [
    { agent: "CLAUDE-Code", count: usage.counts.claude },
    { agent: "codex", count: usage.counts.codex },
  ].filter((item) => item.count > 0);
  return (
    <span className="usageMarkers" aria-label={t("observedUsage")} title={t("observedUsage")}>
      {items.map((item) => (
        <span
          className={`usageMarker ${activationAgentClass(item.agent)}`}
          key={item.agent}
          title={t("observedUsageForAgent", { agent: activationAgentLabel(item.agent), count: item.count })}
          aria-label={t("observedUsageForAgent", { agent: activationAgentLabel(item.agent), count: item.count })}
        >
          <span className="usageMarkerIcon" aria-hidden="true"><AgentIcon agent={item.agent} /></span>
          <span className="usageMarkerCount">{item.count}</span>
        </span>
      ))}
    </span>
  );
}

type CustomSelectOption = { value: string; label: string };

function CustomSelect({
  ariaLabel,
  value,
  placeholder,
  options,
  onChange,
}: {
  ariaLabel: string;
  value: string;
  placeholder: string;
  options: CustomSelectOption[];
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const selected = options.find((option) => option.value === value);

  useEffect(() => {
    if (!open) return;
    function handlePointerDown(event: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
        setOpen(false);
      }
    }
    window.addEventListener("mousedown", handlePointerDown);
    window.addEventListener("keydown", handleKeyDown, true);
    return () => {
      window.removeEventListener("mousedown", handlePointerDown);
      window.removeEventListener("keydown", handleKeyDown, true);
    };
  }, [open]);

  return (
    <div className={`customSelect${open ? " open" : ""}`} ref={rootRef}>
      <button
        type="button"
        className="customSelectButton"
        aria-label={ariaLabel}
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={options.length === 0}
        onClick={() => setOpen((current) => !current)}
      >
        <span className={`customSelectValue${selected ? "" : " placeholder"}`}>
          {selected?.label || placeholder}
        </span>
        <span className="customSelectCaret" aria-hidden="true">
          <svg viewBox="0 0 14 9">
            <path d="M2 2l5 5 5-5" />
          </svg>
        </span>
      </button>
      {open && (
        <div className="customSelectMenu" role="listbox" aria-label={ariaLabel}>
          {options.map((option) => {
            const isSelected = option.value === value;
            return (
              <button
                type="button"
                key={option.value}
                className={`customSelectOption${isSelected ? " selected" : ""}`}
                role="option"
                aria-selected={isSelected}
                onClick={() => {
                  onChange(option.value);
                  setOpen(false);
                }}
              >
                <span className="customSelectCheck" aria-hidden="true">
                  {isSelected ? "✓" : ""}
                </span>
                <span className="customSelectOptionLabel">{option.label}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

function DirPicker({
  initialPath,
  onPick,
  onClose,
  t,
}: {
  initialPath: string;
  onPick: (path: string) => void;
  onClose: () => void;
  t: Translate;
}) {
  const [listing, setListing] = useState<FsListing | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function load(path: string) {
    setLoading(true);
    setError("");
    try {
      setListing(await api<FsListing>(`/api/fs?path=${encodeURIComponent(path)}`));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load(initialPath);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="modalOverlay" onClick={onClose}>
      <div className="modal dirPicker" role="dialog" aria-modal="true" aria-label={t("selectProjectDirectory")} onClick={(e) => e.stopPropagation()}>
        <div className="modalHead">
          <div>
            <h2>{t("selectProjectDirectory")}</h2>
            <p className="mono small">{listing?.path || initialPath || "~"}</p>
          </div>
          <button onClick={onClose}><IconButtonContent icon="close">{t("close")}</IconButtonContent></button>
        </div>
        <div className="modalBody">
          {error && <p className="warning">{error}</p>}
          <div className="dirList">
            {listing?.parent && (
              <button className="dirItem up" onClick={() => load(listing.parent)}>
                <IconButtonContent icon="up">{t("upOneLevel")}</IconButtonContent>
              </button>
            )}
            {listing?.entries.map((entry) => (
              <button key={entry.path} className="dirItem" onClick={() => load(entry.path)}>
                <IconButtonContent icon="folder">{entry.name}/</IconButtonContent>
              </button>
            ))}
            {listing && listing.entries.length === 0 && <p className="muted small">{t("noSubfolders")}</p>}
            {loading && <p className="muted small">{t("loadingDots")}</p>}
          </div>
          <div className="dirPickerActions">
            <button onClick={onClose}><IconButtonContent icon="cancel">{t("cancel")}</IconButtonContent></button>
            <button className="primary" disabled={!listing?.path} onClick={() => listing && onPick(listing.path)}>
              <IconButtonContent icon="select">{t("selectThisFolder")}</IconButtonContent>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function SourceNote({
  source,
  run,
  busyLabel,
  t,
}: {
  source: Source;
  run: (label: string, fn: () => Promise<unknown>) => Promise<void>;
  busyLabel: string | null;
  t: Translate;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(source.note || "");
  const note = (source.note || "").trim();
  const label = `Save note ${source.id}`;
  const saving = busyLabel === label;
  const wordCount = sourceNoteWordCount(draft);
  const overLimit = wordCount > sourceNoteWordLimit;

  function startEdit() {
    setDraft(source.note || "");
    setEditing(true);
  }

  async function save() {
    let ok = false;
    await run(label, async () => {
      await api(`/api/sources/${encodeURIComponent(source.id)}/note`, {
        method: "POST",
        body: JSON.stringify({ note: draft.trim().replace(/\s+/g, " ") }),
      });
      ok = true;
    });
    if (ok) setEditing(false);
  }

  if (editing) {
    const limitId = `source-note-${source.id}-limit`;
    return (
      <form
        className="sourceNote editing"
        onSubmit={(event) => {
          event.preventDefault();
          if (!saving && !overLimit) void save();
        }}
      >
        <label className="sourceNoteInput">
          <input
            value={draft}
            autoFocus
            aria-label={t("notePlaceholder")}
            aria-describedby={limitId}
            aria-invalid={overLimit}
            placeholder={t("notePlaceholder")}
            onChange={(e) => setDraft(e.target.value)}
          />
          <span id={limitId} className={`sourceNoteLimit${overLimit ? " over" : ""}`}>
            {t("noteWordLimit", { count: wordCount })}
          </span>
        </label>
        <div className="sourceNoteActions">
          <button className="primary" type="submit" aria-label={t("save")} disabled={saving || overLimit}>
            {saving && <Spinner />}
            <IconButtonContent icon="save">{t("save")}</IconButtonContent>
          </button>
          <button type="button" aria-label={t("cancel")} disabled={saving} onClick={() => setEditing(false)}>
            <IconButtonContent icon="cancel">{t("cancel")}</IconButtonContent>
          </button>
        </div>
      </form>
    );
  }

  return (
    <div className="sourceNote">
      {note ? <p className="sourceNoteText">{note}</p> : <p className="muted sourceNoteEmpty">{t("noNote")}</p>}
      <button type="button" className="linkBtn" onClick={startEdit}>
        <IconButtonContent icon={note ? "edit" : "add"}>{note ? t("editNote") : t("addNote")}</IconButtonContent>
      </button>
    </div>
  );
}

function SkillNote({
  skill,
  run,
  busyLabel,
  t,
}: {
  skill: Skill;
  run: (label: string, fn: () => Promise<unknown>) => Promise<void>;
  busyLabel: string | null;
  t: Translate;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(skill.note || "");
  const note = (skill.note || "").trim();
  const label = `Save skill note ${skill.id}`;
  const saving = busyLabel === label;
  const wordCount = sourceNoteWordCount(draft);
  const overLimit = wordCount > sourceNoteWordLimit;

  function startEdit() {
    setDraft(skill.note || "");
    setEditing(true);
  }

  async function save() {
    let ok = false;
    await run(label, async () => {
      await api(`/api/skills/${encodeURIComponent(skill.id)}/note`, {
        method: "POST",
        body: JSON.stringify({ note: draft.trim().replace(/\s+/g, " ") }),
      });
      ok = true;
    });
    if (ok) setEditing(false);
  }

  if (editing) {
    return (
      <form
        className="skillNote editing"
        onSubmit={(event) => {
          event.preventDefault();
          if (!saving && !overLimit) void save();
        }}
      >
        <label className="sourceNoteInput skillNoteInput">
          <input
            value={draft}
            autoFocus
            aria-label={t("skillNotePlaceholder")}
            aria-invalid={overLimit}
            onChange={(e) => setDraft(e.target.value)}
          />
        </label>
        <div className="sourceNoteActions skillNoteActions">
          <button className="primary" type="submit" aria-label={t("save")} disabled={saving || overLimit}>
            {saving ? <Spinner /> : <ButtonIcon name="save" />}
          </button>
          <button type="button" aria-label={t("cancel")} disabled={saving} onClick={() => setEditing(false)}>
            <ButtonIcon name="cancel" />
          </button>
        </div>
      </form>
    );
  }

  return (
    <div className={`skillNote${note ? "" : " empty"}`}>
      {note && <span className="skillNoteText" title={note}>{note}</span>}
      <button type="button" className="linkBtn skillNoteButton" onClick={startEdit}>
        <IconButtonContent icon={note ? "edit" : "add"}>{note ? t("editNote") : t("addNote")}</IconButtonContent>
      </button>
    </div>
  );
}

type AgentSettingsRow = { name: string; user_dir: string; project_dir: string };

function settingsAgentName(name: string) {
  const trimmed = name.trim();
  const key = trimmed.toLowerCase();
  if (key === "claude-code") return "Claude-Code";
  if (key === "codex") return "Codex";
  return trimmed;
}

function normalizeAgentSettingsRows(rows: AgentSettingsRow[]) {
  const byKey = new Map<string, AgentSettingsRow & { priority: number }>();
  for (const row of rows) {
    const canonicalName = settingsAgentName(row.name);
    if (!canonicalName) continue;
    const key = canonicalName.toLowerCase();
    const priority = row.name.trim() === canonicalName ? 2 : 1;
    const existing = byKey.get(key);
    if (!existing || priority > existing.priority) {
      byKey.set(key, {
        name: canonicalName,
        user_dir: row.user_dir || existing?.user_dir || "",
        project_dir: row.project_dir || existing?.project_dir || "",
        priority,
      });
    } else {
      existing.user_dir = existing.user_dir || row.user_dir;
      existing.project_dir = existing.project_dir || row.project_dir;
    }
  }
  return Array.from(byKey.values())
    .sort((a, b) => {
      const preferred = ["Claude-Code", "Codex"];
      const ai = preferred.indexOf(a.name);
      const bi = preferred.indexOf(b.name);
      if (ai !== -1 || bi !== -1) return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi);
      return a.name.localeCompare(b.name);
    })
    .map(({ priority, ...row }) => row);
}

function SettingsModal({
  t,
  agents,
  projects,
  reposDir,
  skillsDir,
  onClose,
  onSaved,
  run,
}: {
  t: Translate;
  agents: Record<string, AgentConfig>;
  projects: ProjectRef[];
  reposDir: string;
  skillsDir: string;
  onClose: () => void;
  onSaved: (cfg: AppConfig) => void;
  run: (label: string, fn: () => Promise<unknown>) => Promise<void>;
}) {
  const [agentList, setAgentList] = useState(
    normalizeAgentSettingsRows(Object.entries(agents).map(([name, cfg]) => ({ name, user_dir: cfg.user_dir, project_dir: cfg.project_dir }))),
  );
  const [projectList, setProjectList] = useState<ProjectRef[]>(projects.map((p) => ({ ...p })));
  const [storage, setStorage] = useState({ reposDir, skillsDir });
  const [pickerFor, setPickerFor] = useState<"repos" | "skills" | number | null>(null);

  function updateAgent(i: number, field: "name" | "user_dir" | "project_dir", value: string) {
    setAgentList((prev) => prev.map((a, idx) => (idx === i ? { ...a, [field]: value } : a)));
  }
  function updateProject(i: number, field: "alias" | "path", value: string) {
    setProjectList((prev) => prev.map((p, idx) => (idx === i ? { ...p, [field]: value } : p)));
  }

  async function save() {
    const agentsMap: Record<string, AgentConfig> = {};
    for (const a of normalizeAgentSettingsRows(agentList)) {
      agentsMap[a.name] = { user_dir: a.user_dir.trim(), project_dir: a.project_dir.trim() };
    }
    const projs = projectList
      .map((p) => ({ alias: p.alias.trim(), path: p.path.trim() }))
      .filter((p) => p.alias && p.path);
    let ok = false;
    await run("Save settings", async () => {
      const cfg = await api<AppConfig>("/api/config", {
        method: "POST",
        body: JSON.stringify({
          agents: agentsMap,
          projects: projs,
          repos_dir: storage.reposDir.trim(),
          skills_dir: storage.skillsDir.trim(),
        }),
      });
      onSaved({
        agents: cfg.agents || {},
        projects: cfg.projects || [],
        repos_dir: cfg.repos_dir || "",
        skills_dir: cfg.skills_dir || "",
      });
      ok = true;
    });
    if (ok) onClose();
  }

  return (
    <div className="modalOverlay" onClick={onClose}>
      <div className="modal settingsModal" role="dialog" aria-modal="true" aria-label={t("settings")} onClick={(e) => e.stopPropagation()}>
        <div className="modalHead">
          <h2>{t("settings")}</h2>
          <button onClick={onClose}><IconButtonContent icon="close">{t("close")}</IconButtonContent></button>
        </div>
        <div className="modalBody">
          <div className="controlBlock">
            <h4>{t("appStorageDirectories")}</h4>
            <p className="muted small">{t("appStorageDirectoriesDescription")}</p>
            <div className="settingsTable">
              <div className="settingsRow storageRow settingsHead">
                <span>{t("name")}</span>
                <span>{t("directory")}</span>
                <span />
              </div>
              <div className="settingsRow storageRow">
                <span className="settingsLabel">{t("reposDir")}</span>
                <span className="projectField">
                  <input value={storage.reposDir} onChange={(e) => setStorage((prev) => ({ ...prev, reposDir: e.target.value }))} />
                  <button type="button" onClick={() => setPickerFor("repos")}>
                    <IconButtonContent icon="browse">{t("browse")}</IconButtonContent>
                  </button>
                </span>
                <span />
              </div>
              <div className="settingsRow storageRow">
                <span className="settingsLabel">{t("skillsDir")}</span>
                <span className="projectField">
                  <input value={storage.skillsDir} onChange={(e) => setStorage((prev) => ({ ...prev, skillsDir: e.target.value }))} />
                  <button type="button" onClick={() => setPickerFor("skills")}>
                    <IconButtonContent icon="browse">{t("browse")}</IconButtonContent>
                  </button>
                </span>
                <span />
              </div>
            </div>
          </div>

          <hr className="modalDivider" />

          <div className="controlBlock">
            <h4>{t("agentSkillsDirectories")}</h4>
            <p className="muted small">{t("projectDirDescription")}</p>
            <div className="settingsTable">
              <div className="settingsRow settingsHead">
                <span>{t("agent")}</span>
                <span>{t("userDirGlobal")}</span>
                <span>{t("projectDir")}</span>
                <span />
              </div>
              {agentList.map((a, i) => (
                <div className="settingsRow" key={i}>
                  <input placeholder="CLAUDE-Code" value={a.name} onChange={(e) => updateAgent(i, "name", e.target.value)} />
                  <input placeholder="~/.claude/skills" value={a.user_dir} onChange={(e) => updateAgent(i, "user_dir", e.target.value)} />
                  <input placeholder=".claude/skills" value={a.project_dir} onChange={(e) => updateAgent(i, "project_dir", e.target.value)} />
                  <button className="danger" type="button" onClick={() => setAgentList((prev) => prev.filter((_, idx) => idx !== i))}>
                    <IconButtonContent icon="remove">{t("remove")}</IconButtonContent>
                  </button>
                </div>
              ))}
            </div>
            <button type="button" onClick={() => setAgentList((prev) => [...prev, { name: "", user_dir: "", project_dir: "" }])}>
              <IconButtonContent icon="add">{t("addAgent")}</IconButtonContent>
            </button>
          </div>

          <hr className="modalDivider" />

          <div className="controlBlock">
            <h4>{t("projectShortcuts")}</h4>
            <p className="muted small">{t("projectShortcutsDescription")}</p>
            <div className="settingsTable">
              <div className="settingsRow projectRow settingsHead">
                <span>{t("alias")}</span>
                <span>{t("directory")}</span>
                <span />
              </div>
              {projectList.map((p, i) => (
                <div className="settingsRow projectRow" key={i}>
                  <input placeholder="acme-api" value={p.alias} onChange={(e) => updateProject(i, "alias", e.target.value)} />
                  <span className="projectField">
                    <input placeholder={t("projectPathPlaceholder")} value={p.path} onChange={(e) => updateProject(i, "path", e.target.value)} />
                    <button type="button" onClick={() => setPickerFor(i)}>
                      <IconButtonContent icon="browse">{t("browse")}</IconButtonContent>
                    </button>
                  </span>
                  <button className="danger" type="button" onClick={() => setProjectList((prev) => prev.filter((_, idx) => idx !== i))}>
                    <IconButtonContent icon="remove">{t("remove")}</IconButtonContent>
                  </button>
                </div>
              ))}
            </div>
            <button type="button" onClick={() => setProjectList((prev) => [...prev, { alias: "", path: "" }])}>
              <IconButtonContent icon="add">{t("addProject")}</IconButtonContent>
            </button>
          </div>

          <div className="dirPickerActions">
            <button type="button" onClick={onClose}><IconButtonContent icon="cancel">{t("cancel")}</IconButtonContent></button>
            <button className="primary" type="button" onClick={save}><IconButtonContent icon="save">{t("save")}</IconButtonContent></button>
          </div>
        </div>
      </div>
      {pickerFor !== null && (
        <DirPicker
          t={t}
          initialPath={typeof pickerFor === "number" ? projectList[pickerFor]?.path || "" : pickerFor === "repos" ? storage.reposDir : storage.skillsDir}
          onPick={(path) => {
            if (pickerFor === "repos") {
              setStorage((prev) => ({ ...prev, reposDir: path }));
            } else if (pickerFor === "skills") {
              setStorage((prev) => ({ ...prev, skillsDir: path }));
            } else {
              updateProject(pickerFor, "path", path);
            }
            setPickerFor(null);
          }}
          onClose={() => setPickerFor(null)}
        />
      )}
    </div>
  );
}

function ToastItem({ toast, onDismiss, t }: { toast: Toast; onDismiss: () => void; t: Translate }) {
  const [remaining, setRemaining] = useState(3);
  const [leaving, setLeaving] = useState(false);
  const dismissedRef = useRef(false);

  function beginDismiss() {
    if (dismissedRef.current) return;
    dismissedRef.current = true;
    setLeaving(true);
  }

  useEffect(() => {
    if (leaving) return;
    const timer = window.setTimeout(() => {
      if (remaining <= 1) {
        beginDismiss();
      } else {
        setRemaining((r) => r - 1);
      }
    }, 1000);
    return () => window.clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [remaining, leaving]);

  useEffect(() => {
    if (!leaving) return;
    const timer = window.setTimeout(onDismiss, 260);
    return () => window.clearTimeout(timer);
  }, [leaving, onDismiss]);

  return (
    <div className={`toast ${toast.kind}${leaving ? " leaving" : ""}`}>
      <span className="toastMsg">{toast.message}</span>
      <button className="toastClose" onClick={beginDismiss}>
        <IconButtonContent icon="close">{t("closeCountdown", { seconds: remaining })}</IconButtonContent>
      </button>
    </div>
  );
}

function PopupHistoryBall({ count, onOpen, t }: { count: number; onOpen: () => void; t: Translate }) {
  const [position, setPosition] = useState<PopupHistoryBallPosition>(() => initialPopupHistoryBallPosition());
  const [dragPosition, setDragPosition] = useState<{ x: number; y: number } | null>(null);
  const ballRef = useRef<HTMLButtonElement>(null);
  const dragPositionRef = useRef<{ x: number; y: number } | null>(null);
  const dragOffset = useRef({ x: 0, y: 0 });
  const dragStartedAt = useRef({ x: 0, y: 0 });
  const activePointerId = useRef<number | null>(null);
  const cleanupDragListeners = useRef<(() => void) | null>(null);
  const moved = useRef(false);

  useEffect(() => {
    function handleResize() {
      setPosition((current) => {
        const next = { ...current, y: clampPopupBallY(current.y) };
        localStorage.setItem(popupHistoryBallStorageKey, JSON.stringify(next));
        return next;
      });
    }

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  useEffect(() => {
    return () => cleanupDragListeners.current?.();
  }, []);

  function savePosition(next: PopupHistoryBallPosition) {
    setPosition(next);
    localStorage.setItem(popupHistoryBallStorageKey, JSON.stringify(next));
  }

  function cleanupDrag() {
    cleanupDragListeners.current?.();
    cleanupDragListeners.current = null;
    activePointerId.current = null;
  }

  function updateDragPosition(clientX: number, clientY: number) {
    const size = 56;
    const margin = 12;
    const rawX = clientX - dragOffset.current.x;
    const rawY = clientY - dragOffset.current.y;
    const x = Math.min(window.innerWidth - size - margin, Math.max(margin, rawX));
    const y = clampPopupBallY(rawY);
    const distance = Math.hypot(clientX - dragStartedAt.current.x, clientY - dragStartedAt.current.y);
    if (distance > 4) moved.current = true;
    const nextDragPosition = { x, y };
    dragPositionRef.current = nextDragPosition;
    setDragPosition(nextDragPosition);
  }

  function finishDrag(pointerId: number) {
    if (activePointerId.current !== null && activePointerId.current !== pointerId) return;
    const current = dragPositionRef.current;
    if (ballRef.current?.hasPointerCapture(pointerId)) {
      ballRef.current.releasePointerCapture(pointerId);
    }
    cleanupDrag();
    dragPositionRef.current = null;
    setDragPosition(null);
    if (!current) return;
    const side = current.x + 28 < window.innerWidth / 2 ? "left" : "right";
    savePosition({ side, y: clampPopupBallY(current.y) });
  }

  function cancelDrag(pointerId?: number) {
    if (pointerId !== undefined && activePointerId.current !== null && activePointerId.current !== pointerId) return;
    cleanupDrag();
    dragPositionRef.current = null;
    setDragPosition(null);
  }

  function handlePointerDown(event: React.PointerEvent<HTMLButtonElement>) {
    if (event.button !== 0) return;
    const rect = event.currentTarget.getBoundingClientRect();
    dragOffset.current = { x: event.clientX - rect.left, y: event.clientY - rect.top };
    dragStartedAt.current = { x: event.clientX, y: event.clientY };
    activePointerId.current = event.pointerId;
    moved.current = false;
    const nextDragPosition = { x: rect.left, y: rect.top };
    dragPositionRef.current = nextDragPosition;
    setDragPosition(nextDragPosition);
    event.preventDefault();

    function handleWindowPointerMove(moveEvent: PointerEvent) {
      if (activePointerId.current !== moveEvent.pointerId) return;
      moveEvent.preventDefault();
      updateDragPosition(moveEvent.clientX, moveEvent.clientY);
    }

    function handleWindowPointerUp(upEvent: PointerEvent) {
      if (activePointerId.current !== upEvent.pointerId) return;
      upEvent.preventDefault();
      finishDrag(upEvent.pointerId);
    }

    function handleWindowPointerCancel(cancelEvent: PointerEvent) {
      cancelDrag(cancelEvent.pointerId);
    }

    cleanupDragListeners.current?.();
    window.addEventListener("pointermove", handleWindowPointerMove, { passive: false });
    window.addEventListener("pointerup", handleWindowPointerUp, { passive: false });
    window.addEventListener("pointercancel", handleWindowPointerCancel);
    cleanupDragListeners.current = () => {
      window.removeEventListener("pointermove", handleWindowPointerMove);
      window.removeEventListener("pointerup", handleWindowPointerUp);
      window.removeEventListener("pointercancel", handleWindowPointerCancel);
    };

    event.currentTarget.setPointerCapture(event.pointerId);
  }

  function handleClick() {
    if (moved.current) {
      moved.current = false;
      return;
    }
    onOpen();
  }

  const style = dragPosition
    ? { left: `${dragPosition.x}px`, top: `${dragPosition.y}px` }
    : position.side === "left"
      ? { left: "12px", top: `${position.y}px` }
      : { right: "12px", top: `${position.y}px` };

  return (
    <button
      className={`popupHistoryBall ${position.side} ${dragPosition ? "dragging" : ""}`}
      type="button"
      aria-label={t("popupHistoryOpen")}
      title={t("popupHistoryOpen")}
      ref={ballRef}
      style={style}
      onClick={handleClick}
      onPointerDown={handlePointerDown}
      onPointerCancel={(event) => cancelDrag(event.pointerId)}
    >
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M5 6.5h14M5 12h9M5 17.5h6" />
        <path d="M18 15.5v4l3-2-3-2Z" />
      </svg>
      {count > 0 && <span className="popupHistoryBadge">{count > 99 ? "99+" : count}</span>}
    </button>
  );
}

function PopupHistoryModal({ items, lang, onClose, t }: { items: PopupHistoryItem[]; lang: Lang; onClose: () => void; t: Translate }) {
  return (
    <div className="modalOverlay" onClick={onClose}>
      <div className="modal popupHistoryModal" role="dialog" aria-modal="true" aria-label={t("popupHistory")} onClick={(e) => e.stopPropagation()}>
        <div className="modalHead">
          <div>
            <h2>{t("popupHistory")}</h2>
            <p>{t("popupHistoryCount", { count: items.length })}</p>
          </div>
          <button type="button" onClick={onClose}>
            <IconButtonContent icon="close">{t("close")}</IconButtonContent>
          </button>
        </div>
        <div className="modalBody">
          {items.length === 0 ? (
            <div className="popupHistoryEmpty">{t("popupHistoryEmpty")}</div>
          ) : (
            <ol className="popupHistoryList">
              {items.map((item) => (
                <li className={`popupHistoryItem ${item.kind}`} key={`${item.id}-${item.createdAt}`}>
                  <span className="popupHistoryStatus" aria-hidden="true" />
                  <div className="popupHistoryContent">
                    <div className="popupHistoryMeta">
                      <span>{item.kind === "ok" ? t("popupHistorySuccess") : t("popupHistoryError")}</span>
                      <time dateTime={item.createdAt}>{formatRepoTime(item.createdAt, lang, t)}</time>
                    </div>
                    <p>{item.message}</p>
                  </div>
                </li>
              ))}
            </ol>
          )}
        </div>
      </div>
    </div>
  );
}

function BulkProgressPanel({ progress, onClose, t }: { progress: BulkProgress; onClose: () => void; t: Translate }) {
  const done = progress.items.filter((item) => item.status === "done" || item.status === "error").length;
  const total = progress.items.length;
  const percent = total ? Math.round((done / total) * 100) : 0;
  const title = progress.kind === "check" ? t("bulkCheckTitle") : progress.kind === "sync" ? t("bulkSyncTitle") : t("bulkUsageTitle");
  const waitingSync = progress.items.filter((item) => item.needsSync).length;
  const finished = progress.completed || (total > 0 && done === total);
  const [remaining, setRemaining] = useState(3);
  const listRef = useRef<HTMLDivElement>(null);
  const previousItemsRef = useRef<BulkProgressItem[]>(progress.items);

  useEffect(() => {
    setRemaining(3);
  }, [progress.id, finished]);

  useEffect(() => {
    const previousItems = previousItemsRef.current;
    previousItemsRef.current = progress.items;
    const running = progress.items.find((item) => item.status === "running");
    const changed = progress.items.find((item, index) => {
      const previous = previousItems[index];
      return previous && (previous.status !== item.status || previous.message !== item.message || previous.needsSync !== item.needsSync);
    });
    const targetID = running?.id || changed?.id;
    if (!targetID) return;
    const target = listRef.current?.querySelector<HTMLElement>(`[data-progress-id="${CSS.escape(targetID)}"]`);
    target?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [progress.items]);

  useEffect(() => {
    if (!finished) return;
    const timer = window.setTimeout(() => {
      if (remaining <= 1) {
        onClose();
      } else {
        setRemaining((value) => value - 1);
      }
    }, 1000);
    return () => window.clearTimeout(timer);
  }, [finished, onClose, remaining]);

  return (
    <aside className="bulkProgressPanel" role="status" aria-live="polite">
      <div className="bulkProgressHead">
        <div>
          <strong>{title}</strong>
          <span>{t("progressCount", { done, total })}</span>
          {progress.kind === "check" && (
            <span className={waitingSync > 0 ? "bulkProgressSyncCount active" : "bulkProgressSyncCount"}>
              {t("progressWaitingSync", { count: waitingSync })}
            </span>
          )}
        </div>
        {finished && (
          <button type="button" onClick={onClose}>
            <IconButtonContent icon="close">{t("closeCountdown", { seconds: remaining })}</IconButtonContent>
          </button>
        )}
      </div>
      <div className="bulkProgressTrack" aria-hidden="true">
        <span style={{ width: `${percent}%` }} />
      </div>
      <div className="bulkProgressList" ref={listRef}>
        {progress.items.map((item) => (
          <div className={`bulkProgressItem ${item.status}`} key={item.id} data-progress-id={item.id}>
            <span className="bulkProgressDot" aria-hidden="true" />
            <span className="bulkProgressName">{item.label || item.id}</span>
            <span className="bulkProgressStatus">{progressStatusText(item.status, t)}</span>
            {item.message && <span className="bulkProgressMessage">{item.message}</span>}
          </div>
        ))}
      </div>
    </aside>
  );
}

function progressStatusText(status: BulkProgressStatus, t: Translate) {
  switch (status) {
    case "running":
      return t("progressRunning");
    case "done":
      return t("progressDone");
    case "error":
      return t("progressFailed");
    default:
      return t("progressPending");
  }
}

function LocalSkillsPage({
  t,
  run,
}: {
  t: Translate;
  run: (label: string, fn: () => Promise<unknown>) => Promise<void>;
}) {
  const agentOptions = useMemo(() => ["CLAUDE-Code", "codex"], []);
  const [activeAgent, setActiveAgent] = useState(agentOptions[0] || "CLAUDE-Code");
  const [localSkillFilter, setLocalSkillFilter] = useState<LocalSkillFilter>("all");
  const [localSkills, setLocalSkills] = useState<LocalSkill[]>([]);
  const [agentCounts, setAgentCounts] = useState<Record<string, number>>({});
  const [activeSkill, setActiveSkill] = useState<LocalSkill | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const filterOptions: { value: LocalSkillFilter; label: string }[] = [
    { value: "all", label: t("localSkillFilterAll") },
    { value: "symlink", label: t("localSkillFilterSymlink") },
    { value: "direct", label: t("localSkillFilterDirect") },
  ];
  const visibleLocalSkills = useMemo(
    () => {
      if (localSkillFilter === "symlink") return localSkills.filter((skill) => skill.is_symlink);
      if (localSkillFilter === "direct") return localSkills.filter((skill) => !skill.is_symlink);
      return localSkills;
    },
    [localSkillFilter, localSkills],
  );

  useEffect(() => {
    if (!agentOptions.includes(activeAgent)) {
      setActiveAgent(agentOptions[0] || "CLAUDE-Code");
    }
  }, [activeAgent, agentOptions]);

  useEffect(() => {
    let cancelled = false;
    void Promise.all(
      agentOptions.map(async (agent) => {
        const items = await api<LocalSkill[]>(`/api/local-skills?agent=${encodeURIComponent(agent)}`).catch(() => []);
        return [agent, items.length] as const;
      }),
    ).then((entries) => {
      if (!cancelled) setAgentCounts(Object.fromEntries(entries));
    });
    return () => {
      cancelled = true;
    };
  }, [agentOptions]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError("");
    void api<LocalSkill[]>(`/api/local-skills?agent=${encodeURIComponent(activeAgent)}`)
      .then((items) => {
        if (!cancelled) {
          const nextItems = items || [];
          setLocalSkills(nextItems);
          setAgentCounts((counts) => ({ ...counts, [activeAgent]: nextItems.length }));
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setLocalSkills([]);
          setError(err instanceof Error ? err.message : String(err));
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [activeAgent]);

  return (
    <section>
      <div className="sectionHead localSkillsHead">
        <div>
          <div className="sectionTitleRow">
            <h2>{t("localSkills")}</h2>
            <span className="sectionStats">{t("localSkillsStats", { count: visibleLocalSkills.length })}</span>
            {loading && (
              <span className="inlineLoadingSpinner" role="status" aria-label={t("loading")}>
                <span className="spinner" aria-hidden="true" />
              </span>
            )}
          </div>
          <p>{t("localSkillsDescription")}</p>
        </div>
      </div>
      <div className="localSkillsControls">
        <div className="localAgentSwitch" role="tablist" aria-label={t("agents")}>
          {agentOptions.map((agent) => (
            <button
              key={agent}
              type="button"
              role="tab"
              className={activeAgent === agent ? "active" : ""}
              aria-selected={activeAgent === agent}
              onClick={() => setActiveAgent(agent)}
            >
              <span>{agent}</span>
              <span className="localAgentCount">{agentCounts[agent] ?? "..."}</span>
            </button>
          ))}
        </div>
        <div className="localSkillFilterSwitch" role="tablist" aria-label={t("localSkills")}>
          {filterOptions.map((option) => (
            <button
              key={option.value}
              type="button"
              role="tab"
              className={localSkillFilter === option.value ? "active" : ""}
              aria-selected={localSkillFilter === option.value}
              onClick={() => setLocalSkillFilter(option.value)}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      {error && <p className="warning">{error}</p>}
      {!loading && !error && localSkills.length === 0 && <Empty title={t("localSkills")} body={t("localSkillsEmpty")} />}
      {!loading && !error && localSkills.length > 0 && visibleLocalSkills.length === 0 && (
        <Empty title={t("localSkills")} body={t("localSkillsFilterEmpty")} />
      )}
      {visibleLocalSkills.length > 0 && (
        <div className="localSkillGrid">
          {visibleLocalSkills.map((skill) => (
            <button key={skill.id} type="button" className={`localSkillCard card${skill.is_symlink ? " symlink" : ""}`} onClick={() => setActiveSkill(skill)}>
              <span className="localSkillCardTop">
                <strong>{skill.name}</strong>
                <span className="localSkillBadges">
                  {skill.is_symlink && <span className="symlinkBadge">{t("skillSymlink")}</span>}
                  <span className="agentBadge">{skill.agent}</span>
                </span>
              </span>
              {skill.description && <span className="localSkillDescription">{skill.description}</span>}
              <span className="localSkillMeta">
                <span>{t("localSkillPath")}</span>
                <span className="mono">{skill.root}</span>
              </span>
              {skill.is_symlink && skill.symlink_path && (
                <span className="localSkillMeta linked">
                  <span>{t("skillSymlinkPath")}</span>
                  <span className="mono">{skill.symlink_path}</span>
                </span>
              )}
              {skill.is_symlink && skill.real_path && (
                <span className="localSkillMeta linked">
                  <span>{t("realSkillPath")}</span>
                  <span className="mono">{skill.real_path}</span>
                </span>
              )}
              <span className="localSkillMeta compact">
                <span>{t("localSkillsRoot")}</span>
                <span className="mono">{skill.agent_root}</span>
              </span>
            </button>
          ))}
        </div>
      )}

      {activeSkill && <LocalSkillModal t={t} skill={activeSkill} run={run} onClose={() => setActiveSkill(null)} />}
    </section>
  );
}

function LocalSkillModal({
  t,
  skill,
  run,
  onClose,
}: {
  t: Translate;
  skill: LocalSkill;
  run: (label: string, fn: () => Promise<unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const [content, setContent] = useState<string | null>(null);
  const [tree, setTree] = useState<SkillTree | null>(null);
  const [error, setError] = useState("");
  const [treeError, setTreeError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setContent(null);
    setTree(null);
    setError("");
    setTreeError("");
    void api<{ content: string }>(`/api/local-skills/${encodeURIComponent(skill.id)}/content`)
      .then((data) => {
        if (!cancelled) setContent(data.content || "");
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      });
    void api<SkillTree>(`/api/local-skills/${encodeURIComponent(skill.id)}/tree`)
      .then((data) => {
        if (!cancelled) setTree(data);
      })
      .catch((err) => {
        if (!cancelled) setTreeError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      cancelled = true;
    };
  }, [skill.id]);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const { meta, frontmatter, body } = useMemo(
    () => (content == null ? { meta: null, frontmatter: null, body: "" } : parseFrontmatter(content)),
    [content],
  );

  const html = useMemo(() => {
    if (!body) return "";
    const raw = marked.parse(body, { async: false }) as string;
    return DOMPurify.sanitize(raw);
  }, [body]);

  function openSkillDir(path: string) {
    run("Open directory", () => api(`/api/local-skills/${encodeURIComponent(skill.id)}/open-dir`, { method: "POST", body: JSON.stringify({ path }) }));
  }

  function openSkillPath(path: string) {
    run("Open item", () => api(`/api/local-skills/${encodeURIComponent(skill.id)}/open-path`, { method: "POST", body: JSON.stringify({ path }) }));
  }

  return (
    <div className="modalOverlay" onClick={onClose}>
      <div className="modal localSkillModal" role="dialog" aria-modal="true" aria-label={skill.name} onClick={(e) => e.stopPropagation()}>
        <div className="modalHead">
          <div>
            <h2>{skill.name}</h2>
            <p className="mono small">{skill.root}</p>
          </div>
          <button onClick={onClose}><IconButtonContent icon="close">{t("close")}</IconButtonContent></button>
        </div>
        <div className="modalBody">
          <div className="localSkillSummary">
            <span className="agentBadge">{skill.agent}</span>
            {skill.is_symlink && <span className="symlinkBadge">{t("skillSymlink")}</span>}
            <span className="mono">{skill.relative_path}</span>
            {skill.content_sha && <span className="mono">{shortSha(skill.content_sha)}</span>}
          </div>
          {skill.is_symlink && (
            <div className="localSkillLinkPanel">
              {skill.symlink_path && (
                <span className="localSkillMeta linked">
                  <span>{t("skillSymlinkPath")}</span>
                  <span className="mono">{skill.symlink_path}</span>
                </span>
              )}
              {skill.real_path && (
                <span className="localSkillMeta linked">
                  <span>{t("realSkillPath")}</span>
                  <span className="mono">{skill.real_path}</span>
                </span>
              )}
            </div>
          )}
          {error && <p className="warning">{error}</p>}
          {content == null && !error && <p className="muted">{t("loading")}</p>}
          <SkillTreeView t={t} tree={tree} error={treeError} onOpenDir={openSkillDir} onOpenPath={openSkillPath} />
          {frontmatter && (meta ? <FrontmatterTable meta={meta} /> : <FrontmatterRaw value={frontmatter} />)}
          {content != null && body && <h3 className="contentSectionTitle">{t("skillMarkdown")}</h3>}
          {content != null && body && <div className="markdown" dangerouslySetInnerHTML={{ __html: html }} />}
        </div>
      </div>
    </div>
  );
}

function UsageRankingPage({
  t,
  lang,
  skills,
  agents,
  run,
  refreshedRankings,
  onUsageLoaded,
}: {
  t: Translate;
  lang: Lang;
  skills: Skill[];
  agents: Record<string, AgentConfig>;
  run: (label: string, fn: () => Promise<unknown>) => Promise<void>;
  refreshedRankings: Partial<Record<UsageRange, SkillUsageRanking>>;
  onUsageLoaded: (ranking: SkillUsageRanking) => void;
}) {
  const [range, setRange] = useState<UsageRange>("week");
  const [ranking, setRanking] = useState<SkillUsageRanking | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingSnapshot, setLoadingSnapshot] = useState(true);
  const [error, setError] = useState("");
  const [activeSkillId, setActiveSkillId] = useState<string | null>(null);
  const rangeOptions: { value: UsageRange; label: string }[] = [
    { value: "day", label: t("usageDay") },
    { value: "week", label: t("usageWeek") },
    { value: "month", label: t("usageMonth") },
    { value: "all", label: t("usageAll") },
  ];
  const activeSkill = activeSkillId ? skills.find((skill) => skill.id === activeSkillId) || null : null;
  const refreshedRanking = refreshedRankings[range];

  useEffect(() => {
    let cancelled = false;
    setLoadingSnapshot(true);
    setError("");
    void api<SkillUsageSnapshot>(`/api/usage/ranking-snapshot?range=${encodeURIComponent(range)}`)
      .then((snapshot) => {
        if (cancelled) return;
        const cached = snapshot.available ? snapshot.ranking : null;
        setRanking(cached);
        if (cached) onUsageLoaded(cached);
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (!cancelled) setLoadingSnapshot(false);
      });
    return () => {
      cancelled = true;
    };
  }, [range]);

  useEffect(() => {
    if (!refreshedRanking) return;
    setRanking(refreshedRanking);
    setError("");
    setLoadingSnapshot(false);
  }, [refreshedRanking]);

  async function refreshUsage() {
    setRefreshing(true);
    setError("");
    try {
      const data = await api<SkillUsageRanking>(`/api/usage/ranking?range=${encodeURIComponent(range)}`);
      setRanking(data);
      onUsageLoaded(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setRefreshing(false);
    }
  }

  return (
    <section>
      <div className="sectionHead usageHead">
        <div>
          <div className="sectionTitleRow">
            <h2>{t("usageRanking")}</h2>
            {ranking && <span className="sectionStats">{t("usageTotal", { count: ranking.items.reduce((sum, item) => sum + item.counts.total, 0) })}</span>}
            {ranking && (
              <time className="usageUpdatedAt" dateTime={ranking.generated_at}>
                {t("usageLastUpdated", { time: formatUsageTime(ranking.generated_at, lang) })}
              </time>
            )}
          </div>
          <p>{t("observedUsageDescription")}</p>
        </div>
        <div className="usageControls">
          <div className="usageRangeSwitch" role="tablist" aria-label={t("usageRanking")}>
            {rangeOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                role="tab"
                className={range === option.value ? "active" : ""}
                aria-selected={range === option.value}
                onClick={() => {
                  setRange(option.value);
                }}
              >
                {option.label}
              </button>
            ))}
          </div>
          <button
            type="button"
            className="usageRefreshButton"
            aria-label={t("refreshUsage")}
            title={t("refreshUsage")}
            disabled={refreshing}
            onClick={() => void refreshUsage()}
          >
            {refreshing ? <Spinner /> : <ButtonIcon name="sync" />}
          </button>
        </div>
      </div>

      {error && <p className="warning">{error}</p>}
      {!ranking && !error && <p className="muted">{refreshing || loadingSnapshot ? t("loading") : t("usageRefreshHint")}</p>}
      {ranking && ranking.items.length === 0 && <Empty title={t("usageRanking")} body={t("noUsageRanking")} />}
      {ranking && ranking.items.length > 0 && (
        <div className="usageRankingList">
          {ranking.items.map((item, index) => {
            const skill = skills.find((candidate) => candidate.id === item.skill_id);
            return (
              <button
                key={item.skill_id}
                type="button"
                className="usageRankingItem card"
                onClick={() => {
                  if (skill) setActiveSkillId(skill.id);
                }}
                disabled={!skill}
              >
                <span className="usageRank">#{index + 1}</span>
                <span className="usageRankingMain">
                  <strong>{item.name}</strong>
                  <span className="usageRankingMeta">
                    <span>{t("usageRepo")}</span>
                    <span className="mono">{item.source_id}</span>
                    <span className="repoInlineDot" aria-hidden="true">·</span>
                    <span className="mono">{item.relative_path}</span>
                  </span>
                </span>
                <span className="usageRankingCounts">
                  <SkillUsageMarkers usage={item} t={t} />
                  <span className="usageTotalPill">{t("usageTotal", { count: item.counts.total })}</span>
                </span>
              </button>
            );
          })}
        </div>
      )}

      {activeSkill && <SkillModal t={t} skill={activeSkill} agents={agents} run={run} onClose={() => setActiveSkillId(null)} />}
    </section>
  );
}

function SourcesPage({
  t,
  lang,
  sources,
  skills,
  usageSummary,
  agents,
  projects,
  run,
  runAllSources,
  renameSource,
  deleteSource,
  focusedSourceId,
  onSourceFocused,
  busyLabel,
  bulkRunning,
}: {
  t: Translate;
  lang: Lang;
  sources: Source[];
  skills: Skill[];
  usageSummary: SkillUsageSummary | null;
  agents: Record<string, AgentConfig>;
  projects: ProjectRef[];
  run: (label: string, fn: () => Promise<unknown>) => Promise<void>;
  runAllSources: (kind: SourceBulkProgressKind) => Promise<void>;
  renameSource: (oldId: string, newId: string) => Promise<RenameSourceResult>;
  deleteSource: (id: string) => Promise<void>;
  focusedSourceId: string | null;
  onSourceFocused: () => void;
  busyLabel: string | null;
  bulkRunning: boolean;
}) {
  const [form, setForm] = useState({ id: "", url: "", branch: "" });
  const [localNameEdited, setLocalNameEdited] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [activeSkillId, setActiveSkillId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [selectedTags, setSelectedTags] = useState<Set<string> | null>(null);
  const [tagInput, setTagInput] = useState("");
  const [enable, setEnable] = useState<{ agents: string[]; scope: string; projectRoot: string }>({ agents: [], scope: "user", projectRoot: "" });
  const [pickerOpen, setPickerOpen] = useState(false);
  const [pinnedBySource, setPinnedBySource] = useState<Record<string, string[]>>(() => initialPinnedSkills());
  const [renamingSourceId, setRenamingSourceId] = useState<string | null>(null);
  const [renameDraft, setRenameDraft] = useState("");
  const [deletingSource, setDeletingSource] = useState<Source | null>(null);
  const renameCancelRef = useRef(false);
  const inferredLocalName = cleanLocalRepoName(repoNameFromURL(form.url));
  const availableAgents = useMemo(() => agentNames(agents), [agents]);

  useEffect(() => {
    if (!localNameEdited) setForm((current) => ({ ...current, id: inferredLocalName }));
  }, [inferredLocalName, localNameEdited]);

  function toggleEnableAgent(agent: string) {
    setEnable((prev) => ({
      ...prev,
      agents: prev.agents.includes(agent) ? prev.agents.filter((a) => a !== agent) : [...prev.agents, agent],
    }));
  }

  const allTags = useMemo(() => unique(skills.flatMap((s) => s.tags || [])), [skills]);
  const repoTagSet = useMemo(() => new Set(sources.map((source) => source.id)), [sources]);
  const customTags = useMemo(() => allTags.filter((tag) => !repoTagSet.has(tag)), [allTags, repoTagSet]);
  const repoTags = useMemo(() => allTags.filter((tag) => repoTagSet.has(tag)), [allTags, repoTagSet]);
  const effectiveSelectedTags = useMemo(() => selectedTags ?? new Set(allTags), [selectedTags, allTags]);
  const hasNoTagsSelected = allTags.length > 0 && effectiveSelectedTags.size === 0;

  useEffect(() => {
    setSelectedTags((current) => {
      if (current === null) return null;
      const all = new Set(allTags);
      return new Set(Array.from(current).filter((tag) => all.has(tag)));
    });
  }, [allTags]);

  useEffect(() => {
    if (!focusedSourceId) return;
    setSelectedTags(null);
    setExpanded((current) => {
      const next = new Set(current);
      next.add(focusedSourceId);
      return next;
    });
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        const target = document.querySelector<HTMLElement>(`[data-source-id="${CSS.escape(focusedSourceId)}"]`);
        target?.scrollIntoView({ block: "center", behavior: "smooth" });
        target?.classList.add("sourceArticleFocusPulse");
        window.setTimeout(() => target?.classList.remove("sourceArticleFocusPulse"), 1200);
        onSourceFocused();
      });
    });
  }, [focusedSourceId, onSourceFocused]);

  const skillsBySource = useMemo(() => {
    const map: Record<string, Skill[]> = {};
    for (const skill of skills) {
      if (hasNoTagsSelected) continue;
      if (allTags.length > 0 && !(skill.tags || []).some((tag) => effectiveSelectedTags.has(tag))) continue;
      (map[skill.source_id] ||= []).push(skill);
    }
    for (const [sourceId, list] of Object.entries(map)) {
      const pins = pinnedBySource[sourceId] || [];
      const pinRank = new Map(pins.map((id, index) => [id, index]));
      list.sort((a, b) => {
        const aPinned = pinRank.has(a.id);
        const bPinned = pinRank.has(b.id);
        if (aPinned !== bPinned) return aPinned ? -1 : 1;
        if (aPinned && bPinned) return pinRank.get(a.id)! - pinRank.get(b.id)!;
        return a.name.localeCompare(b.name);
      });
    }
    return map;
  }, [skills, allTags.length, effectiveSelectedTags, hasNoTagsSelected, pinnedBySource]);
  const allSkillsBySource = useMemo(() => {
    const map: Record<string, Skill[]> = {};
    for (const skill of skills) {
      (map[skill.source_id] ||= []).push(skill);
    }
    return map;
  }, [skills]);
  const visibleSources = useMemo(() => {
    if (allTags.length === 0) return sources;
    return sources.filter((source) => (skillsBySource[source.id] || []).length > 0);
  }, [sources, skillsBySource, allTags.length]);
  const visibleStats = useMemo(() => {
    const visibleSkills = visibleSources.flatMap((source) => skillsBySource[source.id] || []);
    return skillActivationSummary(visibleSkills);
  }, [visibleSources, skillsBySource]);

  const activeSkill = activeSkillId ? skills.find((s) => s.id === activeSkillId) || null : null;
  const selectionActive = selected.size > 0;
  const selectedIDs = Array.from(selected);
  const selectedActivations = skills
    .filter((skill) => selected.has(skill.id))
    .flatMap((skill) => skill.activations || []);

  function toggleExpand(id: string) {
    const next = new Set(expanded);
    next.has(id) ? next.delete(id) : next.add(id);
    setExpanded(next);
  }

  function toggleSelect(id: string) {
    const next = new Set(selected);
    next.has(id) ? next.delete(id) : next.add(id);
    setSelected(next);
  }

  function selectSource(ids: string[], on: boolean) {
    const next = new Set(selected);
    for (const id of ids) {
      on ? next.add(id) : next.delete(id);
    }
    setSelected(next);
  }

  function toggleTagFilter(tag: string) {
    setSelectedTags((current) => {
      const next = new Set(current ?? allTags);
      next.has(tag) ? next.delete(tag) : next.add(tag);
      return next;
    });
  }

  function renderTagChip(tag: string) {
    const selected = effectiveSelectedTags.has(tag);
    return (
      <button
        key={tag}
        type="button"
        className={`tagFilterChip${selected ? " selected" : ""}`}
        aria-pressed={selected}
        onClick={() => toggleTagFilter(tag)}
      >
        {tag}
      </button>
    );
  }

  function togglePinnedSkill(sourceId: string, skillId: string) {
    setPinnedBySource((current) => {
      const existing = current[sourceId] || [];
      const nextForSource = existing.includes(skillId) ? existing.filter((id) => id !== skillId) : [...existing, skillId];
      const next = { ...current };
      if (nextForSource.length) {
        next[sourceId] = nextForSource;
      } else {
        delete next[sourceId];
      }
      localStorage.setItem("skillctl-pinned-skills", JSON.stringify(next));
      return next;
    });
  }

  function toggleSourcePin(source: Source) {
    const pinned = !source.pinned;
    run(pinned ? `Pin ${source.id}` : `Unpin ${source.id}`, () =>
      api(`/api/sources/${encodeURIComponent(source.id)}/pin`, {
        method: "POST",
        body: JSON.stringify({ pinned }),
      }),
    );
  }

  function startRename(sourceId: string) {
    renameCancelRef.current = false;
    setRenamingSourceId(sourceId);
    setRenameDraft(sourceId);
  }

  function cancelRename() {
    setRenamingSourceId(null);
    setRenameDraft("");
  }

  async function commitRename(oldId: string) {
    const nextId = cleanLocalRepoName(renameDraft);
    if (!nextId || nextId === oldId) {
      cancelRename();
      return;
    }
    const oldSkills = skills.filter((skill) => skill.source_id === oldId);
    try {
      const result = await renameSource(oldId, nextId);
      const returnedSkills = result.skills || [];
      const idByOldId = new Map<string, string>();
      for (const oldSkill of oldSkills) {
        const renamedSkill = returnedSkills.find((skill) => skill.relative_path === oldSkill.relative_path);
        if (renamedSkill) idByOldId.set(oldSkill.id, renamedSkill.id);
      }
      setExpanded((current) => {
        const next = new Set(current);
        if (next.delete(oldId)) next.add(result.source.id);
        return next;
      });
      setSelected((current) => new Set(Array.from(current, (id) => idByOldId.get(id) || id)));
      setActiveSkillId((current) => current ? idByOldId.get(current) || current : current);
      setSelectedTags((current) => {
        if (current === null || !current.has(oldId)) return current;
        const next = new Set(current);
        next.delete(oldId);
        next.add(result.source.id);
        return next;
      });
      setPinnedBySource((current) => {
        const oldPins = current[oldId] || [];
        const next = { ...current };
        delete next[oldId];
        if (oldPins.length) {
          next[result.source.id] = Array.from(new Set([...oldPins.map((id) => idByOldId.get(id) || id), ...(next[result.source.id] || [])]));
        }
        localStorage.setItem("skillctl-pinned-skills", JSON.stringify(next));
        return next;
      });
      cancelRename();
    } catch {
      // The shared runner already shows the error toast. Keep the field open so the value can be corrected.
    }
  }

  async function removeSource(source: Source) {
    try {
      await deleteSource(source.id);
      setExpanded((current) => {
        const next = new Set(current);
        next.delete(source.id);
        return next;
      });
      setSelected((current) => new Set(Array.from(current).filter((id) => !id.startsWith(`${source.id}::`))));
      setPinnedBySource((current) => {
        if (!current[source.id]) return current;
        const next = { ...current };
        delete next[source.id];
        localStorage.setItem("skillctl-pinned-skills", JSON.stringify(next));
        return next;
      });
      setDeletingSource(null);
    } catch {
      // The delete helper already reports the error.
    }
  }

  return (
    <section>
      <div className="sectionHead">
        <div>
          <div className="sectionTitleRow">
            <h2>{t("sources")}</h2>
            <span className="sectionStats">
              {t("sourcesInventoryStats", { repos: visibleSources.length, skills: visibleStats.skills })}
            </span>
            <span className="sectionStats">
              {t("sourcesEnabledStats", { global: visibleStats.global, project: visibleStats.project })}
            </span>
          </div>
          <p>{t("sourcesDescription")}</p>
        </div>
        <div className="actions sourceBulkActions">
          <button disabled={bulkRunning || busyLabel != null} onClick={() => runAllSources("check")}>
            {busyLabel === "Check all sources" && <Spinner />}
            <IconButtonContent icon="check">{t("checkAll")}</IconButtonContent>
          </button>
          <button className="primary" disabled={bulkRunning || busyLabel != null} onClick={() => runAllSources("sync")}>
            {busyLabel === "Sync all sources" && <Spinner />}
            <IconButtonContent icon="sync">{t("syncAll")}</IconButtonContent>
          </button>
        </div>
      </div>

      <div className={`sourceSelectionZone${selectionActive ? " withSelection" : ""}`}>
        <div className="sourceSelectionMain">
          <form
            className="card formGrid sourceImportPanel"
            onSubmit={(event) => {
              event.preventDefault();
              const id = cleanLocalRepoName(form.id || repoNameFromURL(form.url));
              run("Add source", () => api("/api/sources", { method: "POST", body: JSON.stringify({ ...form, id }) }));
            }}
          >
            <label>
              {t("gitUrl")}
              <input value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://github.com/org/repo.git" required />
            </label>
            <label>
              {t("branch")}
              <input value={form.branch} onChange={(e) => setForm({ ...form, branch: e.target.value })} placeholder="main" />
            </label>
            <label>
              {t("sourceId")}
              <input
                value={form.id}
                onChange={(e) => {
                  setLocalNameEdited(e.target.value.trim() !== "");
                  setForm({ ...form, id: e.target.value });
                }}
                placeholder="my-repo"
              />
            </label>
            <button className="primary" type="submit" disabled={busyLabel === "Add source"}>
              {busyLabel === "Add source" && <Spinner />}
              <IconButtonContent icon="repoAdd">{t("addSource")}</IconButtonContent>
            </button>
          </form>

          <div className="filterBar card tagFilterPanel">
            <div className="inlineField">
              <span className="fieldLabel" aria-label={t("filterByTag")}>
                {lang === "zh" ? (
                  <>
                    <span>筛选</span>
                    <span>标签</span>
                  </>
                ) : (
                  <>
                    <span>FILTER</span>
                    <span>BY TAG</span>
                  </>
                )}
              </span>
              <div className="tagFilterList" aria-label={t("filterByTag")}>
                {allTags.length === 0 && <span className="muted small">{t("noTags")}</span>}
                {customTags.length > 0 && (
                  <div className="tagFilterGroup">
                    <span className="tagFilterGroupLabel">{t("customTags")}</span>
                    <div className="tagFilterChips">{customTags.map(renderTagChip)}</div>
                  </div>
                )}
                {customTags.length > 0 && repoTags.length > 0 && <span className="tagFilterDivider" aria-hidden="true" />}
                {repoTags.length > 0 && (
                  <div className="tagFilterGroup">
                    <span className="tagFilterGroupLabel">{t("repoTags")}</span>
                    <div className="tagFilterChips">{repoTags.map(renderTagChip)}</div>
                  </div>
                )}
              </div>
            </div>
          </div>

          {hasNoTagsSelected && (
            <div className="filterEmptyState card" role="status">
              {t("noSkillsSelected")}
            </div>
          )}
        </div>

        {selected.size > 0 && (
          <aside className="bulk selectionDrawer card" aria-live="polite">
            <strong>{t("selectedCount", { count: selected.size })}</strong>
            <div className="bulkTagRow">
              <input placeholder={t("tagPlaceholder")} value={tagInput} onChange={(e) => setTagInput(e.target.value)} />
              <button disabled={!splitTags(tagInput).length} onClick={() => run("Add tags", () => api("/api/skills/tags", { method: "POST", body: JSON.stringify({ skill_ids: selectedIDs, tags: splitTags(tagInput), action: "add" }) }))}>
                <IconButtonContent icon="tagAdd">{t("addTags")}</IconButtonContent>
              </button>
            </div>
            <div className="agentPick">
              <span className="muted small">{t("agents")}</span>
              {availableAgents.length === 0 && <span className="muted small">{t("noneConfigured")}</span>}
              {availableAgents.map((agent) => (
                <label key={agent} className={`agentChip${enable.agents.includes(agent) ? " on" : ""}`}>
                  <input type="checkbox" checked={enable.agents.includes(agent)} onChange={() => toggleEnableAgent(agent)} />
                  {activationAgentLabel(agent)}
                </label>
              ))}
            </div>
            <select className="scopeSelect" aria-label={t("activationScope")} value={enable.scope} onChange={(e) => setEnable({ ...enable, scope: e.target.value })}>
              <option value="user">{t("global")}</option>
              <option value="project">{t("project")}</option>
            </select>
            {enable.scope === "project" && (
              <span className="projectField bulkProjectField">
                {projects.length > 0 && (
                  <select className="projectSelect" aria-label={t("savedProject")} value="" onChange={(e) => { if (e.target.value) setEnable((prev) => ({ ...prev, projectRoot: e.target.value })); }}>
                    <option value="">{t("savedProject")}</option>
                    {projects.map((p) => <option key={p.alias} value={p.path}>{p.alias}</option>)}
                  </select>
                )}
                <input placeholder={t("projectPathPlaceholder")} value={enable.projectRoot} onChange={(e) => setEnable({ ...enable, projectRoot: e.target.value })} />
                <button type="button" onClick={() => setPickerOpen(true)}>
                  <IconButtonContent icon="browse">{t("browse")}</IconButtonContent>
                </button>
              </span>
            )}
            <div className="selectionActions">
              <button className="primary" disabled={!enable.agents.length || (enable.scope === "project" && !enable.projectRoot)} onClick={() => run(`Enable for ${enable.agents.join(", ")}`, async () => {
                for (const agent of enable.agents) {
                  await api("/api/activations", { method: "POST", body: JSON.stringify({ skill_ids: selectedIDs, agent, scope: enable.scope, project_root: enable.projectRoot }) });
                }
              })}><IconButtonContent icon="enable">{t("enable")}</IconButtonContent></button>
              <button className="danger" disabled={!selectedActivations.length} onClick={() => run("Close links", async () => {
                for (const activation of selectedActivations) {
                  await api(`/api/activations/${activation.id}`, { method: "DELETE" });
                }
              })}><IconButtonContent icon="unlink">{t("closeLinks")}</IconButtonContent></button>
              <button className="ghost" onClick={() => setSelected(new Set())}><IconButtonContent icon="clear">{t("clear")}</IconButtonContent></button>
            </div>
          </aside>
        )}
      </div>

      <div className="stack sourceList">
        {visibleSources.map((source) => {
          const isOpen = expanded.has(source.id);
          const sourceSkills = skillsBySource[source.id] || [];
          const sourceIDs = sourceSkills.map((s) => s.id);
          const selectedCount = sourceIDs.filter((id) => selected.has(id)).length;
          const allSelected = sourceIDs.length > 0 && selectedCount === sourceIDs.length;
          const someSelected = selectedCount > 0;
          const dimmed = selectionActive && !someSelected;
          const sourceEnabledAgents = sourceFullyEnabledAgents(allSkillsBySource[source.id] || []);
          const updatedAt = source.last_commit_at || source.last_fetch_at;
          const updatedText = formatRepoTime(updatedAt, lang, t);
          return (
            <article className={`card sourceArticle${source.pinned ? " pinned" : ""}${dimmed ? " dimmed" : ""}${someSelected ? " picked" : ""}`} key={source.id} data-source-id={source.id}>
              <div className="sourceCard">
                <input
                  type="checkbox"
                  className="repoCheck"
                  aria-label={t("selectAllSkills", { name: source.id })}
                  checked={allSelected}
                  disabled={sourceIDs.length === 0}
                  ref={(el) => { if (el) el.indeterminate = someSelected && !allSelected; }}
                  onChange={() => selectSource(sourceIDs, !allSelected)}
                />
                <div className="sourceCardMain">
                  <div
                    className="row sourceCardToggle"
                    role="button"
                    tabIndex={0}
                    aria-expanded={isOpen}
                    aria-label={t(isOpen ? "collapseDirectory" : "expandDirectory", { name: source.id })}
                    onClick={() => toggleExpand(source.id)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        toggleExpand(source.id);
                      }
                    }}
                  >
                    <span className="sourceHeaderTitle">
                      <span className={`caret ${isOpen ? "open" : ""}`} aria-hidden="true">▶</span>
                      {renamingSourceId === source.id ? (
                        <input
                          className="sourceNameInput"
                          value={renameDraft}
                          autoFocus
                          aria-label={t("renameSource")}
                          onClick={(event) => event.stopPropagation()}
                          onFocus={(event) => event.currentTarget.select()}
                          onChange={(event) => setRenameDraft(event.target.value)}
                          onBlur={() => {
                            if (renameCancelRef.current) {
                              renameCancelRef.current = false;
                              return;
                            }
                            commitRename(source.id);
                          }}
                          onKeyDown={(event) => {
                            event.stopPropagation();
                            if (event.key === "Enter") {
                              event.preventDefault();
                              event.currentTarget.blur();
                            }
                            if (event.key === "Escape") {
                              event.preventDefault();
                              renameCancelRef.current = true;
                              cancelRename();
                            }
                          }}
                        />
                      ) : (
                        <>
                          <h3>{source.id}</h3>
                          <button
                            type="button"
                            className={`sourcePinButton${source.pinned ? " pinned" : ""}`}
                            aria-label={t(source.pinned ? "unpinSource" : "pinSource", { name: source.id })}
                            aria-pressed={Boolean(source.pinned)}
                            title={t(source.pinned ? "unpinSource" : "pinSource", { name: source.id })}
                            onClick={(event) => {
                              event.stopPropagation();
                              toggleSourcePin(source);
                            }}
                          >
                            <ButtonIcon name="pin" />
                          </button>
                          <button
                            type="button"
                            className="sourceRenameButton"
                            aria-label={t("renameSource")}
                            title={t("renameSource")}
                            onClick={(event) => {
                              event.stopPropagation();
                              startRename(source.id);
                            }}
                          >
                            <ButtonIcon name="edit" />
                          </button>
                          <SourceActivationMarkers agents={sourceEnabledAgents} t={t} />
                        </>
                      )}
                    </span>
                    <span className="sourceHeaderMeta">
                      <span className="sourceSkillCount">{t("sourceSkillCount", { count: source.skill_count })}</span>
                      <Status value={source.status} t={t} />
                      {(source.remote_sha || source.local_sha || source.local_branch) && (
                        <span className="repoCommitPills">
                          {!source.local_source && source.remote_sha && (
                            <span className="commitPill" title={source.remote_sha}>{t("remoteCommit")} {shortSha(source.remote_sha)}</span>
                          )}
                          {source.local_sha && (
                            <span className="commitPill" title={source.local_sha}>{t("localCommit")} {shortSha(source.local_sha)}</span>
                          )}
                          {source.local_source && source.local_branch && (
                            <span className="commitPill" title={source.local_branch}>{t("branch")} {source.local_branch}</span>
                          )}
                        </span>
                      )}
                    </span>
                  </div>
                  <div className="repoDetails">
                    {!source.local_source && sourceRemoteRows(source).map((remote, index) => (
                      <div className="repoLine remoteRepoLine" key={`${remote.name}-${remote.branch || index}`}>
                        <span className="repoLineLabel iconOnly" aria-label={t("remoteRepo")} title={t("remoteRepo")}>
                          <span className="githubMark" aria-label="GitHub">
                            <svg viewBox="0 0 16 16" aria-hidden="true">
                              <path fill="currentColor" d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82A7.65 7.65 0 0 1 8 3.86c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
                            </svg>
                          </span>
                        </span>
                        <span className="remoteNameChip">{remote.name}</span>
                        <a className="mono repoPath repoLink" href={githubWebURL(remote.url)} target="_blank" rel="noreferrer" title={remote.url}>
                          <MiddleEllipsisText value={remote.url} />
                        </a>
                        <span className="repoInlineDot" aria-hidden="true">·</span>
                        <span className="repoInlineMeta">{sourceRemoteBranchMeta(remote, source.branch, t)}</span>
                        {remote.sha && <span className="commitPill remoteCommitPill" title={remote.sha}>{shortSha(remote.sha)}</span>}
                      </div>
                    ))}
                    <div className="repoLine">
                      <span className="repoLineLabel iconOnly" aria-label={t("localRepo")} title={t("localRepo")}>
                        <span className="localRepoMark" aria-label="Local repository">
                          <svg viewBox="0 0 16 16" aria-hidden="true">
                            <circle cx="8" cy="8" r="7.15" fill="none" stroke="currentColor" strokeWidth="1.45" />
                            <path fill="currentColor" d="M4.2 5.25c0-.55.45-1 1-1h5.6c.55 0 1 .45 1 1v3.55c0 .55-.45 1-1 1H8.7l.18.85h1.36a.6.6 0 0 1 0 1.2H5.76a.6.6 0 0 1 0-1.2h1.36l.18-.85H5.2c-.55 0-1-.45-1-1V5.25Zm1.2.2V8.6h5.2V5.45H5.4Z" />
                          </svg>
                        </span>
                      </span>
                      <button
                        type="button"
                        className="mono repoPath repoLink repoPathButton"
                        title={source.local_path || source.checkout_path}
                        onClick={() => run(`Open ${source.id}`, () => api(`/api/sources/${encodeURIComponent(source.id)}/open-dir`, { method: "POST" }))}
                      >
                        <MiddleEllipsisText value={source.local_path || source.checkout_path} />
                      </button>
                    </div>
                  </div>
                  {source.message && <p className="warning">{source.message}</p>}
                  <SourceNote source={source} run={run} busyLabel={busyLabel} t={t} />
                </div>
                <div className="sourceActions">
                  {source.local_source ? (
                    <div className="actions sourceActionButtons">
                      <button className="primary" disabled={bulkRunning || busyLabel === `Rescan ${source.id}`} onClick={(e) => { e.stopPropagation(); run(`Rescan ${source.id}`, () => api(`/api/sources/${encodeURIComponent(source.id)}/sync`, { method: "POST" })); }}>
                        {busyLabel === `Rescan ${source.id}` && <Spinner />}
                        <IconButtonContent icon="sync">{t("rescanIndex")}</IconButtonContent>
                      </button>
                    </div>
                  ) : <div className="actions sourceActionButtons">
                    <button disabled={bulkRunning || busyLabel === `Check ${source.id}`} onClick={(e) => { e.stopPropagation(); run(`Check ${source.id}`, () => api(`/api/sources/${encodeURIComponent(source.id)}/check`, { method: "POST" })); }}>
                      {busyLabel === `Check ${source.id}` && <Spinner />}
                      <IconButtonContent icon="check">{t("check")}</IconButtonContent>
                    </button>
                    <button className="primary" disabled={bulkRunning || busyLabel === `Sync ${source.id}`} onClick={(e) => { e.stopPropagation(); run(`Sync ${source.id}`, () => api(`/api/sources/${encodeURIComponent(source.id)}/sync`, { method: "POST" })); }}>
                      {busyLabel === `Sync ${source.id}` && <Spinner />}
                      <IconButtonContent icon="sync">{t("sync")}</IconButtonContent>
                    </button>
                  </div>}
                  {updatedText && (
                    <span className="repoUpdatedPill sourceUpdatedPill" title={updatedAt}>
                      {t("lastUpdated", { time: updatedText })}
                    </span>
                  )}
                  <button
                    type="button"
                    className="sourceDeleteButton danger"
                    aria-label={t("deleteSource", { name: source.id })}
                    title={t("deleteSource", { name: source.id })}
                    disabled={bulkRunning || busyLabel != null}
                    onClick={() => setDeletingSource(source)}
                  >
                    <ButtonIcon name="remove" />
                  </button>
                </div>
              </div>

              {isOpen && (
                <div className="skillList">
                  {sourceSkills.map((skill) => {
                    const isSelected = selected.has(skill.id);
                    const enabled = (skill.activations || []).length > 0;
                    const pinned = (pinnedBySource[source.id] || []).includes(skill.id);
                    const changeClass = skill.local_changed ? " localChanged" : skill.remote_changed ? " remoteChanged" : "";
                    return (
                      <div className={`skillCell ${isSelected ? "selected" : ""}${pinned ? " pinned" : ""}${changeClass}`} key={skill.id}>
                        <input
                          type="checkbox"
                          className="skillCheck"
                          aria-label={t("selectSkill", { name: skill.name })}
                          checked={isSelected}
                          onChange={() => toggleSelect(skill.id)}
                        />
                        <div className="skillRow">
                          <button className="skillRowOpen" type="button" onClick={() => setActiveSkillId(skill.id)}>
                            <strong className="skillRowName">
                              <span className="skillRowTitleText">{skill.name}</span>
                              {enabled && <SkillActivationMarkers activations={skill.activations || []} t={t} />}
                              <SkillUsageMarkers usage={usageSummary?.counts?.[skill.id]} t={t} />
                            </strong>
                            {skill.description && (
                              <span className="skillRowDescription" title={skill.description}>{skill.description}</span>
                            )}
                          </button>
                          <SkillNote skill={skill} run={run} busyLabel={busyLabel} t={t} />
                          {Boolean(skill.tags?.length) && (
                            <span className="skillRowTags" aria-label={t("tags")}>
                              {skill.tags!.map((tag) => (
                                <span className="tag skillRowTag" key={tag}>{tag}</span>
                              ))}
                            </span>
                          )}
                        </div>
                        <button
                          type="button"
                          className={`skillPin${pinned ? " pinned" : ""}`}
                          aria-label={t(pinned ? "unpinSkill" : "pinSkill", { name: skill.name })}
                          aria-pressed={pinned}
                          onClick={(event) => {
                            event.stopPropagation();
                            togglePinnedSkill(source.id, skill.id);
                          }}
                        >
                          <svg viewBox="0 0 16 16" aria-hidden="true">
                            <path fill="currentColor" d="M5.8 2.1h4.4l-.45 4.2 2.05 2.05v1.1H8.6V14H7.4V9.45H4.2v-1.1L6.25 6.3 5.8 2.1Zm1.35 1.2.36 3.38-1.57 1.57h4.12L8.49 6.68l.36-3.38h-1.7Z" />
                          </svg>
                        </button>
                      </div>
                    );
                  })}
                  {sourceSkills.length === 0 && (
                    <p className="muted skillListEmpty">
                      {hasNoTagsSelected ? t("noSkillsSelected") : allTags.length > 0 ? t("noSkillsForTag") : t("noSkillsDiscovered")}
                    </p>
                  )}
                </div>
              )}
            </article>
          );
        })}
        {sources.length === 0 && <Empty title={t("noSourcesTitle")} body={t("noSourcesBody")} />}
      </div>

      {activeSkill && <SkillModal t={t} skill={activeSkill} agents={agents} run={run} onClose={() => setActiveSkillId(null)} />}
      {deletingSource && (
        <DeleteSourceModal
          source={deletingSource}
          busy={busyLabel === t("deleteSource", { name: deletingSource.id })}
          onClose={() => setDeletingSource(null)}
          onConfirm={() => void removeSource(deletingSource)}
          t={t}
        />
      )}
      {pickerOpen && (
        <DirPicker
          t={t}
          initialPath={enable.projectRoot}
          onPick={(path) => { setEnable((prev) => ({ ...prev, projectRoot: path })); setPickerOpen(false); }}
          onClose={() => setPickerOpen(false)}
        />
      )}
    </section>
  );
}

function SkillModal({
  t,
  skill,
  agents,
  run,
  onClose,
}: {
  t: Translate;
  skill: Skill;
  agents: Record<string, AgentConfig>;
  run: (label: string, fn: () => Promise<unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const [content, setContent] = useState<string | null>(null);
  const [tree, setTree] = useState<SkillTree | null>(null);
  const [error, setError] = useState("");
  const [treeError, setTreeError] = useState("");
  const [tagInput, setTagInput] = useState("");
  const [enable, setEnable] = useState({ agent: agentNames(agents)[0] || "", scope: "user", projectRoot: "" });
  const agentOptions = agentNames(agents).map((agent) => ({ value: agent, label: activationAgentLabel(agent) }));
  const scopeOptions = [
    { value: "user", label: t("global") },
    { value: "project", label: t("project") },
  ];
  const tags = skill.tags || [];
  const activations = skill.activations || [];

  useEffect(() => {
    let cancelled = false;
    setContent(null);
    setTree(null);
    setError("");
    setTreeError("");
    void api<{ content: string }>(`/api/skills/${encodeURIComponent(skill.id)}/content`)
      .then((data) => {
        if (!cancelled) setContent(data.content || "");
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      });
    void api<SkillTree>(`/api/skills/${encodeURIComponent(skill.id)}/tree`)
      .then((data) => {
        if (!cancelled) setTree(data);
      })
      .catch((err) => {
        if (!cancelled) setTreeError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      cancelled = true;
    };
  }, [skill.id]);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape" && !document.querySelector(".customSelect.open")) onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const { meta, frontmatter, body } = useMemo(
    () => (content == null ? { meta: null, frontmatter: null, body: "" } : parseFrontmatter(content)),
    [content],
  );

  const html = useMemo(() => {
    if (!body) return "";
    const raw = marked.parse(body, { async: false }) as string;
    return DOMPurify.sanitize(raw);
  }, [body]);

  function openSkillDir(path: string) {
    run("Open directory", () => api(`/api/skills/${encodeURIComponent(skill.id)}/open-dir`, { method: "POST", body: JSON.stringify({ path }) }));
  }

  function openSkillPath(path: string) {
    run("Open item", () => api(`/api/skills/${encodeURIComponent(skill.id)}/open-path`, { method: "POST", body: JSON.stringify({ path }) }));
  }

  return (
    <div className="modalOverlay" onClick={onClose}>
      <div className="modal" role="dialog" aria-modal="true" aria-label={skill.name} onClick={(e) => e.stopPropagation()}>
        <div className="modalHead">
          <div>
            <h2>{skill.name}</h2>
            <p className="mono small">{skill.id}</p>
          </div>
          <button onClick={onClose}><IconButtonContent icon="close">{t("close")}</IconButtonContent></button>
        </div>
        <div className="modalBody">
          <div className="skillControls">
            <div className="controlBlock">
              <h4>{t("tags")}</h4>
              <div className="tagRow">
                {tags.length === 0 && <span className="muted">{t("noTags")}</span>}
                {tags.map((tag) => (
                  <span className="tag removable" key={tag}>
                    {tag}
                    <button
                      className="tagRemove"
                      aria-label={`Remove tag ${tag}`}
                      onClick={() => run("Remove tag", () => api("/api/skills/tags", { method: "POST", body: JSON.stringify({ skill_ids: [skill.id], tags: [tag], action: "remove" }) }))}
                    >
                      ×
                    </button>
                  </span>
                ))}
              </div>
              <div className="row">
                <input
                  placeholder={t("tagPlaceholderSingle")}
                  value={tagInput}
                  onChange={(e) => setTagInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && splitTags(tagInput).length) {
                      e.preventDefault();
                      run("Add tags", () => api("/api/skills/tags", { method: "POST", body: JSON.stringify({ skill_ids: [skill.id], tags: splitTags(tagInput), action: "add" }) }));
                      setTagInput("");
                    }
                  }}
                />
                <button
                  disabled={!splitTags(tagInput).length}
                  onClick={() => {
                    run("Add tags", () => api("/api/skills/tags", { method: "POST", body: JSON.stringify({ skill_ids: [skill.id], tags: splitTags(tagInput), action: "add" }) }));
                    setTagInput("");
                  }}
                >
                  <IconButtonContent icon="tagAdd">{t("add")}</IconButtonContent>
                </button>
              </div>
            </div>

            <div className="controlBlock">
              <div className="controlTitleRow">
                <h4>{t("enable")}</h4>
                <span className={`enableState ${activations.length > 0 ? "enabled" : "idle"}`}>
                  <span className="enableStateDot" aria-hidden="true" />
                  {activations.length > 0 ? t("enabled") : t("notEnabled")}
                </span>
              </div>
              {activations.length > 0 && (
                <div className="activationChipRow">
                  {activations.map((activation) => {
                    const label = `${activationAgentLabel(activation.agent)}: ${scopeText(activation.scope, t)}`;
                    const title = activation.project_root ? `${label} · ${activation.project_root}` : label;
                    return (
                      <span className="activationChip removable" key={activation.id} title={title}>
                        <span className="activationLabel">
                          <strong>{activationAgentLabel(activation.agent)}</strong>
                          {`: ${scopeText(activation.scope, t)}`}
                        </span>
                        {activation.project_root && <span className="activationProject">{activation.project_root}</span>}
                        <button
                          className="tagRemove"
                          aria-label={`${t("disable")} ${label}`}
                          onClick={() => run("Disable skill", () => api(`/api/activations/${activation.id}`, { method: "DELETE" }))}
                        >
                          ×
                        </button>
                      </span>
                    );
                  })}
                </div>
              )}
              <div className="row enableRow">
                <CustomSelect
                  ariaLabel={t("agent")}
                  value={enable.agent}
                  placeholder={`${t("agent")}...`}
                  options={agentOptions}
                  onChange={(agent) => setEnable({ ...enable, agent })}
                />
                <CustomSelect
                  ariaLabel={t("activationScope")}
                  value={enable.scope}
                  placeholder={t("activationScope")}
                  options={scopeOptions}
                  onChange={(scope) => setEnable({ ...enable, scope })}
                />
                {enable.scope === "project" && (
                  <input className="enableProjectInput" placeholder={t("projectPathPlaceholder")} value={enable.projectRoot} onChange={(e) => setEnable({ ...enable, projectRoot: e.target.value })} />
                )}
                <button
                  className="primary enableSubmitButton"
                  disabled={!enable.agent}
                  onClick={() => run("Enable skill", () => api("/api/activations", { method: "POST", body: JSON.stringify({ skill_ids: [skill.id], agent: enable.agent, scope: enable.scope, project_root: enable.projectRoot }) }))}
                >
                  <IconButtonContent icon="enable">{t("enable")}</IconButtonContent>
                </button>
              </div>
            </div>
          </div>

          <hr className="modalDivider" />

          {error && <p className="warning">{error}</p>}
          {content == null && !error && <p className="muted">{t("loading")}</p>}
          <SkillTreeView t={t} tree={tree} error={treeError} onOpenDir={openSkillDir} onOpenPath={openSkillPath} />
          {frontmatter && (meta ? <FrontmatterTable meta={meta} /> : <FrontmatterRaw value={frontmatter} />)}
          {content != null && body && <h3 className="contentSectionTitle">{t("skillMarkdown")}</h3>}
          {content != null && body && <div className="markdown" dangerouslySetInnerHTML={{ __html: html }} />}
        </div>
      </div>
    </div>
  );
}

function SkillTreeView({
  t,
  tree,
  error,
  onOpenDir,
  onOpenPath,
}: {
  t: Translate;
  tree: SkillTree | null;
  error: string;
  onOpenDir: (path: string) => void;
  onOpenPath: (path: string) => void;
}) {
  return (
    <section className="skillTreePanel" aria-labelledby="skill-tree-title">
      <div className="skillTreeHead">
        <div>
          <h3 id="skill-tree-title">{t("skillContents")}</h3>
          <p className="muted small">{t("skillContentsDescription")}</p>
        </div>
        {tree?.root && <p className="mono small">{tree.root}</p>}
      </div>
      {error && <p className="warning">{t("skillContentsUnavailable")} {error}</p>}
      {!tree && !error && <p className="muted">{t("loading")}</p>}
      {tree && tree.entries.length === 0 && <p className="muted">{t("skillContentsUnavailable")}</p>}
      {tree && tree.entries.length > 0 && (
        <div className="skillTree" role="tree">
          {tree.entries.map((entry) => (
            <SkillTreeNode key={entry.path || entry.name} t={t} entry={entry} level={1} onOpenDir={onOpenDir} onOpenPath={onOpenPath} />
          ))}
        </div>
      )}
    </section>
  );
}

function SkillTreeNode({
  t,
  entry,
  level,
  onOpenDir,
  onOpenPath,
}: {
  t: Translate;
  entry: SkillTreeEntry;
  level: number;
  onOpenDir: (path: string) => void;
  onOpenPath: (path: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const isDir = entry.kind === "dir";
  const hasChildren = isDir && !!entry.children?.length;
  return (
    <div className="skillTreeNode" role="treeitem" aria-level={level} aria-expanded={hasChildren ? expanded : undefined}>
      <div className={`skillTreeItem ${isDir ? "dir" : "file"}`} style={{ paddingLeft: `${(level - 1) * 1.1}rem` }}>
        {hasChildren ? (
          <button
            type="button"
            className="skillTreeToggle"
            aria-label={t(expanded ? "collapseDirectory" : "expandDirectory", { name: entry.name })}
            aria-expanded={expanded}
            onClick={() => setExpanded((value) => !value)}
          >
            {expanded ? "▾" : "▸"}
          </button>
        ) : (
          <span className="skillTreeGlyph" aria-hidden="true">{isDir ? "▸" : "·"}</span>
        )}
        {isDir ? (
          <button type="button" className="skillTreeOpen" onClick={() => onOpenDir(entry.path)}>
            {entry.name}
          </button>
        ) : (
          <button type="button" className="skillTreeOpen skillTreeFile" onClick={() => onOpenPath(entry.path)}>
            {entry.name}
          </button>
        )}
      </div>
      {hasChildren && expanded && (
        <div role="group">
          {entry.children!.map((child) => (
            <SkillTreeNode key={child.path || child.name} t={t} entry={child} level={level + 1} onOpenDir={onOpenDir} onOpenPath={onOpenPath} />
          ))}
        </div>
      )}
    </div>
  );
}

function parseFrontmatter(text: string): { meta: Record<string, unknown> | null; frontmatter: string | null; body: string } {
  const match = text.match(/^\uFEFF?---[ \t]*\r?\n([\s\S]*?)\r?\n---[ \t]*\r?\n?/);
  if (!match) return { meta: null, frontmatter: null, body: text };
  const body = text.slice(match[0].length);
  const frontmatter = match[1];
  try {
    const parsed = yaml.load(frontmatter);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return { meta: parsed as Record<string, unknown>, frontmatter, body };
    }
  } catch {
    // Malformed YAML should still be shown as front matter, not rendered as markdown.
  }
  return { meta: null, frontmatter, body };
}

function FrontmatterRaw({ value }: { value: string }) {
  return (
    <pre className="frontmatterRaw">
      <code>{value}</code>
    </pre>
  );
}

function FrontmatterTable({ meta }: { meta: Record<string, unknown> }) {
  const entries = Object.entries(meta);
  if (!entries.length) return null;
  return (
    <table className="frontmatter">
      <tbody>
        {entries.map(([key, value]) => (
          <tr key={key}>
            <th>{key}</th>
            <td>{formatMetaValue(value)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function formatMetaValue(value: unknown): string {
  if (value == null) return "";
  if (Array.isArray(value)) return value.map((item) => formatMetaValue(item)).join(", ");
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function DoctorPanel({ checks, onClose, t }: { checks: DoctorCheck[]; onClose: () => void; t: Translate }) {
  const failed = checks.filter((check) => !check.ok).length;
  const [remaining, setRemaining] = useState(3);
  useEffect(() => {
    const timer = window.setTimeout(() => {
      if (remaining <= 1) {
        onClose();
      } else {
        setRemaining((value) => value - 1);
      }
    }, 1000);
    return () => window.clearTimeout(timer);
  }, [onClose, remaining]);
  return (
    <aside className="doctor" role="status">
      <div className="row">
        <h2>{t("doctor")}</h2>
        <button onClick={onClose}><IconButtonContent icon="close">{t("closeCountdown", { seconds: remaining })}</IconButtonContent></button>
      </div>
      <p className={failed ? "warning" : "success"}>{failed ? t("issuesFound", { count: failed }) : t("allChecksPassed")}</p>
      <div className="stack compact">
        {checks.map((check, index) => (
          <div className="doctorItem" key={`${check.name}-${index}`}>
            <Status value={check.ok ? "OK" : "Issue"} t={t} />
            <div>
              <strong>{check.name}</strong>
              {check.path && <p className="mono small">{check.path}</p>}
              {check.message && <p className="muted">{check.message}</p>}
            </div>
          </div>
        ))}
      </div>
    </aside>
  );
}

function Status({ value, t }: { value: string; t?: Translate }) {
  const normalized = value.toLowerCase().replace(/\s+/g, "-");
  return <span className={`status ${normalized}`}>{t ? statusText(normalized, value, t) : value}</span>;
}

function statusText(normalized: string, fallback: string, t: Translate) {
  switch (normalized) {
    case "ok":
      return t("statusOk");
    case "issue":
      return t("statusIssue");
    case "up-to-date":
      return t("statusUpToDate");
    case "update-available":
      return t("statusUpdateAvailable");
    case "local-changes":
      return t("statusLocalChanges");
    case "local-source":
      return t("statusLocalSource");
    case "sync-failed":
      return t("statusSyncFailed");
    default:
      return fallback;
  }
}

function Empty({ title, body }: { title: string; body: string }) {
  return (
    <div className="empty">
      <h3>{title}</h3>
      <p>{body}</p>
    </div>
  );
}

function splitTags(input: string) {
  return input.split(",").map((tag) => tag.trim()).filter(Boolean);
}

function unique(items: string[]) {
  return Array.from(new Set(items)).sort((a, b) => a.localeCompare(b));
}

createRoot(document.getElementById("root")!).render(<App />);
