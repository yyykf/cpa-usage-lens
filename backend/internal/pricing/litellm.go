package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// LiteLLMURL 是业界标准的 LiteLLM 价格表数据源。
const LiteLLMURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// litellmEntry 是 LiteLLM JSON 里单个模型的价格字段（只取我们要的几项）。
type litellmEntry struct {
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
	var raw map[string]litellmEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(wanted))
	for _, w := range wanted {
		want[w] = true
	}

	out := make([]model.ModelPrice, 0, len(want))
	for name, e := range raw {
		if len(wanted) > 0 && !want[name] {
			continue
		}
		if e.InputCostPerToken == nil && e.OutputCostPerToken == nil {
			continue // 非模型元数据键
		}
		out = append(out, model.ModelPrice{
			Model:                     name,
			InputCostPerToken:         e.InputCostPerToken,
			OutputCostPerToken:        e.OutputCostPerToken,
			CacheReadCostPerToken:     e.CacheReadInputTokenCost,
			CacheCreationCostPerToken: e.CacheCreationInputTokenCost,
			Currency:                  "USD",
			Source:                    "litellm",
		})
	}
	return out, nil
}
