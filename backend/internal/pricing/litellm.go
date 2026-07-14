package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// LiteLLMURL 是业界标准的 LiteLLM 价格表数据源。
const LiteLLMURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// litellmEntry 是 LiteLLM JSON 里单个模型的价格字段（只取我们要的几项）。
type litellmEntry struct {
	Provider                    string   `json:"litellm_provider"`
	InputCostPerToken           *float64 `json:"input_cost_per_token"`
	OutputCostPerToken          *float64 `json:"output_cost_per_token"`
	CacheReadInputTokenCost     *float64 `json:"cache_read_input_token_cost"`
	CacheCreationInputTokenCost *float64 `json:"cache_creation_input_token_cost"`
}

// FetchPrices 拉取 LiteLLM 价格表，只返回 wanted 集合里的模型（wanted 为空则全部）。
func FetchPrices(ctx context.Context, client *http.Client, url string, wanted []string) ([]model.ModelPrice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("litellm 价格表 HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ParsePrices(body, wanted)
}

// ParsePrices 从 LiteLLM JSON 解析出 wanted 模型的价格；跳过无价的元数据键（如 sample_spec）。
func ParsePrices(data []byte, wanted []string) ([]model.ModelPrice, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(wanted))
	for _, w := range wanted {
		want[w] = true
	}

	out := make([]model.ModelPrice, 0, len(want))
	for name, entryJSON := range raw {
		if len(wanted) > 0 && !want[name] {
			continue
		}
		var e litellmEntry
		if err := json.Unmarshal(entryJSON, &e); err != nil {
			return nil, err
		}
		if e.InputCostPerToken == nil && e.OutputCostPerToken == nil {
			continue // 非模型元数据键
		}
		price := model.ModelPrice{
			Model:                     name,
			Provider:                  e.Provider,
			InputCostPerToken:         e.InputCostPerToken,
			OutputCostPerToken:        e.OutputCostPerToken,
			CacheReadCostPerToken:     e.CacheReadInputTokenCost,
			CacheCreationCostPerToken: e.CacheCreationInputTokenCost,
			Currency:                  "USD",
			Source:                    "litellm",
		}
		mapLongContextTier(entryJSON, &price)
		out = append(out, price)
	}
	return out, nil
}

// mapLongContextTier 解析 LiteLLM 的动态 above_<threshold>_tokens 字段。
// 当前数据模型只支持一个 threshold；遇到多个 threshold 时保持为空，避免把多档价格误压成一档。
func mapLongContextTier(data []byte, price *model.ModelPrice) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return
	}

	const (
		prefix = "input_cost_per_token_above_"
		suffix = "_tokens"
	)
	labels := make([]string, 0, 1)
	for key := range fields {
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			labels = append(labels, strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix))
		}
	}
	if len(labels) != 1 {
		return
	}

	threshold, ok := parseTokenThreshold(labels[0])
	if !ok {
		return
	}
	price.LongContextThresholdTokens = &threshold
	tierSuffix := "_above_" + labels[0] + "_tokens"
	price.LongContextInputCostPerToken = floatField(fields, "input_cost_per_token"+tierSuffix)
	price.LongContextOutputCostPerToken = floatField(fields, "output_cost_per_token"+tierSuffix)
	price.LongContextCacheReadCostPerToken = floatField(fields, "cache_read_input_token_cost"+tierSuffix)
	price.LongContextCacheCreationCostPerToken = floatField(fields, "cache_creation_input_token_cost"+tierSuffix)
}

func parseTokenThreshold(label string) (int64, bool) {
	multiplier := int64(1)
	if strings.HasSuffix(label, "k") {
		multiplier = 1000
		label = strings.TrimSuffix(label, "k")
	}
	n, err := strconv.ParseInt(label, 10, 64)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n * multiplier, true
}

func floatField(fields map[string]json.RawMessage, key string) *float64 {
	raw, ok := fields[key]
	if !ok {
		return nil
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return &value
}
