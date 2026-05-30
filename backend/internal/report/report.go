// Package report 把按 (账号+模型+天) 的聚合行 + 价格表，组装成前端要的 DTO（含 query-time 成本）。
package report

import (
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

// BuildOverview 汇总周期内总请求/token/成本/失败。
func BuildOverview(rows []model.DailyUsage, prices map[string]model.ModelPrice) model.Overview {
	var o model.Overview
	for _, r := range rows {
		o.Requests += r.Requests
		o.Tokens += r.Tokens.Total
		o.Failed += r.FailedRequests
	}
	if c, known := aggCost(rows, prices); known {
		o.Cost = &c
	}
	return o
}

// BuildAccounts 按账号汇总用量榜（保持首次出现顺序，调用方可再排序）。
func BuildAccounts(rows []model.DailyUsage, prices map[string]model.ModelPrice) []model.AccountUsage {
	type acc struct {
		requests, tokens, failed int64
		rows                     []model.DailyUsage
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
		a.rows = append(a.rows, r)
	}
	out := make([]model.AccountUsage, 0, len(order))
	for _, s := range order {
		a := m[s]
		au := model.AccountUsage{Source: s, Requests: a.requests, Tokens: a.tokens, Failed: a.failed}
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
