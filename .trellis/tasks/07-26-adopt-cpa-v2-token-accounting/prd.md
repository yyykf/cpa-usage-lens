# Adopt CPA v2 token accounting quality

## Goal

接入 CLIProxyAPI v7.2.97 新增的 `accounting_version: 2` 与规范化
`token_breakdown`，让 CPA Usage Lens 能区分完整、未分类和不一致的用量记录；
正常记录继续展示完整成本，少量低质量记录只影响自身，并以“部分统计”明确提示，
避免把不可信拆桶伪装成精确计费结果，也避免一条异常记录让整个页面长期显示“未知”。

## What I already know

- CPA v7.2.97 继续保留旧 `tokens` 字段，因此当前 Go JSON 解析不会因新增字段失败。
- CPA v2 accounting quality 包含 `complete`、`unclassified`、`inconsistent`。
- `complete` 的 input/output 子桶互斥且能与父级、总量对齐，可以精确拆桶估算成本。
- `unclassified` 仍可能有权威总 Token，也可能包含部分已分类桶；未分类部分不能猜价格。
- `inconsistent` 表示父子桶或总量互相矛盾，不可信部分不能参与成本估算。
- 当前主要场景是 Codex OAuth；正常 Codex/OpenAI 记录预期应为 `complete`。
- Lens v0.5.0 已实现 provider-aware cache/reasoning 去重和 long-context 计费，本任务不重做该逻辑。
- 当前 collector 会静默忽略 `accounting_version` 和 `token_breakdown`，数据库、API、前端也没有质量信息。
- 当前 Token 构成图只有未缓存输入、缓存读取、缓存写入、输出四段；Codex reasoning 包含在 output 父级中。

## Assumptions

- 新版 CPA v2 breakdown 对 v2 事件是 canonical source of truth；旧 CPA 或缺失 breakdown 的事件继续走现有 provider-aware legacy fallback。
- 历史聚合不做推测式回填，从新版 Lens 上线后开始记录 v2 accounting quality。
- 保留旧 token 字段，避免破坏旧 CPA 兼容和现有 API；canonical 字段作为增量契约接入。
- 不让单条非 `complete` 记录把整个查询范围成本置为未知。

## Requirements

- Collector 必须可选解析 `accounting_version` 和 `token_breakdown`，并兼容缺少这些字段的旧 CPA payload。
- 对 `accounting_version == 2` 的记录，必须校验 schema version、quality 枚举、非负数和三组加和不变量。
- 合法的 v2 breakdown 必须优先于 Lens 旧字段形态猜测；legacy payload 继续使用当前 provider-aware fallback。
- 必须保留权威总 Token，不能把 `unclassified` 等同于丢弃记录。
- `complete` 记录正常参与拆桶与成本估算。
- `unclassified` 记录的已分类部分可以参与成本估算，`unclassified_tokens` 不得被猜测分桶或计费。
- `inconsistent` 记录的不可信拆桶不得参与成本估算，但必须保留质量计数和诊断信息。
- 聚合 API 必须区分完整、部分统计和完全不可计费三种状态。
- 页面在少量非完整记录存在时继续展示可计费部分估算，并明确标注未分类 Token/异常记录未计入。
- 只有查询范围内不存在任何可靠可计费数据时，成本才显示“未知”。
- 前端 Token 构成必须端到端使用互斥 canonical 六桶：未缓存输入、缓存读取、缓存写入、普通输出、推理输出、无法分类。
- `unclassified_tokens == 0` 时允许隐藏“无法分类”图段，但 API 字段必须稳定存在。
- long-context 继续按原始 `input_tokens > model threshold` 判定，本任务不得改用 `total_tokens`。

## Acceptance Criteria (evolving)

