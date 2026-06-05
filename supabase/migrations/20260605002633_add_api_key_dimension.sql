-- ============================================================
-- CPA Usage Lens — 增加「API key 用量」维度（脱敏存储）
--
-- 背景：把统计维度下钻到「客户端 API key」这一层（与账号 source 正交并列）。
-- 隐私底线（卖点不破）：明文 api_key 绝不入库，仅存不可逆的两列——
--   key_fingerprint = sha256(明文 api_key) 全长小写 hex（精确区分/做聚合主键）
--   key_mask        = token 族前缀…后4位（如 sk-…2216，界面展示用，不可逆；短 key 退化为定长占位）
-- 非 api_key 认证（oauth 等）或空 key → 落哨兵 key_fingerprint='none' 归「其他」桶。
--
-- 安全约定承袭 init：新列不破坏「开 RLS 且不建任何策略」的模型（backend 以 postgres
-- 角色 bypassrls 直连读写；前端永不直连 DB）。
--
-- 迁移顺序（关键）：daily 表先加带默认哨兵的列（让存量行能进新主键不冲突）→
--   drop 旧主键 → add 含 key_fingerprint 的新主键。
--
-- 存量历史的 key 归属：库里从未存过明文 api_key（隐私底线），**没有原始 key 可还原**。
--   故存量行本次统一落哨兵 key_fingerprint='none' 归「其他」桶；之后由独立回填脚本按
--   code4j 的决策把存量统一归入其指定的当前在用 key（见 PRD「回填策略」），未回填前留在 none 桶。
--   这不是「等以后能恢复真实 key」——真实 key 物理上不存在，只能按约定整体归并。
-- ============================================================

-- ------------------------------------------------------------
-- 1) request_events_hot（热明细）：可空两列，明文剥离后仅存指纹+掩码
--    可空：明细短期保留 + 非 key 认证场景容忍空（聚合表才用哨兵兜底）
-- ------------------------------------------------------------
alter table public.request_events_hot
    add column if not exists key_fingerprint text,   -- sha256(明文 api_key) 全长小写 hex；明文绝不入库
    add column if not exists key_mask        text;   -- token 族前缀…后4位（如 sk-…2216），界面展示用

-- ------------------------------------------------------------
-- 2) daily_account_usage（日聚合）：key 进主键，长期可见
--    NOT NULL + 默认哨兵 'none' / ''，存量行先平滑落桶；新主键加上 key_fingerprint
-- ------------------------------------------------------------
alter table public.daily_account_usage
    add column if not exists key_fingerprint text not null default 'none',  -- 'none' = 非 key 认证/未知归属
    add column if not exists key_mask        text not null default '';      -- 随指纹带出的展示掩码

-- 扩主键为 (天×账号×模型×key)：先去旧主键约束，再建新的。
-- 包在 DO 块里按「主键实际列名集合」判定是否已迁移，保证整段可安全重跑
--（承袭 init 的全幂等风格；docs 提供「直接在 SQL Editor 跑」路径，重复执行不报错）。
-- ⚠️ 用列名集合（而非仅列数=4）判定：避免「列数对但列错」（如误带了别的列）时被当成已迁移而跳过。
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
  where c.conname = 'daily_account_usage_pkey'
    and c.conrelid = 'public.daily_account_usage'::regclass;

  -- 仅当列名集合「恰为」目标 4 列时才认定已迁移并跳过；任何偏差（缺列/多列/列名不符）都重建。
  if pk_cols is null
     or array(select unnest(pk_cols) order by 1)
        is distinct from array['key_fingerprint', 'model', 'source', 'usage_date'] then
    alter table public.daily_account_usage drop constraint if exists daily_account_usage_pkey;
    alter table public.daily_account_usage
      add constraint daily_account_usage_pkey
      primary key (usage_date, source, model, key_fingerprint);
  end if;
end $$;

comment on column public.daily_account_usage.key_fingerprint is 'sha256(明文 api_key) 全长小写 hex；明文绝不入库；非 key 认证/未知归属落哨兵 none';
comment on column public.daily_account_usage.key_mask is '展示掩码（token 族前缀…后4位，如 sk-…2216），随指纹聚合带出';

-- 便于按 key 聚合查询（key 榜按 key_fingerprint 分组、按 usage_date 过滤）
create index if not exists idx_daily_keyfp_date on public.daily_account_usage (key_fingerprint, usage_date);
