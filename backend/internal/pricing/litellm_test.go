package pricing

import "testing"

func TestParsePrices_FiltersWantedAndSkipsMeta(t *testing.T) {
	data := []byte(`{
		"sample_spec": {"notes": "metadata, no prices"},
		"gpt-5.4": {"input_cost_per_token": 0.000001, "output_cost_per_token": 0.000002, "cache_read_input_token_cost": 0.0000001},
		"claude-x": {"input_cost_per_token": 0.000003, "output_cost_per_token": 0.000006},
		"unused-model": {"input_cost_per_token": 0.1, "output_cost_per_token": 0.2}
	}`)
	out, err := ParsePrices(data, []string{"gpt-5.4", "claude-x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 wanted models, got %d", len(out))
	}
	got := map[string]bool{}
	for _, p := range out {
		got[p.Model] = true
	}
	if !got["gpt-5.4"] || !got["claude-x"] {
		t.Errorf("wanted models missing: %+v", out)
	}
	if got["unused-model"] || got["sample_spec"] {
		t.Error("should not include unwanted or metadata keys")
	}
}

func TestParsePrices_CacheReadMapped(t *testing.T) {
	data := []byte(`{"m": {"input_cost_per_token": 1e-6, "output_cost_per_token": 2e-6, "cache_read_input_token_cost": 1e-7}}`)
	out, _ := ParsePrices(data, nil)
	if len(out) != 1 {
		t.Fatalf("got %d", len(out))
	}
	if out[0].CacheReadCostPerToken == nil || *out[0].CacheReadCostPerToken != 1e-7 {
		t.Error("cache_read cost not mapped")
	}
	if out[0].LongContextThresholdTokens != nil {
		t.Errorf("model without threshold should stay base-only: %+v", out[0])
	}
}

func TestParsePrices_MultipleThresholdsDoNotCollapseIntoBooleanTier(t *testing.T) {
	data := []byte(`{"m": {
		"input_cost_per_token": 1e-6,
		"output_cost_per_token": 2e-6,
		"input_cost_per_token_above_128k_tokens": 2e-6,
		"input_cost_per_token_above_272k_tokens": 3e-6
	}}`)
	out, err := ParsePrices(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].LongContextThresholdTokens != nil {
		t.Fatalf("multi-threshold model must remain base-only in boolean MVP: %+v", out)
	}
}

func TestParsePrices_MapsProviderAndLongContextTier(t *testing.T) {
	data := []byte(`{
		"gpt-5.6-sol": {
			"litellm_provider": "openai",
			"input_cost_per_token": 0.000005,
			"output_cost_per_token": 0.000030,
			"cache_read_input_token_cost": 0.0000005,
			"cache_creation_input_token_cost": 0.00000625,
			"input_cost_per_token_above_272k_tokens": 0.000010,
			"output_cost_per_token_above_272k_tokens": 0.000045,
			"cache_read_input_token_cost_above_272k_tokens": 0.000001,
			"cache_creation_input_token_cost_above_272k_tokens": 0.0000125
		}
	}`)
	out, err := ParsePrices(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d prices, want 1", len(out))
	}
	p := out[0]
	if p.Provider != "openai" {
		t.Errorf("provider = %q, want openai", p.Provider)
	}
	if p.LongContextThresholdTokens == nil || *p.LongContextThresholdTokens != 272000 {
		t.Errorf("threshold = %v, want 272000", p.LongContextThresholdTokens)
	}
	assertPrice := func(name string, got *float64, want float64) {
		t.Helper()
		if got == nil || *got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	assertPrice("long input", p.LongContextInputCostPerToken, 0.000010)
	assertPrice("long output", p.LongContextOutputCostPerToken, 0.000045)
	assertPrice("long cache read", p.LongContextCacheReadCostPerToken, 0.000001)
	assertPrice("long cache creation", p.LongContextCacheCreationCostPerToken, 0.0000125)
}
