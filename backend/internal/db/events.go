package db

import (
	"context"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

// InsertEvents 批量幂等插入明细（request_id 冲突则跳过）。返回真正新插入的条数。
func (d *DB) InsertEvents(ctx context.Context, events []model.UsageEvent) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}
	batch := &pgx.Batch{}
	for _, e := range events {
		batch.Queue(`
INSERT INTO request_events_hot (
  request_id, event_ts, source, auth_index, provider, model, alias, endpoint, auth_type,
  key_fingerprint, key_mask,
  input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, total_tokens,
  latency_ms, ttft_ms, failed, fail_status_code, reasoning_effort, service_tier
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)
ON CONFLICT (request_id) DO NOTHING`,
			e.RequestID, e.EventTS, e.Source, e.AuthIndex, e.Provider, e.Model, e.Alias, e.Endpoint, e.AuthType,
			e.KeyFingerprint, e.KeyMask,
			e.Tokens.Input, e.Tokens.Output, e.Tokens.Reasoning, e.Tokens.Cached, e.Tokens.CacheRead, e.Tokens.CacheCreation, e.Tokens.Total,
			e.LatencyMs, e.TTFTMs, e.Failed, e.FailStatusCode, e.ReasoningEffort, e.ServiceTier)
	}
	br := d.Pool.SendBatch(ctx, batch)
	defer br.Close()

	var inserted int64
	for range events {
		ct, err := br.Exec()
		if err != nil {
			return inserted, err
		}
		inserted += ct.RowsAffected()
	}
	return inserted, nil
}
