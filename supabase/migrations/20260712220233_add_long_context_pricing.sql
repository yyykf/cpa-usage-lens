-- ============================================================
-- CPA Usage Lens — model-specific long-context pricing
--
-- Adds LiteLLM provider/threshold/high-tier prices and persists the selected
-- request tier in daily_account_usage. Existing daily rows intentionally stay
-- in the base bucket; historical request detail is not reconstructed.
--
-- Breaking change: daily_account_usage primary key grows from 4 to 5 columns.
-- Stop the old Lens binary before applying this migration.
-- ============================================================

alter table public.model_prices
    add column if not exists provider                                  text not null default '',
    add column if not exists long_context_threshold_tokens             bigint,
    add column if not exists long_context_input_cost_per_token         numeric(20, 12),
    add column if not exists long_context_output_cost_per_token        numeric(20, 12),
    add column if not exists long_context_cache_read_cost_per_token    numeric(20, 12),
    add column if not exists long_context_cache_creation_cost_per_token numeric(20, 12);

comment on column public.model_prices.provider is 'LiteLLM provider used to select provider-specific token accounting semantics';
comment on column public.model_prices.long_context_threshold_tokens is 'Strict input-token boundary: long tier applies only when input_tokens > threshold';

alter table public.daily_account_usage
    add column if not exists long_context boolean not null default false;

do $$
declare
  pk_cols text[];
begin
  select array_agg(a.attname order by a.attnum)
    into pk_cols
  from pg_constraint c
  join pg_attribute a
    on a.attrelid = c.conrelid
   and a.attnum = any(c.conkey)
  where c.conname = 'daily_account_usage_pkey'
    and c.conrelid = 'public.daily_account_usage'::regclass;

  if pk_cols is null
     or array(select unnest(pk_cols) order by 1)
        is distinct from array['key_fingerprint', 'long_context', 'model', 'source', 'usage_date'] then
    alter table public.daily_account_usage drop constraint if exists daily_account_usage_pkey;
    alter table public.daily_account_usage
      add constraint daily_account_usage_pkey
      primary key (usage_date, source, model, key_fingerprint, long_context);
  end if;
end $$;

comment on column public.daily_account_usage.long_context is 'True when the original request input_tokens strictly exceeded its model price threshold';