- [x] CPA v7.2.97 payload 可解析且 v2 accounting 字段不会被静默丢弃。
- [x] 旧 CPA payload 仍通过 collector、rollup 和报表回归测试。
- [x] `complete` fixture 的 canonical 子桶相加等于总 Token，成本与当前正确的 Codex/OpenAI 口径一致。
- [x] `unclassified` fixture 保留总 Token，只计算已分类部分，并在 API/页面标为“部分统计”。
- [x] `inconsistent` fixture 不把矛盾桶计入成本，并在 API/页面暴露异常记录数量。
- [x] 单条非完整记录不会让同一查询范围内其他正常记录的已知成本消失。
- [x] 当查询范围内完全没有可靠可计费数据时，页面显示“未知”并解释原因。
- [x] Token 构成图使用六个互斥 canonical buckets，各段之和等于权威总 Token。
- [x] 普通输出与推理输出独立展示，且 Codex/OpenAI reasoning 不与 output 父级重复相加。
- [x] 旧 token aliases 不重复相加，Codex reasoning 不在 output 之外再次计费。
- [x] long-context 价格档位行为保持不变。
- [x] 数据库迁移、滚动边界、部署顺序和回退限制有文档说明。

## Definition of Done

- 后端单元测试、数据库集成测试和前端构建通过。
- `go test ./...`、`go vet ./...`、`gofmt -l .` 和 `npm run build` 通过。
- 新增真实形状的 CPA v7.2.97 complete/unclassified/inconsistent fixtures。
- API DTO、前端类型和页面状态保持跨层一致。
- 更新部署说明和 `.project_context/execution/usage/` 执行摘要。
- 使用 Conventional Commits，提交 body 末尾包含 `[#AI]`。

## Out of Scope (explicit)

- 不实现 Responses WebSocket `1001 upstream requires HTTP replay`；Lens 只消费 HTTP management usage queue。
- 不修改 CPA/Redis 配置、认证、启动参数或部署拓扑。
- 不推测回填历史 usage quality 或历史 canonical token buckets。
- 不根据缓存命中、时间顺序或 TTL 猜测 cache write。
- 不在本任务中移除旧 `tokens` 字段或旧 CPA 兼容路径。
- 不把本任务扩展到非 usage 的 CPA v7.2.97 WebSocket、压缩 transcript 或工具调用缓存修复。

## Technical Approach

- 在 collector raw DTO 中增加独立的 CPA accounting v2 类型，避免与现有前端 API 的 `TokenBreakdown` 命名/语义混淆。
- 热明细保存 accounting version、quality、canonical buckets 与 unclassified tokens；日聚合保存 canonical sums 和各质量请求计数。
- v2 事件使用 CPA canonical breakdown；legacy 事件保留当前 provider-aware 归一化。
- 报表分别累加可靠成本和不可计费 Token/记录，返回成本覆盖状态，而不是用一条异常记录把整个聚合置空。
- 前端根据覆盖状态显示正常估算、部分统计或未知。

## Decision (ADR-lite)

**Context**：CPA v7.2.97 已能表达拆桶质量，Lens 继续只读 legacy fields 会丢失统计可信度。

**Decision**：采用端到端完整适配：canonical-first、legacy-fallback；成本按可可靠分类部分累计，并单独暴露未分类与不一致范围；前端升级为 canonical 六桶构成，而不是只增加质量提示。

**Consequences**：需要跨 collector、数据库、rollup、report API 和 frontend 的迁移；收益是成本展示不再虚假精确，同时不会因少量异常完全不可用。

## Technical Notes

- `backend/internal/collector/payload.go`：当前 raw queue DTO，仅声明 legacy `tokens`。
- `backend/internal/collector/sanitize.go`：当前按 event provider 归一化 legacy cache aliases。
- `backend/internal/pricing/cost.go`：当前 provider-aware cost 和 legacy fallback。
- `backend/internal/db/events.go`、`backend/internal/db/rollup.go`：hot 与 daily 数据落点。
- `backend/internal/report/report.go`：成本和 token 聚合。
- `frontend/src/lib/tokens.ts`：当前四段 Token 构成。
- `supabase/migrations/`：需要新增向前迁移，不修改既有 migration。
- 上游参考：CLIProxyAPI `v7.2.97:sdk/cliproxy/usage/accounting.go` 与 `v7.2.97:internal/redisqueue/plugin.go`。
