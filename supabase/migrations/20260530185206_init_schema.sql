-- ============================================================
-- CPA Usage Lens — 初始 schema
-- 4 张表：热明细 / 日聚合 / 价格表 / 采集器状态
--
-- 安全约定（对应 PRD D1）：
--   全部表 ENABLE RLS 且【不建任何 anon/authenticated 策略】。
--   backend 用 Supabase 连接串以 postgres 角色直连（bypassrls），可正常读写；
--   前端永不直连 DB，因此 Data API 即便被误暴露也默认拒绝。
-- ============================================================

-- ------------------------------------------------------------
-- 1) request_events_hot：热明细，每请求一行，短期保留（默认 7 天，backend 可配）
--    去重：以 CPA 提供的 request_id 作主键 → 采集器 INSERT ... ON CONFLICT DO NOTHING 幂等
--    不入库（剥离）：api_key / response_headers / fail.body —— 敏感且大
-- ------------------------------------------------------------
create table if not exists public.request_events_hot (
    request_id              text        primary key,            -- 幂等去重核心
    event_ts                timestamptz not null,               -- 队列 timestamp（带时区偏移，存 UTC），用于"天"归属与清理
    source                  text        not null,               -- 账号（邮箱）
    auth_index              text,
    provider                text,
    model                   text        not null,
    alias                   text,
    endpoint                text,
    auth_type               text,
    -- token 明细（v7.1.31 真实拆分，便于按读/写不同单价精确算成本）
    input_tokens            bigint      not null default 0,
    output_tokens           bigint      not null default 0,
    reasoning_tokens        bigint      not null default 0,
    cached_tokens           bigint      not null default 0,
    cache_read_tokens       bigint      not null default 0,
    cache_creation_tokens   bigint      not null default 0,
    total_tokens            bigint      not null default 0,
    -- 性能 / 失败状态
    latency_ms              integer,
    ttft_ms                 integer,                            -- 首字节延迟（可选性能指标）
    failed                  boolean     not null default false,
    fail_status_code        integer,                            -- v7.1.31 真实存在；fail.body 不存
    reasoning_effort        text,
    service_tier            text,
    ingested_at             timestamptz not null default now()  -- 采集写入时间（排查用）
);
comment on table public.request_events_hot is '热明细：每请求一行，短期保留（默认7天，可配），按 request_id 去重；不存 api_key/response_headers/fail.body';

create index if not exists idx_hot_event_ts   on public.request_events_hot (event_ts);
create index if not exists idx_hot_source_ts  on public.request_events_hot (source, event_ts);
create index if not exists idx_hot_model_ts   on public.request_events_hot (model,  event_ts);

-- ------------------------------------------------------------
-- 2) daily_account_usage：按 账号+模型+天 聚合，长期保留
--    幂等 rollup：以 (usage_date, source, model) 作主键 → INSERT ... ON CONFLICT DO UPDATE
--    不存 cost：查询时按 model_prices 当前单价实时算（价格变了无需回填）
-- ------------------------------------------------------------
create table if not exists public.daily_account_usage (
    usage_date              date        not null,               -- 按可配时区界定的"天"
    source                  text        not null,
    model                   text        not null,
    requests                bigint      not null default 0,
    failed_requests         bigint      not null default 0,
    input_tokens            bigint      not null default 0,
    output_tokens           bigint      not null default 0,
    reasoning_tokens        bigint      not null default 0,
    cached_tokens           bigint      not null default 0,
    cache_read_tokens       bigint      not null default 0,
    cache_creation_tokens   bigint      not null default 0,
    total_tokens            bigint      not null default 0,
    updated_at              timestamptz not null default now(),
    primary key (usage_date, source, model)
);
comment on table public.daily_account_usage is '日聚合：账号+模型+天，长期保留；不存 cost，查询时用 model_prices 实时算';

create index if not exists idx_daily_date         on public.daily_account_usage (usage_date);
create index if not exists idx_daily_source_date  on public.daily_account_usage (source, usage_date);

-- ------------------------------------------------------------
-- 3) model_prices：价格表，只存"用过"的模型；LiteLLM 每日 + 手动刷新
--    单价为每 token 的 USD（LiteLLM 原始口径）；缺价 = 无行（查询时标"未知"）
-- ------------------------------------------------------------
create table if not exists public.model_prices (
    model                           text        primary key,
    input_cost_per_token            numeric(20, 12),
    output_cost_per_token           numeric(20, 12),
    cache_read_cost_per_token       numeric(20, 12),
    cache_creation_cost_per_token   numeric(20, 12),
    currency                        text        not null default 'USD',
    source                          text        not null default 'litellm',
    updated_at                      timestamptz not null default now()
);
comment on table public.model_prices is 'LiteLLM 价格（每 token USD），只存用过的模型；缺价标未知（无行）';

-- ------------------------------------------------------------
-- 4) collector_state：采集器游标 + 健康（延迟/最后错误），单行（id=1）
--    健康模块读取；采集器每轮更新
-- ------------------------------------------------------------
create table if not exists public.collector_state (
    id                      smallint    primary key default 1,
    last_poll_at            timestamptz,
    last_event_ts           timestamptz,
    last_request_id         text,
    events_ingested         bigint      not null default 0,
    last_error              text,
    last_error_at           timestamptz,
    updated_at              timestamptz not null default now(),
    constraint collector_state_singleton check (id = 1)
);
comment on table public.collector_state is '采集器游标与健康，单行（id=1）';

-- ------------------------------------------------------------
-- 安全：开 RLS，不建任何策略 → Data API 默认拒绝；backend 以 postgres 角色直连（bypassrls）正常读写
-- ------------------------------------------------------------
alter table public.request_events_hot   enable row level security;
alter table public.daily_account_usage  enable row level security;
alter table public.model_prices         enable row level security;
alter table public.collector_state      enable row level security;
