# 0001 — request_events_hot 采用复合主键去重（弃用 request_id 单列主键）

- 状态：已确认
- 日期：2026-06-08
- 关联任务：06-08-fix-ws-multi-turn-usage-undercount

## 背景

lens 从 CPA `GET /v0/management/usage-queue` **破坏性 pop** 取 usage，写入 `request_events_hot`。初始 schema 用 `request_id` 做主键 + 入库 `ON CONFLICT (request_id) DO NOTHING` 去重。

CPA 的 WebSocket responses 路径下，一个 ws 连接（= 一个 HTTP 升级请求）承载多轮对话、**整连接共享同一个 `request_id`**（上游 ws 主循环每轮复用同一 `gin.Context`，`request_id` 仅在升级时生成一次，用途是日志关联）。结果：同连接的后续轮入库时主键冲突被 `DO NOTHING` 静默丢弃 → **系统性漏记约 71% + 缓存命中率统计塌陷**（丢的恰是高 cache 的深层轮）。

根因经上游 fork 插桩抓帧（WSPROBE）+ Supabase 实时对账实证锁定（详细调研留存本地、未纳入公开仓库）。

## 决策

`request_events_hot` 主键从 `(request_id)` 改为复合主键 **`(request_id, event_ts, total_tokens)`**，入库**保留 `ON CONFLICT ... DO NOTHING`**。

- **修漏记**：同连接多轮 `request_id` 相同，但 `event_ts` 每轮独立、不同 → 复合键不同 → 各轮都入库。
- **保幂等**：buffer 崩溃恢复重放的是同一条物理记录，三列一字不差 → 复合键相同 → 跳过，不产生重复行（破坏性 pop 不可回放，buffer 重放幂等是硬约束）。

关键依据（已核实）：CPA 的 usage `timestamp` 源自每轮 `time.Now()`、**纳秒精度、每轮独立**（与连接级共享的 `request_id` 不同）；同连接多轮在 CPA 侧串行执行（锁串行复用上游连接），两轮 `event_ts` 落同一微秒在机制上不可能。lens 现有 `time.Parse(time.RFC3339, ...)` 会自动吸收并保留亚秒（已实测），落库 `timestamptz` 微秒精度足够区分。`total_tokens` 作为额外一层独立保险（成本近乎为零）。

## 否决的备选

- **`(request_id, event_ts)`**：正确，但去重正确性单押 `event_ts` 一个上游字段行为；复合加 `total_tokens` 以近乎零成本多一层兜底，正好回应「本 bug 根因就是把正确性单押在一个上游字段（request_id）上」的教训。
- **`event_hash`（全字段 sha256 指纹）**：最鲁棒（不赌任何上游字段），但需加列 + 改 sanitize/model + 算 hash；在 `event_ts` 已被证实可靠后，其额外鲁棒性是趋近 0 的边际，按 YAGNI 不值；sha256 hex 索引也比复合键（~25B/行）更大。
- **代理主键（自增 id）+ 纯 insert**：会丢崩溃恢复幂等（重放生成新 id → 重复行），单事务只能压窗口、不能恢复幂等。

## 后果

- daily 聚合（`RollupRange`）是 hot 表的**纯派生**（`count(*)`/`sum(...)` + 覆盖式 upsert，无 request_id 概念）→ hot 入库修对后 daily 自动正确，**聚合层零改动**。
- 残留尾巴：仅「同连接两轮同一微秒完成且 `total_tokens` 相同」才误去重——被 CPA 串行机制堵死、趋近 0；即便发生，丢的也是 token=0 的 failed 记录（对用量/成本零影响）。
- **历史漏记不可恢复**：漏掉的轮当时未入 hot 表、物理不存在；修复只对部署后新数据生效。
- **上游 CPA 不改动**：`request_id` 连接级共享是其既定行为（日志关联用途），属上游约定。

## 生态先例

`Willxup/cpa-usage-keeper` 早期同坑，2026-05-14 migration `removeUsageEventEventKeyUniqueIndexMigration` 拆 request_id 唯一约束、改每条队列消息独立入库。其它面板各自规避（CPA-Manager-Plus 用含 token 的复合 `event_hash` 等）。
