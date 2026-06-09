-- ============================================================
-- CPA Usage Lens — request_events_hot 改用复合主键去重（修 ws 多轮系统性漏记）
--
-- 背景（根因，已由上游 fork 插桩抓帧 + Supabase 对账实证锁定）：
--   CPA WebSocket responses 路径下，一个 ws 连接（= 一个 HTTP 升级请求）承载的多轮
--   对话**共享同一个 request_id**（上游仅在升级时生成一次，用途是日志关联）。
--   lens 初始 schema 用 request_id 单列作主键 + 入库 ON CONFLICT (request_id) DO NOTHING，
--   导致同连接的后续轮（含高 cache 的深层轮）主键冲突被静默丢弃 → 系统性漏记约 71%
--   + 缓存命中率统计塌陷。
--
-- 修复：把主键从 (request_id) 改为复合主键 (request_id, event_ts, total_tokens)。
--   · 修漏记：同连接多轮 request_id 相同，但 event_ts 每轮独立、不同（源自每轮各自的
--     time.Now()，纳秒精度、CPA 侧串行执行保证两轮不落同一微秒）→ 复合键不同 → 各轮都入库。
--   · 保幂等：崩溃恢复时 buffer 重放的是同一条物理记录，三列一字不差 → 复合键相同 →
--     入库侧保留的 ON CONFLICT ... DO NOTHING 跳过，不产生重复行
--     （破坏性 pop 不可回放，buffer 重放幂等是硬约束，绝不能丢）。
--   · total_tokens 作额外一层独立保险：即便未来上游改执行模型让 event_ts 不再可靠，
--     total 不同仍能区分多轮；成本近乎为零（主键多一列）。
--
-- request_id 降为普通列（保留，用于日志关联）。复合主键 B-tree 的最左前缀已覆盖按
--   request_id 的等值查询，无需额外单列索引。
--
-- 存量数据平滑迁移（无需回填、无需停机）：
--   迁移前主键即 request_id（唯一）→ 存量行的 (request_id, event_ts, total_tokens) 必然唯一
--   → drop 旧主键 / add 复合主键不会冲突、不丢行。
--
-- 幂等可重跑（承袭 20260605002633 的全幂等风格；docs 提供「直接在 Supabase SQL Editor 跑」
--   路径，重复执行不报错）：下面 DO 块按「主键实际列名集合」判定是否已迁移，仅当当前主键列集
--   不等于目标列集时才 drop + add。
-- ============================================================

-- ------------------------------------------------------------
-- 把 request_events_hot 主键改为复合 (request_id, event_ts, total_tokens)。
-- 包在 DO 块里按「主键实际列名集合」判定是否已迁移，保证整段可安全重跑。
-- ⚠️ 用列名集合（而非仅列数=3）判定：避免「列数对但列错」时被当成已迁移而跳过。
-- ------------------------------------------------------------
do $$
declare
  pk_cols text[];
begin
  -- 取当前主键约束的实际列名（按 attnum 升序聚合），不存在主键时为 NULL。
  select array_agg(a.attname order by a.attnum)
    into pk_cols
  from pg_constraint c
  join pg_attribute a
    on a.attrelid = c.conrelid
   and a.attnum = any(c.conkey)
  where c.conname = 'request_events_hot_pkey'
    and c.conrelid = 'public.request_events_hot'::regclass;

  -- 仅当列名集合「恰为」目标 3 列时才认定已迁移并跳过；任何偏差（缺列/多列/列名不符）都重建。
  if pk_cols is null
     or array(select unnest(pk_cols) order by 1)
        is distinct from array['event_ts', 'request_id', 'total_tokens'] then
    alter table public.request_events_hot drop constraint if exists request_events_hot_pkey;
    alter table public.request_events_hot
      add constraint request_events_hot_pkey
      primary key (request_id, event_ts, total_tokens);
  end if;
end $$;

-- 覆盖旧的「按 request_id 去重」表注释：去重键已是复合键，可区分同一 ws 连接的多轮。
comment on table public.request_events_hot is '热明细：每请求一行，短期保留（默认7天，可配）；按复合主键 (request_id, event_ts, total_tokens) 去重——request_id 在 CPA ws 路径为连接级共享，单列去重会误吞同连接多轮，复合键可区分；不存 api_key/response_headers/fail.body';
