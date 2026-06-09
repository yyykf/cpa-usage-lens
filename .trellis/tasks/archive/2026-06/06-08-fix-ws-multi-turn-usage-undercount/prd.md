# Fix WS multi-turn usage undercount

## Goal

修复 lens 对 CPA WebSocket responses 路径用量的**系统性漏记（约 71%）+ 缓存命中率统计塌陷**。

根因（已由上游 fork 插桩抓帧 + Supabase 对账 100% 锁定）：CPA 的 ws responses 路径下，**一个 ws 连接（= 一个 HTTP 升级请求）承载的多轮对话共享同一个 `request_id`**。lens 用 `request_id` 做 `request_events_hot` 主键，入库时 `ON CONFLICT (request_id) DO NOTHING` → **每个 ws 连接只入库第一轮，后续轮（含高 cache 深层轮）被静默丢弃**。

修复：把去重键从「会误杀多轮的 `request_id`」换成「能区分多轮的复合键」`(request_id, event_ts, total_tokens)`，**保留 `ON CONFLICT ... DO NOTHING`**——既修漏记，又不丢崩溃恢复的幂等性。

## Requirements

- `request_events_hot` 主键从 `(request_id)` 改为复合主键 `(request_id, event_ts, total_tokens)`。
- `request_id` 降为普通列（保留，用于日志关联）。复合主键 B-tree 的最左前缀已覆盖按 `request_id` 的等值查询，**无需额外单列索引**。
- `events.go` `InsertEvents` 的 `ON CONFLICT` 目标从 `(request_id)` 改为 `(request_id, event_ts, total_tokens)`，**保留 `DO NOTHING`**（幂等语义不变，批量/计数/错误处理逻辑全部不动）。
- 新增 migration 幂等可重跑（承袭 `20260605002633_add_api_key_dimension.sql` 的 `DO $$` 块按主键列名集合判定的范式），存量数据平滑迁移、可直接在 Supabase SQL Editor 重复执行。
- 同步更新「按 `request_id` 去重」的过时措辞：`request_events_hot` 表注释（用新 migration 的 `comment on` 覆盖）、`events.go:10` 函数注释、`collector.go:18` 注释，以及 README / docs 中相关描述（实施时 grep 核对）。
- **CPA 不改动**：`request_id` 连接级共享是上游既定行为（用途为日志关联，CPA PR #3162），ADR 0001 已定调不动上游。

## Acceptance Criteria

- [ ] migration 在 Supabase 上幂等执行：连续跑两次均不报错，主键最终为 `(request_id, event_ts, total_tokens)`。
- [ ] 存量行平滑迁移：迁移前 `request_id` 唯一 → 复合键必然唯一 → 加复合主键不冲突、不丢行。
- [ ] 同一 ws 连接多轮（同 `request_id`、不同 `event_ts`）入库后**各成一行**，不再被去重吞掉。
- [ ] 崩溃恢复幂等仍成立：buffer recover 重放**完全相同**的记录时被 `DO NOTHING` 跳过，不产生重复行。
- [ ] `go build ./...`、`go vet ./...`、`go test ./...` 全绿；新增/更新测试覆盖「同 `request_id` 多轮入库」与「同记录重放幂等」（视现有 db 层测试基建，必要时在 collector/db 层补）。

## Definition of Done

- 上述 Acceptance Criteria 全部满足。
- 测试 / lint / vet / build 通过。
- 行为变更涉及的注释与文档已同步（无残留「request_id 去重」描述）。
- **双关验收**（按 code4j 既定规矩，碰生产功能必走）：AI 自测 + Codex review，两者都过才提交。
- 生产 Supabase migration 的执行方式与时机在实施收尾时与 code4j 确认（docs 已提供「SQL Editor 直接跑」路径）。

## Technical Approach

**方案丙：复合主键 `(request_id, event_ts, total_tokens)` + 保留 `ON CONFLICT DO NOTHING`。**

为什么这三列能同时满足「修漏记」和「保幂等」：

- **修漏记**：同连接多轮 `request_id` 相同，但 `event_ts` 每轮独立、不同 → 复合键不同 → 各轮都入库。
- **保幂等**：buffer 重放的是**同一条物理记录**，三列一字不差 → 复合键相同 → `DO NOTHING` 跳过。

为什么 `event_ts` 可靠（关键事实，已核实）：

- CPA 的 usage `timestamp` 源自 `UsageReporter.requestedAt = time.Now()`，字段为 `time.Time` 且无自定义 `MarshalJSON` → 默认序列化为 **RFC3339Nano（纳秒级）**。
- **每轮独立**：每轮 `ExecuteStream` 各自 `new reporter` 打一次 `time.Now()`，与 `request_id` 的「连接级共享」**不是同一回事**。
- **串行保证**：同一 ws 连接的多轮在 CPA 侧串行执行（锁串行复用上游连接），第二轮必等第一轮 publish 完才开始 → 两轮 `time.Now()` 至少差一整轮推理时间（数百 ms~秒级）→ **同连接两轮 `event_ts` 落在同一微秒在机制上不可能**。
- lens 现有 `time.Parse(time.RFC3339, ...)` 解析会**自动吸收并保留亚秒**（已写最小程序实测：纳秒原样保留）→ 落库 Postgres `timestamptz`（微秒精度）足以区分多轮。

