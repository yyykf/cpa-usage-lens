package db

import (
	"context"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// QueryDailyUsage 取 [startDate, endDate) 区间按 (账号,模型,天) 的聚合行。
// startDate/endDate 为 YYYY-MM-DD（按配置时区界定的天边界，end 半开）。
func (d *DB) QueryDailyUsage(ctx context.Context, startDate, endDate string) ([]model.DailyUsage, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT usage_date, source, model, requests, failed_requests,
       input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, total_tokens
FROM daily_account_usage
WHERE usage_date >= $1::date AND usage_date < $2::date
ORDER BY usage_date, source, model`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.DailyUsage
	for rows.Next() {
		var u model.DailyUsage
		if err := rows.Scan(&u.UsageDate, &u.Source, &u.Model, &u.Requests, &u.FailedRequests,
			&u.Tokens.Input, &u.Tokens.Output, &u.Tokens.Reasoning, &u.Tokens.Cached,
			&u.Tokens.CacheRead, &u.Tokens.CacheCreation, &u.Tokens.Total); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// Capacity 返回明细表与聚合表的真实字节大小（Postgres total_relation_size，绝对值不算百分比）。
func (d *DB) Capacity(ctx context.Context) (hotBytes, dailyBytes int64, err error) {
	err = d.Pool.QueryRow(ctx, `
SELECT pg_total_relation_size('public.request_events_hot'),
       pg_total_relation_size('public.daily_account_usage')`).Scan(&hotBytes, &dailyBytes)
	return hotBytes, dailyBytes, err
}
