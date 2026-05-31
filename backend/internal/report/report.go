// Package report 把按 (账号+模型+天) 的聚合行 + 价格表，组装成前端要的 DTO（含 query-time 成本）。
package report

import (
	"sort"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
	"github.com/code4j/cpa-usage-lens/backend/internal/pricing"
)

// aggCost 累加一组按 model 的用量成本；任一行缺价或缺价格则整体成本标记未知（返回 known=false）。
func aggCost(rows []model.DailyUsage, prices map[string]model.ModelPrice) (float64, bool) {
	var total float64
	known := true
	for _, r := range rows {
		p, ok := prices[r.Model]
		if !ok {
			known = false
			continue
		}
		c, ok := pricing.Cost(r.Tokens, p)
		if !ok {
			known = false
			continue
		}
		total += c
	}
	return total, known
}

// BuildOverview 汇总周期内总请求/token/成本/失败 + token 拆分。
func BuildOverview(rows []model.DailyUsage, prices map[string]model.ModelPrice) model.Overview {
	var o model.Overview
	for _, r := range rows {
		o.Requests += r.Requests
		o.Tokens += r.Tokens.Total
		o.Failed += r.FailedRequests
		o.InputTokens += r.Tokens.Input
		o.OutputTokens += r.Tokens.Output
		o.ReasoningTokens += r.Tokens.Reasoning
		o.CachedTokens += r.Tokens.Cached
		o.CacheReadTokens += r.Tokens.CacheRead
		o.CacheCreationTokens += r.Tokens.CacheCreation
	}
	if c, known := aggCost(rows, prices); known {
		o.Cost = &c
	}
	return o
}

// BuildAccounts 按账号汇总用量榜（保持首次出现顺序，调用方可再排序）。
func BuildAccounts(rows []model.DailyUsage, prices map[string]model.ModelPrice) []model.AccountUsage {
	type acc struct {
		requests, tokens, failed                                int64
		input, output, reasoning, cached, cacheRead, cacheCreat int64
		rows                                                    []model.DailyUsage
	}
	m := map[string]*acc{}
	order := []string{}
	for _, r := range rows {
		a := m[r.Source]
		if a == nil {
			a = &acc{}
			m[r.Source] = a
			order = append(order, r.Source)
		}
		a.requests += r.Requests
		a.tokens += r.Tokens.Total
		a.failed += r.FailedRequests
		a.input += r.Tokens.Input
		a.output += r.Tokens.Output
		a.reasoning += r.Tokens.Reasoning
		a.cached += r.Tokens.Cached
		a.cacheRead += r.Tokens.CacheRead
		a.cacheCreat += r.Tokens.CacheCreation
		a.rows = append(a.rows, r)
	}
	out := make([]model.AccountUsage, 0, len(order))
	for _, s := range order {
		a := m[s]
		au := model.AccountUsage{
			Source: s, Requests: a.requests, Tokens: a.tokens, Failed: a.failed,
			InputTokens: a.input, OutputTokens: a.output, ReasoningTokens: a.reasoning,
			CachedTokens: a.cached, CacheReadTokens: a.cacheRead, CacheCreationTokens: a.cacheCreat,
		}
		if c, known := aggCost(a.rows, prices); known {
			au.Cost = &c
		}
		out = append(out, au)
	}
	return out
}

// BuildTrend 按天汇总趋势（usage_date 已是按时区界定的"天"，直接格式化）。
func BuildTrend(rows []model.DailyUsage, prices map[string]model.ModelPrice) []model.TrendPoint {
	type day struct {
		requests, tokens, failed int64
		rows                     []model.DailyUsage
	}
	m := map[string]*day{}
	order := []string{}
	for _, r := range rows {
		key := r.UsageDate.Format("2006-01-02")
		d := m[key]
		if d == nil {
			d = &day{}
			m[key] = d
			order = append(order, key)
		}
		d.requests += r.Requests
		d.tokens += r.Tokens.Total
		d.failed += r.FailedRequests
		d.rows = append(d.rows, r)
	}
	out := make([]model.TrendPoint, 0, len(order))
	for _, k := range order {
		d := m[k]
		tp := model.TrendPoint{Date: k, Requests: d.requests, Tokens: d.tokens, Failed: d.failed}
		if c, known := aggCost(d.rows, prices); known {
			tp.Cost = &c
		}
		out = append(out, tp)
	}
	return out
}

// BuildModelBreakdown 按 模型×天 透视 total_tokens（仅按 token，不涉及成本）。
// Models 按周期总 token 降序（相同则按 model 名字典序，保证确定性）；
// Daily 按日期升序，每天的 Tokens map 仅含当天有数据的模型。
func BuildModelBreakdown(rows []model.DailyUsage) model.ModelBreakdown {
	modelTotal := map[string]int64{}                  // model -> 周期总 token（用于排序 Models）
	dayTokens := map[string]map[string]int64{}        // date -> model -> 当天 total_tokens
	for _, r := range rows {
		modelTotal[r.Model] += r.Tokens.Total
		date := r.UsageDate.Format("2006-01-02")
		dm := dayTokens[date]
		if dm == nil {
			dm = map[string]int64{}
			dayTokens[date] = dm
		}
		dm[r.Model] += r.Tokens.Total
	}

	models := make([]string, 0, len(modelTotal))
	for name := range modelTotal {
		models = append(models, name)
	}
	sort.Slice(models, func(i, j int) bool {
		if modelTotal[models[i]] != modelTotal[models[j]] {
			return modelTotal[models[i]] > modelTotal[models[j]] // 总 token 降序
		}
		return models[i] < models[j] // 同量按名字典序
	})

	dates := make([]string, 0, len(dayTokens))
	for date := range dayTokens {
		dates = append(dates, date)
	}
	sort.Strings(dates) // YYYY-MM-DD 字典序即时间升序

	daily := make([]model.ModelDailyPoint, 0, len(dates))
	for _, date := range dates {
		daily = append(daily, model.ModelDailyPoint{Date: date, Tokens: dayTokens[date]})
	}
	return model.ModelBreakdown{Models: models, Daily: daily}
}
