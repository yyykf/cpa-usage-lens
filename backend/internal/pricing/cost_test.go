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
