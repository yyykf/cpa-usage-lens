# ADR 0003: CPA v2 accounting is canonical with legacy fallback

- 状态：已接受
- 日期：2026-07-26

## 背景

CLIProxyAPI v7.2.97 在 Redis usage 队列中增加 `accounting_version: 2` 和互斥的
`token_breakdown`，并用 `complete`、`unclassified`、`inconsistent` 表达拆桶质量。
Lens 旧逻辑只能按 provider 和 legacy 字段形态推断，无法说明成本覆盖是否完整。

## 决策

- 合法 v2 breakdown 是 canonical source of truth，入口校验一次后贯穿 hot、daily、API 和前端。
- 旧 CPA payload 保留 provider-aware fallback，并标记为 legacy。
- 成本只累计可可靠分类的桶；未分类和不一致部分不猜价格。
- 少量低质量数据产生“部分统计”，不清空同范围内其他可靠成本；完全没有可靠分类数据时才显示未知。
- 保留 legacy 原始字段用于兼容和诊断，不做历史推测式回填。

## 后果

收益是六桶相互排斥、成本覆盖可解释，未知 provider 不再被静默猜测。成本是增加一次向前数据库迁移，
且历史 legacy 数据仍只能使用旧语义。回退旧 binary 前必须先确认其 SQL 不依赖新增列；新增列本身不改变既有主键。
