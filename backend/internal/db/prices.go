package db

import (
	"context"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
	"github.com/jackc/pgx/v5"
)

// UpsertPrices 批量 upsert 模型价格（按 model 覆盖）。
func (d *DB) UpsertPrices(ctx context.Context, prices []model.ModelPrice) error {
	if len(prices) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, p := range prices {
		batch.Queue(`
INSERT INTO model_prices (model, input_cost_per_token, output_cost_per_token, cache_read_cost_per_token, cache_creation_cost_per_token, currency, source, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7, now())
ON CONFLICT (model) DO UPDATE SET
  input_cost_per_token          = EXCLUDED.input_cost_per_token,
  output_cost_per_token         = EXCLUDED.output_cost_per_token,
  cache_read_cost_per_token     = EXCLUDED.cache_read_cost_per_token,
  cache_creation_cost_per_token = EXCLUDED.cache_creation_cost_per_token,
  currency                      = EXCLUDED.currency,
  source                        = EXCLUDED.source,
  updated_at                    = now()`,
			p.Model, p.InputCostPerToken, p.OutputCostPerToken, p.CacheReadCostPerToken, p.CacheCreationCostPerToken,
			defaultStr(p.Currency, "USD"), defaultStr(p.Source, "litellm"))
	}
	br := d.Pool.SendBatch(ctx, batch)
	defer br.Close()
	for range prices {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetPriceMap 读取所有价格 → map[model]ModelPrice（numeric cast 成 float8 便于 scan）。
func (d *DB) GetPriceMap(ctx context.Context) (map[string]model.ModelPrice, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT model,
       input_cost_per_token::float8, output_cost_per_token::float8,
       cache_read_cost_per_token::float8, cache_creation_cost_per_token::float8,
       currency, source, updated_at
FROM model_prices`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]model.ModelPrice)
	for rows.Next() {
		var p model.ModelPrice
		if err := rows.Scan(&p.Model, &p.InputCostPerToken, &p.OutputCostPerToken,
			&p.CacheReadCostPerToken, &p.CacheCreationCostPerToken, &p.Currency, &p.Source, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out[p.Model] = p
	}
	return out, rows.Err()
}

// ListUsedModels 返回出现过的模型（hot ∪ daily 去重），用于"只刷用过的模型"价格。
func (d *DB) ListUsedModels(ctx context.Context) ([]string, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT DISTINCT model FROM (
  SELECT model FROM request_events_hot
  UNION
  SELECT model FROM daily_account_usage
) m WHERE model <> '' ORDER BY model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
