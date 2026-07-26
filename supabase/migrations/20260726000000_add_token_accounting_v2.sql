-- CPA v7.2.97 canonical token accounting and quality coverage.
alter table public.request_events_hot
    add column if not exists accounting_version smallint not null default 1,
    add column if not exists accounting_quality text not null default 'complete',
    add column if not exists uncached_input_tokens bigint not null default 0,
    add column if not exists canonical_cache_read_tokens bigint not null default 0,
    add column if not exists canonical_cache_creation_tokens bigint not null default 0,
    add column if not exists non_reasoning_output_tokens bigint not null default 0,
    add column if not exists canonical_reasoning_tokens bigint not null default 0,
    add column if not exists unclassified_tokens bigint not null default 0;

update public.request_events_hot h
set uncached_input_tokens = case
        when lower(coalesce(h.provider, '')) in ('codex', 'openai')
          then greatest(h.input_tokens - greatest(h.cached_tokens, h.cache_read_tokens) - h.cache_creation_tokens, 0)
        else h.input_tokens end,
    canonical_cache_read_tokens = case
        when lower(coalesce(h.provider, '')) in ('codex', 'openai')
          then greatest(h.cached_tokens, h.cache_read_tokens)
        else h.cache_read_tokens end,
    canonical_cache_creation_tokens = h.cache_creation_tokens,
    non_reasoning_output_tokens = case
        when lower(coalesce(h.provider, '')) in ('codex', 'openai')
          then greatest(h.output_tokens - h.reasoning_tokens, 0)
        else h.output_tokens end,
    canonical_reasoning_tokens = h.reasoning_tokens
where h.accounting_version = 1;

alter table public.request_events_hot
    drop constraint if exists request_events_hot_accounting_quality_check;
alter table public.request_events_hot
    add constraint request_events_hot_accounting_quality_check
    check (accounting_quality in ('complete', 'unclassified', 'inconsistent'));

alter table public.daily_account_usage
    add column if not exists uncached_input_tokens bigint not null default 0,
    add column if not exists canonical_cache_read_tokens bigint not null default 0,
    add column if not exists canonical_cache_creation_tokens bigint not null default 0,
    add column if not exists non_reasoning_output_tokens bigint not null default 0,
    add column if not exists canonical_reasoning_tokens bigint not null default 0,
    add column if not exists unclassified_tokens bigint not null default 0,
    add column if not exists complete_requests bigint not null default 0,
    add column if not exists unclassified_requests bigint not null default 0,
    add column if not exists inconsistent_requests bigint not null default 0,
    add column if not exists legacy_requests bigint not null default 0;

update public.daily_account_usage d
set uncached_input_tokens = case
        when lower(coalesce(mp.provider, '')) = 'openai'
          then greatest(d.input_tokens - greatest(d.cached_tokens, d.cache_read_tokens) - d.cache_creation_tokens, 0)
        else d.input_tokens end,
    canonical_cache_read_tokens = case
        when lower(coalesce(mp.provider, '')) = 'openai'
          then greatest(d.cached_tokens, d.cache_read_tokens)
        else d.cache_read_tokens end,
    canonical_cache_creation_tokens = d.cache_creation_tokens,
    non_reasoning_output_tokens = case
        when lower(coalesce(mp.provider, '')) = 'openai'
          then greatest(d.output_tokens - d.reasoning_tokens, 0)
        else d.output_tokens end,
    canonical_reasoning_tokens = d.reasoning_tokens,
    complete_requests = d.requests,
    legacy_requests = d.requests
from public.model_prices mp
where mp.model = d.model
  and d.complete_requests = 0 and d.unclassified_requests = 0 and d.inconsistent_requests = 0;

-- Models without price metadata retain the legacy independent shape.
update public.daily_account_usage d
set uncached_input_tokens = d.input_tokens,
    canonical_cache_read_tokens = d.cache_read_tokens,
    canonical_cache_creation_tokens = d.cache_creation_tokens,
    non_reasoning_output_tokens = d.output_tokens,
    canonical_reasoning_tokens = d.reasoning_tokens,
    complete_requests = d.requests,
    legacy_requests = d.requests
where d.complete_requests = 0 and d.unclassified_requests = 0 and d.inconsistent_requests = 0;
