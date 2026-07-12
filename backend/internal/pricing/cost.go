// Package pricing 负责 LiteLLM 价格表与 query-time 成本计算（不在库里存死 cost）。
package pricing

import (
	"strings"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

// InputTokenBreakdown 是互斥的输入计费类别，供成本与报表展示复用同一口径。
type InputTokenBreakdown struct {
	Uncached      int64
	CacheRead     int64
	CacheCreation int64
}

// CostAtTier 按 daily 行已持久化的价格档位计算成本。长上下文是整请求切价，
// 因此先替换四类单价，再复用同一套 token 拆分逻辑。
func CostAtTier(t model.Tokens, p model.ModelPrice, longContext bool) (float64, bool) {
	if !longContext {
		return Cost(t, p)
	}
	p.InputCostPerToken = p.LongContextInputCostPerToken
	p.OutputCostPerToken = p.LongContextOutputCostPerToken
	p.CacheReadCostPerToken = p.LongContextCacheReadCostPerToken
	p.CacheCreationCostPerToken = p.LongContextCacheCreationCostPerToken
	return Cost(t, p)
}

// Cost 用价格表算一组 token 的成本（USD）。
// 规则：input/output 是必须有价的核心维度——若对应 token>0 但缺单价，返回 ok=false（成本"未知"）。
// OpenAI reasoning 是 output 的拆分，不额外叠加；其他 provider 保留独立 reasoning 计费。
// cache_read/cache_creation 有专价用专价，否则回退到 input 单价。
func Cost(t model.Tokens, p model.ModelPrice) (float64, bool) {
	ip, op := p.InputCostPerToken, p.OutputCostPerToken
	reasoningAdditional := t.Reasoning > 0 && !strings.EqualFold(p.Provider, "openai")
	if (t.Input > 0 && ip == nil) || (t.Output > 0 && op == nil) {
		return 0, false
	}
	if reasoningAdditional && op == nil {
		return 0, false
	}
	var c float64

	input := SplitInputTokens(t, p.Provider)
	if ip != nil {
		c += float64(input.Uncached) * *ip
	}
	if op != nil {
		c += float64(t.Output) * *op
		if reasoningAdditional {
			c += float64(t.Reasoning) * *op
		}
	}
	c += cacheCost(input.CacheRead, p.CacheReadCostPerToken, ip)
	c += cacheCost(input.CacheCreation, p.CacheCreationCostPerToken, ip)
	return c, true
}

// SplitInputTokens 把 provider 原始字段归一为互斥的普通输入/缓存读/缓存写三类。
func SplitInputTokens(t model.Tokens, provider string) InputTokenBreakdown {
	cacheRead := t.CacheRead
	uncached := t.Input
	if inputIncludesCache(t, provider) {
		if t.Cached > cacheRead {
			cacheRead = t.Cached
		}
		uncached = t.Input - cacheRead - t.CacheCreation
		if uncached < 0 {
			uncached = 0
		}
	}
	return InputTokenBreakdown{Uncached: uncached, CacheRead: cacheRead, CacheCreation: t.CacheCreation}
}

// inputIncludesCache 优先使用 LiteLLM provider 元数据决定语义。
// Provider 为空只用于兼容尚未刷新 provider 的旧价格行，沿用旧版字段形状判断。
func inputIncludesCache(t model.Tokens, provider string) bool {
	switch strings.ToLower(provider) {
	case "openai":
		return true
	case "anthropic":
		return false
	default:
		return t.Cached > 0 && (t.CacheRead == 0 || t.CacheRead == t.Cached)
	}
}

// cacheCost 缓存 token 成本：优先专价，否则回退 input 价（缓存读写近似按输入计）。
func cacheCost(tokens int64, special, fallback *float64) float64 {
	if tokens == 0 {
		return 0
	}
	if special != nil {
		return float64(tokens) * *special
	}
	if fallback != nil {
		return float64(tokens) * *fallback
	}
	return 0
}
