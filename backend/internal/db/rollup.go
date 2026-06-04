package db

import "context"

// RollupRange 把 [startDate, endDate]（含端点，YYYY-MM-DD）的 hot 明细按 tz 时区的"天"
// 聚合进 daily_account_usage（幂等覆盖这些天）。可安全重复调用以重算最近几天的延迟事件。
func (d *DB) RollupRange(ctx context.Context, startDate, endDate, tz string) error {
	// key 维度：按 coalesce(key_fingerprint,'none') 分组（hot 可空 → 兜底哨兵，对齐 daily 列默认与采集器）；
	// key_mask 同指纹下一致，用 max(...) 任取一个带出（coalesce 防 NULL 违反 daily NOT NULL）。
	_, err := d.Pool.Exec(ctx, `
INSERT INTO daily_account_usage (
  usage_date, source, model, key_fingerprint, key_mask, requests, failed_requests,
  input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, total_tokens, updated_at
)
SELECT
  (event_ts AT TIME ZONE $3)::date AS usage_date,
  source, model,
  coalesce(key_fingerprint, 'none') AS key_fingerprint,
  coalesce(max(key_mask), '')       AS key_mask,
  count(*),
  count(*) FILTER (WHERE failed),
  sum(input_tokens), sum(output_tokens), sum(reasoning_tokens), sum(cached_tokens),
  sum(cache_read_tokens), sum(cache_creation_tokens), sum(total_tokens),
  now()
FROM request_events_hot
WHERE (event_ts AT TIME ZONE $3)::date >= $1::date
  AND (event_ts AT TIME ZONE $3)::date <= $2::date
GROUP BY 1, source, model, coalesce(key_fingerprint, 'none')
ON CONFLICT (usage_date, source, model, key_fingerprint) DO UPDATE SET
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
  updated_at            = now()`,
		startDate, endDate, tz)
	return err
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