`total_tokens` 的作用：在 `event_ts` 已足够的基础上**再加一道独立保险**——即使未来 CPA 改了执行模型让 `event_ts` 不再可靠，`total` 不同仍能区分多轮。成本近乎为零（主键多一列、改动与纯 `(request_id, event_ts)` 完全一样），正好回应「本 bug 的根因就是把正确性单押在一个上游字段行为上」的教训。

存量迁移：现有主键即 `request_id`（唯一），故存量行的 `(request_id, event_ts, total_tokens)` 必然唯一，drop 旧主键 → add 复合主键不会冲突，无需回填、无需停机。

## Decision (ADR-lite)

**Context**：`request_id` 在 CPA ws 路径是连接级共享，lens 拿它做主键 + `ON CONFLICT DO NOTHING` 导致同连接多轮被静默去重，系统性漏记 ~71%。需要一个既能区分多轮、又不丢崩溃恢复幂等的去重键。

**Decision**：采用复合主键 `(request_id, event_ts, total_tokens)`，保留 `ON CONFLICT ... DO NOTHING`。

**备选与否决理由**：
- **甲 `(request_id, event_ts)`**：正确且更纯，但去重正确性单押 `event_ts` 一个上游字段；丙以近乎为零的成本多加一层兜底，更稳健。
- **乙 `event_hash`（全字段 sha256 指纹）**：最鲁棒（不赌任何上游字段），但需加列 + 改 sanitize/model + 算 hash；在 `event_ts` 已被证实可靠（纳秒、每轮独立、串行保证）后，其额外鲁棒性是趋近 0 的边际收益，按 YAGNI 不值；且 sha256 hex（64B）索引比复合键（~25B）更大。
- **代理主键（自增 id）+ 纯 insert**：会丢掉崩溃恢复幂等（重放生成新 id → 重复行），单事务也只能压窗口不能恢复幂等，否决。

**Consequences**：
- daily 聚合链路（`RollupRange`）是 hot 表的纯派生（`count(*)`/`sum(...)` + 覆盖式 upsert，无 request_id 概念），hot 入库修对后 daily 自动正确，**聚合层零改动**。
- 残留尾巴：仅「同连接两轮在同一微秒完成且 `total_tokens` 相同」才会误去重——被 CPA 串行机制堵死，趋近 0；即便发生，丢的也是 token=0 的 failed 记录（对用量/成本零影响）。
- **历史漏记不可恢复**：漏掉的轮当时根本没进 hot 表，物理不存在；修复只对部署后新 pop 的数据生效。

## Out of Scope

- 历史漏记数据回补（物理不可恢复）。
- 改动 CPA 上游。
- `event_hash` 全字段指纹方案（YAGNI，见 ADR-lite）。
- usage-queue 的 redis `SUBSCRIBE` 抢占风险（CPA 侧真实代码风险，但与本次漏记无关，属上游议题）。

## Technical Notes

改动文件（预估）：
- `supabase/migrations/<新时间戳>_fix_hot_composite_pk.sql`（新增）
- `backend/internal/db/events.go`（`ON CONFLICT` 目标 + 函数注释）
- `backend/internal/collector/collector.go`（注释措辞）
- 文档/README 中「request_id 去重」描述（grep 核对）

参考（公开、仓库无关）：
- CPA 上游：`request_id` 为连接级共享、用途是日志关联（上游已将其固化为连接级 context 取值）——故下游不能拿它当唯一键。
- 生态先例：`Willxup/cpa-usage-keeper` 的 migration `removeUsageEventEventKeyUniqueIndexMigration`（2026-05-14，拆 request_id 唯一约束、每条队列消息独立入库）。
- 项目内决策记录（lens 消费者视角，长期 living）：`.project_context/design/decisions/0001-usage-hot-composite-pk.md`

> 根因经上游 fork 插桩抓帧（WSPROBE 日志）+ Supabase 实时对账实证锁定；详细调研报告与决策记录留存于本地、**未纳入本公开仓库**。关键结论、实测证据与取舍均已内联于本 prd 的「Technical Approach」「Decision」两节，实施不依赖任何外部文件。

幂等 migration 范式参考（本仓库）：`supabase/migrations/20260605002633_add_api_key_dimension.sql` 的 `DO $$ ... $$` 按主键列名集合判定块。

数据流关键点：collector `pollOnce`（pop → 落盘 buffer → InsertEvents → commit）+ `recoverPending`（重放未 commit 批次）→ 重放幂等依赖 `ON CONFLICT DO NOTHING` 保留。
