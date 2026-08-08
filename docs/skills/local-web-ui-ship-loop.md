# local-web-ui-ship-loop

`local-web-ui-ship-loop` 是小范围本地 Web UI 修改的执行闭环。它覆盖读代码、做最小改动、构建测试、浏览器验证，以及在用户要求时提交、推送和发布。

| 项 | 内容 |
| --- | --- |
| 类型 | UI 交付循环 |
| 原始 skill | `personal-agent-skills/skills/local-web-ui-ship-loop/SKILL.md` |
| 主要产出 | 已验证的 UI 改动、构建/测试结果、浏览器行为验证记录、可选 commit/push/tag |

## 何时使用

- 用户要求实现、修复或打磨本地 Web UI，而不只是给建议。
- 项目类似 React/Vite 前端或 Go embedded frontend。
- 需要确认可见布局、交互行为、语言文案和构建产物是否一致。

## 工作方式

1. 用一句话明确改动范围和行为假设。
2. 用 `rg` 找组件、CSS、可见文案、API 和 i18n key。
3. 在受影响组件附近做最小改动，复用已有控件和样式。
4. 机械验证：前端 build、后端 test、`git diff --check`。
5. 浏览器验证：启动本地服务，测试目标行为，必要时测量 DOM 尺寸。
6. 只有用户要求或项目规范明确时才 commit/push。

## 验证重点

- 不用“构建通过”替代可视化验证。
- 中英文文案同时存在时要一起检查。
- Go 嵌入前端资产时，构建前端后还要确认服务加载的是新 bundle。
- 结束时停止自己启动的本地服务。

## 边界与注意

- 不扩大成重设计。
- 不抽象单一调用点。
- 不把本地生成数据、测试草稿或用户 skill 意外 staged。

## 导航

上一项：[organize-html-reports](organize-html-reports.html) · [返回首页](../index.html) · 下一项：[init-agents-md](init-agents-md.html)
