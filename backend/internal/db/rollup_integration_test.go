//go:build integration

package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

const longContextMigration = "20260712220233_add_long_context_pricing.sql"
const accountingV2Migration = "20260726000000_add_token_accounting_v2.sql"

func TestLongContextMigrationAndRollup(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	database, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	if _, err := database.Pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"20260530185206_init_schema.sql",
		"20260605002633_add_api_key_dimension.sql",
		"20260608155535_request_events_hot_composite_pk.sql",
	} {
		execMigration(t, ctx, database, name)
	}

	// 存量 daily 行先存在，再跑新迁移；它必须平滑落入 base tier。
	_, err = database.Pool.Exec(ctx, `
INSERT INTO daily_account_usage (usage_date, source, model, key_fingerprint, key_mask, requests)
VALUES ('2026-07-11', 'legacy', 'gpt-5.6-sol', 'none', '(no key)', 1)`)
	if err != nil {
		t.Fatal(err)
	}
	execMigration(t, ctx, database, longContextMigration)
	execMigration(t, ctx, database, longContextMigration) // idempotency gate
	execMigration(t, ctx, database, accountingV2Migration)
	execMigration(t, ctx, database, accountingV2Migration) // idempotency gate
	var legacyLong bool
	if err := database.Pool.QueryRow(ctx, `
SELECT long_context FROM daily_account_usage
WHERE usage_date='2026-07-11' AND source='legacy'`).Scan(&legacyLong); err != nil {
		t.Fatal(err)
	}
	if legacyLong {
		t.Fatal("legacy daily row must migrate to long_context=false")
	}

	threshold := int64(272000)
	baseInput, baseOutput := 5e-6, 30e-6
	baseCacheRead, baseCacheCreation := 0.5e-6, 6.25e-6
	longInput, longOutput := 10e-6, 45e-6
	longCacheRead, longCacheCreation := 1e-6, 12.5e-6
	if err := database.UpsertPrices(ctx, []model.ModelPrice{{
		Model: "roundtrip", Provider: "openai",
		InputCostPerToken:                    &baseInput,
		OutputCostPerToken:                   &baseOutput,
		CacheReadCostPerToken:                &baseCacheRead,
		CacheCreationCostPerToken:            &baseCacheCreation,
		LongContextThresholdTokens:           &threshold,
		LongContextInputCostPerToken:         &longInput,
		LongContextOutputCostPerToken:        &longOutput,
		LongContextCacheReadCostPerToken:     &longCacheRead,
		LongContextCacheCreationCostPerToken: &longCacheCreation,
		Currency:                             "USD",
		Source:                               "integration-test",
	}}); err != nil {
		t.Fatal(err)
	}
	priceMap, err := database.GetPriceMap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	gotPrice := priceMap["roundtrip"]
	if gotPrice.Provider != "openai" || gotPrice.Currency != "USD" || gotPrice.Source != "integration-test" {
		t.Errorf("model price roundtrip lost metadata: %+v", gotPrice)
	}
	if gotPrice.LongContextThresholdTokens == nil || *gotPrice.LongContextThresholdTokens != threshold {
		t.Errorf("threshold = %v, want %d", gotPrice.LongContextThresholdTokens, threshold)
	}
	assertNullableFloat(t, "base input", gotPrice.InputCostPerToken, baseInput)
	assertNullableFloat(t, "base output", gotPrice.OutputCostPerToken, baseOutput)
	assertNullableFloat(t, "base cache read", gotPrice.CacheReadCostPerToken, baseCacheRead)
	assertNullableFloat(t, "base cache creation", gotPrice.CacheCreationCostPerToken, baseCacheCreation)
	assertNullableFloat(t, "long input", gotPrice.LongContextInputCostPerToken, longInput)
	assertNullableFloat(t, "long output", gotPrice.LongContextOutputCostPerToken, longOutput)
	assertNullableFloat(t, "long cache read", gotPrice.LongContextCacheReadCostPerToken, longCacheRead)
	assertNullableFloat(t, "long cache creation", gotPrice.LongContextCacheCreationCostPerToken, longCacheCreation)

	_, err = database.Pool.Exec(ctx, `
INSERT INTO model_prices (
  model, provider, input_cost_per_token, output_cost_per_token,
  long_context_threshold_tokens, long_context_input_cost_per_token, long_context_output_cost_per_token
) VALUES ('gpt-5.6-sol', 'openai', 0.000005, 0.000030, 272000, 0.000010, 0.000045);

INSERT INTO request_events_hot (
  request_id, event_ts, source, provider, model, key_fingerprint, key_mask,
  input_tokens, output_tokens, total_tokens,
  accounting_version, accounting_quality, uncached_input_tokens, non_reasoning_output_tokens
) VALUES
  ('base', '2026-07-12T01:00:00Z', 'user', 'codex', 'gpt-5.6-sol', 'none', '(no key)', 272000, 10, 272010, 2, 'complete', 272000, 10),
  ('long', '2026-07-12T02:00:00Z', 'user', 'codex', 'gpt-5.6-sol', 'none', '(no key)', 272001, 10, 272011, 2, 'complete', 272001, 10)`)
	if err != nil {
		t.Fatal(err)
	}

	if err := database.RollupRange(ctx, "2026-07-12", "2026-07-12", "UTC"); err != nil {
		t.Fatal(err)
	}
	rows, err := database.QueryDailyUsage(ctx, "2026-07-12", "2026-07-13")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("mixed threshold rows = %d, want 2: %+v", len(rows), rows)
	}
	if rows[0].LongContext || !rows[1].LongContext {
		t.Fatalf("strict threshold classification wrong: %+v", rows)
	}
	if rows[0].Accounting.CompleteRequests != 1 || rows[0].Accounting.Tokens.UncachedInput != 272000 {
		t.Fatalf("accounting v2 rollup lost canonical fields: %+v", rows[0].Accounting)
	}

	// 元数据变化后重跑：旧 long=true 行必须被删除，而不是与新 base 行并存造成双算。
	if _, err := database.Pool.Exec(ctx, `
UPDATE model_prices SET long_context_threshold_tokens=400000 WHERE model='gpt-5.6-sol'`); err != nil {
		t.Fatal(err)
	}
	if err := database.RollupRange(ctx, "2026-07-12", "2026-07-12", "UTC"); err != nil {
		t.Fatal(err)
	}
	rows, err = database.QueryDailyUsage(ctx, "2026-07-12", "2026-07-13")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].LongContext || rows[0].Requests != 2 {
		t.Fatalf("reclassified rows = %+v, want one base row with 2 requests", rows)
	}
}

func assertNullableFloat(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func execMigration(t *testing.T, ctx context.Context, database *DB, name string) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "supabase", "migrations", name)
	sql, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	if _, err := database.Pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("execute migration %s: %v", name, err)
	}
}
