package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// RollupRange 把 [startDate, endDate]（含端点，YYYY-MM-DD）的 hot 明细按 tz 时区的"天"
// 聚合进 daily_account_usage（幂等覆盖这些天）。可安全重复调用以重算最近几天的延迟事件。
func (d *DB) RollupRange(ctx context.Context, startDate, endDate, tz string) error {
	tx, err := d.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 价格元数据变化可能让 retained hot 行在 base/long 两个桶之间移动；先删后重建
	// 才能移除已经不存在的旧桶。整个窗口仍在 hot retention 内，明细完整，事务保证原子替换。
	if _, err := tx.Exec(ctx, `
DELETE FROM daily_account_usage
WHERE usage_date >= $1::date AND usage_date <= $2::date`, startDate, endDate); err != nil {
		return err
	}

	// key 维度：按 coalesce(key_fingerprint,'none') 分组（hot 可空 → 兜底哨兵，对齐 daily 列默认与采集器）；
	// key_mask 同指纹下一致，用 max(...) 任取一个带出（coalesce 防 NULL 违反 daily NOT NULL）。
	_, err = tx.Exec(ctx, `
INSERT INTO daily_account_usage (
  usage_date, source, model, key_fingerprint, key_mask, long_context, requests, failed_requests,
  input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, total_tokens,
  uncached_input_tokens, canonical_cache_read_tokens, canonical_cache_creation_tokens,
  non_reasoning_output_tokens, canonical_reasoning_tokens, unclassified_tokens,
  complete_requests, unclassified_requests, inconsistent_requests, legacy_requests, updated_at
)
SELECT
  (event_ts AT TIME ZONE $3)::date AS usage_date,
  h.source, h.model,
  coalesce(h.key_fingerprint, 'none') AS key_fingerprint,
  coalesce(max(h.key_mask), '')       AS key_mask,
  coalesce(h.input_tokens > mp.long_context_threshold_tokens, false) AS long_context,
  count(*),
  count(*) FILTER (WHERE h.failed),
  sum(h.input_tokens), sum(h.output_tokens), sum(h.reasoning_tokens), sum(h.cached_tokens),
  sum(h.cache_read_tokens), sum(h.cache_creation_tokens), sum(h.total_tokens),
  sum(h.uncached_input_tokens), sum(h.canonical_cache_read_tokens), sum(h.canonical_cache_creation_tokens),
  sum(h.non_reasoning_output_tokens), sum(h.canonical_reasoning_tokens), sum(h.unclassified_tokens),
  count(*) filter (where h.accounting_quality='complete'),
  count(*) filter (where h.accounting_quality='unclassified'),
  count(*) filter (where h.accounting_quality='inconsistent'),
  count(*) filter (where h.accounting_version=1),
  now()
FROM request_events_hot h
LEFT JOIN model_prices mp ON mp.model = h.model
WHERE (h.event_ts AT TIME ZONE $3)::date >= $1::date
  AND (h.event_ts AT TIME ZONE $3)::date <= $2::date
GROUP BY 1, h.source, h.model, coalesce(h.key_fingerprint, 'none'), 6
ON CONFLICT (usage_date, source, model, key_fingerprint, long_context) DO UPDATE SET
  key_mask              = EXCLUDED.key_mask,
  requests              = EXCLUDED.requests,
  failed_requests       = EXCLUDED.failed_requests,
  input_tokens          = EXCLUDED.input_tokens,
  output_tokens         = EXCLUDED.output_tokens,
  reasoning_tokens      = EXCLUDED.reasoning_tokens,
  cached_tokens         = EXCLUDED.cached_tokens,
  cache_read_tokens     = EXCLUDED.cache_read_tokens,
  cache_creation_tokens = EXCLUDED.cache_creation_tokens,
  total_tokens          = EXCLUDED.total_tokens,
  uncached_input_tokens = EXCLUDED.uncached_input_tokens,
  canonical_cache_read_tokens = EXCLUDED.canonical_cache_read_tokens,
  canonical_cache_creation_tokens = EXCLUDED.canonical_cache_creation_tokens,
  non_reasoning_output_tokens = EXCLUDED.non_reasoning_output_tokens,
  canonical_reasoning_tokens = EXCLUDED.canonical_reasoning_tokens,
  unclassified_tokens = EXCLUDED.unclassified_tokens,
  complete_requests = EXCLUDED.complete_requests,
  unclassified_requests = EXCLUDED.unclassified_requests,
  inconsistent_requests = EXCLUDED.inconsistent_requests,
  legacy_requests = EXCLUDED.legacy_requests,
  updated_at            = now()`,
		startDate, endDate, tz)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteHotBefore 删除 event_ts 早于 beforeDate（按 tz 的天）的明细，返回删除行数。
// 调用方必须先确保这些天已 rollup（删除窗口 > 聚合重算窗口）。
func (d *DB) DeleteHotBefore(ctx context.Context, beforeDate, tz string) (int64, error) {
	ct, err := d.Pool.Exec(ctx,
		`DELETE FROM request_events_hot WHERE (event_ts AT TIME ZONE $2)::date < $1::date`,
		beforeDate, tz)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
