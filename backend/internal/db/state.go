package db

import (
	"context"
	"errors"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

// BumpCollectorState upsert 采集器状态（单行 id=1）；events_ingested 为"本次新增"，累加到现有值。
func (d *DB) BumpCollectorState(ctx context.Context, s model.CollectorState) error {
	_, err := d.Pool.Exec(ctx, `
INSERT INTO collector_state (id, last_poll_at, last_event_ts, last_request_id, events_ingested, last_error, last_error_at, updated_at)
VALUES (1, $1, $2, $3, $4, $5, $6, now())
ON CONFLICT (id) DO UPDATE SET
  last_poll_at    = EXCLUDED.last_poll_at,
  last_event_ts   = COALESCE(EXCLUDED.last_event_ts, collector_state.last_event_ts),
  last_request_id = COALESCE(NULLIF(EXCLUDED.last_request_id, ''), collector_state.last_request_id),
  events_ingested = collector_state.events_ingested + EXCLUDED.events_ingested,
  last_error      = EXCLUDED.last_error,
  last_error_at   = EXCLUDED.last_error_at,
  updated_at      = now()`,
		s.LastPollAt, s.LastEventTS, s.LastRequestID, s.EventsIngested, s.LastError, s.LastErrorAt)
	return err
}

// GetCollectorState 读取采集器状态；无行时返回零值 + ok=false。
func (d *DB) GetCollectorState(ctx context.Context) (model.CollectorState, bool, error) {
	var s model.CollectorState
	err := d.Pool.QueryRow(ctx, `
SELECT last_poll_at, last_event_ts, last_request_id, events_ingested, last_error, last_error_at, updated_at
FROM collector_state WHERE id = 1`).Scan(
		&s.LastPollAt, &s.LastEventTS, &s.LastRequestID, &s.EventsIngested, &s.LastError, &s.LastErrorAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, false, nil
	}
	if err != nil {
		return s, false, err
	}
	return s, true, nil
}
