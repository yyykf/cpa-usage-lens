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
}
