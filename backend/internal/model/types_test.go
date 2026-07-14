package model

import (
	"encoding/json"
	"testing"
)

func TestTokenBreakdownDTOsMarshalFlat(t *testing.T) {
	breakdown := TokenBreakdown{
		InputTokens:              1,
		UncachedInputTokens:      2,
		OutputTokens:             3,
		ReasoningTokens:          4,
		CachedTokens:             5,
		CacheReadTokens:          6,
		CanonicalCacheReadTokens: 7,
		CacheCreationTokens:      8,
	}
	breakdownJSON := map[string]any{
		"inputTokens":              1.0,
		"uncachedInputTokens":      2.0,
		"outputTokens":             3.0,
		"reasoningTokens":          4.0,
		"cachedTokens":             5.0,
		"cacheReadTokens":          6.0,
		"canonicalCacheReadTokens": 7.0,
		"cacheCreationTokens":      8.0,
	}
	tests := map[string]struct {
		dto  any
		want map[string]any
	}{
		"overview": {
			dto: Overview{Requests: 9, Tokens: 10, Failed: 11, TokenBreakdown: breakdown, HasPrevious: true},
			want: mergeJSONFields(breakdownJSON, map[string]any{
				"requests": 9.0, "tokens": 10.0, "cost": nil, "failed": 11.0,
				"hasPrevious": true, "previous": nil,
			}),
		},
		"account": {
			dto: AccountUsage{Source: "account", Requests: 9, Tokens: 10, Failed: 11, TokenBreakdown: breakdown},
			want: mergeJSONFields(breakdownJSON, map[string]any{
				"source": "account", "requests": 9.0, "tokens": 10.0, "cost": nil, "failed": 11.0,
			}),
		},
		"key": {
			dto: KeyUsage{Fingerprint: "fingerprint", KeyMask: "mask", Requests: 9, Tokens: 10, Failed: 11, TokenBreakdown: breakdown},
			want: mergeJSONFields(breakdownJSON, map[string]any{
				"fingerprint": "fingerprint", "keyMask": "mask", "requests": 9.0,
				"tokens": 10.0, "cost": nil, "failed": 11.0,
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(test.dto)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("JSON fields = %v, want exactly %v", got, test.want)
			}
			for field, value := range test.want {
				if got[field] != value {
					t.Errorf("%s = %v, want %v; JSON=%s", field, got[field], value, data)
				}
			}
		})
	}
}

func TestTokenBreakdownDTOZeroValuesRemainVisible(t *testing.T) {
	for name, dto := range map[string]any{
		"overview": Overview{},
		"account":  AccountUsage{},
		"key":      KeyUsage{},
	} {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{
				"inputTokens", "uncachedInputTokens", "outputTokens", "reasoningTokens",
				"cachedTokens", "cacheReadTokens", "canonicalCacheReadTokens", "cacheCreationTokens",
			} {
				if value, ok := got[field]; !ok || value != float64(0) {
					t.Errorf("zero field %s = %v (present=%v); JSON=%s", field, value, ok, data)
				}
			}
		})
	}
}

func mergeJSONFields(sets ...map[string]any) map[string]any {
	merged := make(map[string]any)
	for _, fields := range sets {
		for key, value := range fields {
			merged[key] = value
		}
	}
	return merged
}
