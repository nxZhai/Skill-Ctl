# hatch-pet

`hatch-pet` 用于从概念、参考图、公司或产品品牌线索生成 Codex 兼容的动画 pet 包。它把 `$imagegen` 用作视觉生成层，把确定性的脚本用于精灵图几何、透明背景、验证、预览和打包。

| 项 | 内容 |
| --- | --- |
| 类型 | 视觉资产 |
| 原始 skill | `personal-agent-skills/skills/hatch-pet/SKILL.md` |
| 主要产出 | `pet.json`、`spritesheet.webp`、contact sheet、逐行动画预览、验证 JSON |

## 何时使用

- 用户想创建、修复、验证或打包 Codex app 自定义动画 pet。
- 输入可能是文本概念、角色图、品牌/公司/产品名称或视觉参考。
- 需要完整 8x9 atlas、透明未用格、QA contact sheet 和 Codex pet 包。

## 工作方式

1. 如果用户只给品牌或公司名，先做轻量品牌发现，提炼视觉和气质线索。
2. 准备 pet run 目录和 `imagegen-jobs.json`。
3. 用 `$imagegen` 生成 base，再以 canonical base 约束每个动画 row。
4. 按状态生成 `idle`、`running-right`、`running-left`、`waving`、`jumping`、`failed`、`waiting`、`running` 和 `review`。
5. 只有在视觉安全时，才用脚本从 `running-right` 派生 `running-left`。
6. 完成所有视觉 job 后，运行帧提取、检查、atlas 合成、验证、contact sheet 和 GIF 预览。
7. 通过视觉 QA 后写入 Codex pet 目录，并清理中间产物。

## 质量标准

- 单格按 `192x208` 设计，最终 atlas 为 `1536x1872`。
- 同一个 pet 的脸、比例、材质、配色、道具和轮廓必须跨行一致。
- 背景应可干净移除，未用格应完全透明。
- 禁止文本、UI、可读 logo、阴影、光晕、漂浮特效、速度线、灰尘和会破坏透明提取的元素。
- contact sheet 和 row GIF 必须通过视觉 QA，不能只看 JSON 验证结果。

## 边界与注意

- 视觉生成只能走 `$imagegen`，不能换成自写图片生成脚本或其他图像 API。
- parent agent 负责 manifest、复制、确定性脚本、打包和清理；轻量 worker 只负责单个视觉 job 或最终视觉 QA。
- 修复失败时只重做最小失败范围，不重做整张表。

## 导航

上一项：[self](self.html) · [返回首页](../index.html)
