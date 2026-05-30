// Package pricing 负责 LiteLLM 价格表与 query-time 成本计算（不在库里存死 cost）。
package pricing

import "github.com/code4j/cpa-usage-lens/backend/internal/model"

// Cost 用价格表算一组 token 的成本（USD）。
// 规则：input/output 是必须有价的核心维度——若对应 token>0 但缺单价，返回 ok=false（成本"未知"）。
// reasoning 按 output 单价计；cache_read/cache_creation 有专价用专价，否则回退到 input 单价。
func Cost(t model.Tokens, p model.ModelPrice) (float64, bool) {
	ip, op := p.InputCostPerToken, p.OutputCostPerToken
	if (t.Input > 0 && ip == nil) || (t.Output > 0 && op == nil) {
		return 0, false
	}
	if t.Reasoning > 0 && op == nil {
		return 0, false
	}

	var c float64
	if ip != nil {
		c += float64(t.Input) * *ip
	}
	if op != nil {
		c += float64(t.Output) * *op
		c += float64(t.Reasoning) * *op // reasoning 计入 output 单价
	}
	c += cacheCost(t.CacheRead, p.CacheReadCostPerToken, ip)
	c += cacheCost(t.CacheCreation, p.CacheCreationCostPerToken, ip)
	return c, true
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
