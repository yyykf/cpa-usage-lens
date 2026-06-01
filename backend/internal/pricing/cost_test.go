package pricing

import (
	"math"
	"testing"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

func fp(v float64) *float64 { return &v }

func TestCost_FullPrice(t *testing.T) {
	p := model.ModelPrice{InputCostPerToken: fp(1e-6), OutputCostPerToken: fp(2e-6)}
	c, ok := Cost(model.Tokens{Input: 1000, Output: 500}, p)
	if !ok {
		t.Fatal("expected known cost")
	}
	want := 1000*1e-6 + 500*2e-6
	if math.Abs(c-want) > 1e-15 {
		t.Errorf("cost = %v, want %v", c, want)
	}
}

func TestCost_MissingInputPriceWhenInputUsed(t *testing.T) {
	p := model.ModelPrice{OutputCostPerToken: fp(2e-6)} // 无 input 价
	if _, ok := Cost(model.Tokens{Input: 10, Output: 5}, p); ok {
		t.Error("expected unknown when input price missing and input>0")
	}
}

func TestCost_ZeroInputNoPriceStillKnown(t *testing.T) {
	p := model.ModelPrice{OutputCostPerToken: fp(2e-6)} // 无 input 价，但 input=0
	if _, ok := Cost(model.Tokens{Input: 0, Output: 5}, p); !ok {
		t.Error("input=0 should not require input price")
	}
}

func TestCost_CacheReadFallbackToInput(t *testing.T) {
	p := model.ModelPrice{InputCostPerToken: fp(1e-6), OutputCostPerToken: fp(2e-6)} // 无 cache_read 专价
	c, ok := Cost(model.Tokens{Input: 100, CacheRead: 50}, p)
	if !ok {
		t.Fatal("expected known (cache_read falls back to input price)")
	}
	want := 100*1e-6 + 50*1e-6
	if math.Abs(c-want) > 1e-15 {
		t.Errorf("cost = %v, want %v", c, want)
	}
}

func TestCost_CacheReadSpecialPrice(t *testing.T) {
	p := model.ModelPrice{InputCostPerToken: fp(1e-6), OutputCostPerToken: fp(2e-6), CacheReadCostPerToken: fp(1e-7)}
	c, ok := Cost(model.Tokens{CacheRead: 1000}, p)
	if !ok {
		t.Fatal("expected known")
	}
	if math.Abs(c-1000*1e-7) > 1e-15 {
		t.Errorf("cache_read special price not applied: %v", c)
	}
}

// 容差：成本是 token 数 × 极小单价的累加，1e-9 足以吸收浮点误差。
const costEps = 1e-9

// case 1：OpenAI 风格（cached 是 input 子集）且有 cache_read 专价。
// input 里命中缓存的部分必须按 cache_read 折扣价，非缓存部分才按 input 全价，否则高估。
func TestCost_OpenAIStyle_WithCacheReadPrice(t *testing.T) {
	p := model.ModelPrice{InputCostPerToken: fp(2.5e-6), OutputCostPerToken: fp(15e-6), CacheReadCostPerToken: fp(0.25e-6)}
	c, ok := Cost(model.Tokens{Input: 81310, Cached: 64896, Output: 987}, p)
	if !ok {
		t.Fatal("expected known cost")
	}
	want := float64(81310-64896)*2.5e-6 + 64896*0.25e-6 + 987*15e-6
	if math.Abs(c-want) > costEps {
		t.Errorf("cost = %v, want %v", c, want)
	}
}

// case 2：OpenAI 风格但缺 cache_read 专价 → cached 回退 input 价。
// (input-cached)*ip + cached*ip == input*ip，因此结果等价于全量 input 按 input 价 + output。
func TestCost_OpenAIStyle_NoCacheReadPriceFallsBackToInput(t *testing.T) {
	p := model.ModelPrice{InputCostPerToken: fp(2.5e-6), OutputCostPerToken: fp(15e-6)} // 无 cache_read 专价
	c, ok := Cost(model.Tokens{Input: 81310, Cached: 64896, Output: 987}, p)
	if !ok {
		t.Fatal("expected known cost")
	}
	want := 81310*2.5e-6 + 987*15e-6
	if math.Abs(c-want) > costEps {
		t.Errorf("cost = %v, want %v", c, want)
	}
}

// case 3：Claude 风格（cached=0，用独立 cache_read/cache_creation）。
// 验证新逻辑不影响 Claude 路径：input 全价 + 各自缓存专价 + output。
func TestCost_ClaudeStyle_Unaffected(t *testing.T) {
	p := model.ModelPrice{
		InputCostPerToken:         fp(3e-6),
		OutputCostPerToken:        fp(15e-6),
		CacheReadCostPerToken:     fp(0.3e-6),
		CacheCreationCostPerToken: fp(3.75e-6),
	}
	c, ok := Cost(model.Tokens{Input: 2000, Cached: 0, CacheRead: 1000, CacheCreation: 500, Output: 300}, p)
	if !ok {
		t.Fatal("expected known cost")
	}
	want := 2000*3e-6 + 1000*0.3e-6 + 500*3.75e-6 + 300*15e-6
	if math.Abs(c-want) > costEps {
		t.Errorf("cost = %v, want %v", c, want)
	}
}

// case 4：纯 input/output 无任何缓存，回归不变。
func TestCost_PlainNoCache(t *testing.T) {
	p := model.ModelPrice{InputCostPerToken: fp(2.5e-6), OutputCostPerToken: fp(15e-6)}
	c, ok := Cost(model.Tokens{Input: 1200, Output: 800}, p)
	if !ok {
		t.Fatal("expected known cost")
	}
	want := 1200*2.5e-6 + 800*15e-6
	if math.Abs(c-want) > costEps {
		t.Errorf("cost = %v, want %v", c, want)
	}
}

// case 5：Input>0 但缺 input 价 → 成本未知。
func TestCost_MissingInputPriceUnknown(t *testing.T) {
	p := model.ModelPrice{OutputCostPerToken: fp(15e-6)} // 无 input 价
	if _, ok := Cost(model.Tokens{Input: 1000, Output: 500}, p); ok {
		t.Error("expected unknown when input price missing and input>0")
	}
}

// case 6：异常数据 cached>input → billableInput 兜底 0，不 panic、无负成本。
func TestCost_CachedExceedsInput(t *testing.T) {
	p := model.ModelPrice{InputCostPerToken: fp(2.5e-6), OutputCostPerToken: fp(15e-6), CacheReadCostPerToken: fp(0.25e-6)}
	c, ok := Cost(model.Tokens{Input: 100, Cached: 200, CacheRead: 0}, p)
	if !ok {
		t.Fatal("expected known cost")
	}
	want := 0*2.5e-6 + 200*0.25e-6 // billableInput 兜底为 0
	if math.Abs(c-want) > costEps {
		t.Errorf("cost = %v, want %v", c, want)
	}
	if c < 0 {
		t.Errorf("cost must not be negative: %v", c)
	}
}
