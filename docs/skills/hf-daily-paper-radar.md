# hf-daily-paper-radar

`hf-daily-paper-radar` 用于抓取 Hugging Face Daily Papers，按用户给出的研究兴趣排序、总结并保存本地 Markdown 报告。它把官方 Hugging Face 数据源作为论文列表的准确信源。

| 项 | 内容 |
| --- | --- |
| 类型 | 论文雷达 |
| 原始 skill | `personal-agent-skills/skills/hf-daily-paper-radar/SKILL.md` |
| 主要产出 | 当前目录下的 Markdown 论文雷达报告、趋势总结、推荐阅读顺序 |

## 何时使用

- 用户要求阅读、筛选、总结、优先级排序 Hugging Face Daily Papers。
- 用户给定研究兴趣，例如 latent reasoning、on-policy distillation、agentic RL 或评测安全方向。
- 用户说“未读 HF Daily Papers 邮件”时，也可以先用 Apple Mail 识别日期，再从官方 Hugging Face 源拉取论文。

## 工作方式

1. 优先使用官方 `hf papers` CLI；不可用时使用 skill 自带 REST helper。
2. 解析日期：显式日期、today/yesterday、latest 或 last N issues 都要转成具体 issue。
3. 规范化论文数据，包括标题、摘要、upvotes、关键词、HF/arXiv/project/GitHub 链接和发布时间。
4. 按主兴趣、方法相关性、迁移价值、次级兴趣和 upvotes 进行综合排序。
5. 分组为 Direct Hits、Adjacent Methods、Agentic RL / Evaluation / Safety 和 Peripheral。
6. 保存中文 Markdown 报告，再询问是否发布到飞书。

## 报告结构

报告应包含标题、来源日期、用户关注点、执行摘要、排序后的论文分析、方向级趋势和推荐阅读顺序。重点是趋势和方法启发，而不是机械复述摘要。

## 边界与注意

- 不默认安装或升级 Hugging Face 工具。
- 不下载所有 PDF，除非用户要求深读。
- Apple Mail 只用于识别 digest 日期和读状态，不作为最终论文数据源。
- 发布到飞书必须先获得用户确认。

## 导航

上一项：[arxiv-daily-paper-radar](arxiv-daily-paper-radar.html) · [返回首页](../index.html) · 下一项：[research-ui-frontend](research-ui-frontend.html)
