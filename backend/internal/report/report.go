// Package report 把按 (账号+模型+天) 的聚合行 + 价格表，组装成前端要的 DTO（含 query-time 成本）。
package report

import (
	"sort"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
	"github.com/code4j/cpa-usage-lens/backend/internal/pricing"
)

type aggregateTokens struct {
	total, input, uncachedInput, output, nonReasoningOutput, reasoning, cached, cacheRead, canonicalCacheRead, cacheCreation, unclassified int64
	completeRequests, unclassifiedRequests, inconsistentRequests, legacyRequests                                                           int64
}

type costAccumulator struct {
	total float64
	known bool
}

func newCostAccumulator() costAccumulator {
	return costAccumulator{known: true}
}

func (a *costAccumulator) add(row model.DailyUsage, price model.ModelPrice, priceKnown bool) {
	if !priceKnown {
		a.known = false
		return
	}
	accounting := accountingForRow(row, price.Provider)
	var cost float64
	var costKnown bool
	if accounting.LegacyRequests == row.Requests {
		cost, costKnown = pricing.CostAtTier(row.Tokens, price, row.LongContext)
	} else {
		cost, costKnown = pricing.CostCanonicalAtTier(accounting.Tokens, price, row.LongContext)
	}
	if !costKnown {
		a.known = false
		return
	}
	a.total += cost
}

func (a *aggregateTokens) add(row model.DailyUsage, provider string) {
	accounting := accountingForRow(row, provider)
	a.total += row.Tokens.Total
	a.input += row.Tokens.Input
	a.uncachedInput += accounting.Tokens.UncachedInput
	a.output += row.Tokens.Output
	a.nonReasoningOutput += accounting.Tokens.NonReasoningOutput
	a.reasoning += accounting.Tokens.Reasoning
	a.cached += row.Tokens.Cached
	a.cacheRead += row.Tokens.CacheRead
	a.canonicalCacheRead += accounting.Tokens.CacheRead
	a.cacheCreation += accounting.Tokens.CacheCreation
	a.unclassified += accounting.Tokens.Unclassified
	a.completeRequests += accounting.CompleteRequests
	a.unclassifiedRequests += accounting.UnclassifiedRequests
	a.inconsistentRequests += accounting.InconsistentRequests
	a.legacyRequests += accounting.LegacyRequests
}

func (a aggregateTokens) breakdown() model.TokenBreakdown {
	return model.TokenBreakdown{
		InputTokens:              a.input,
		UncachedInputTokens:      a.uncachedInput,
		OutputTokens:             a.output,
		ReasoningTokens:          a.reasoning,
		CachedTokens:             a.cached,
		CacheReadTokens:          a.cacheRead,
		CanonicalCacheReadTokens: a.canonicalCacheRead,
		CacheCreationTokens:      a.cacheCreation,
		NonReasoningOutputTokens: a.nonReasoningOutput,
		UnclassifiedTokens:       a.unclassified,
	}
}

func (a aggregateTokens) quality() model.AccountingQualitySummary {
	coverage := "complete"
	if a.unclassifiedRequests > 0 || a.inconsistentRequests > 0 {
		if a.total-a.unclassified > 0 {
			coverage = "partial"
		} else {
			coverage = "unknown"
		}
	}
	return model.AccountingQualitySummary{CostCoverage: coverage, CompleteRequests: a.completeRequests,
		UnclassifiedRequests: a.unclassifiedRequests, InconsistentRequests: a.inconsistentRequests,
		LegacyRequests: a.legacyRequests}
}

func accountingForRow(row model.DailyUsage, provider string) model.AccountingRollup {
	a := row.Accounting
	if a.CompleteRequests+a.UnclassifiedRequests+a.InconsistentRequests > 0 {
		a.Tokens.Total = a.Tokens.UncachedInput + a.Tokens.CacheRead + a.Tokens.CacheCreation +
			a.Tokens.NonReasoningOutput + a.Tokens.Reasoning + a.Tokens.Unclassified
		return a
	}
	split := pricing.SplitInputTokens(row.Tokens, provider)
	nonReasoning := max(row.Tokens.Output-row.Tokens.Reasoning, 0)
	a.Tokens = model.CanonicalTokens{UncachedInput: split.Uncached, CacheRead: split.CacheRead,
		CacheCreation: split.CacheCreation, NonReasoningOutput: nonReasoning, Reasoning: row.Tokens.Reasoning}
	a.Tokens.Total = row.Tokens.Total
	a.CompleteRequests, a.LegacyRequests = row.Requests, row.Requests
	return a
}

// aggCost 累加一组按 model 的用量成本；任一行缺价或缺价格则整体成本标记未知（返回 known=false）。
func aggCost(rows []model.DailyUsage, prices map[string]model.ModelPrice) (float64, bool) {
	cost := newCostAccumulator()
	for _, r := range rows {
		price, priceKnown := prices[r.Model]
		cost.add(r, price, priceKnown)
	}
	return cost.total, cost.known
}

// BuildOverview 汇总周期内总请求/token/成本/失败 + token 拆分。
func BuildOverview(rows []model.DailyUsage, prices map[string]model.ModelPrice) model.Overview {
	var o model.Overview
	var tokens aggregateTokens
	for _, r := range rows {
		o.Requests += r.Requests
		o.Failed += r.FailedRequests
		tokens.add(r, prices[r.Model].Provider)
	}
	o.Tokens = tokens.total
	o.TokenBreakdown = tokens.breakdown()
	o.AccountingQualitySummary = tokens.quality()
	if c, known := aggCost(rows, prices); known && o.CostCoverage != "unknown" {
		o.Cost = &c
	} else if !known {
		o.CostCoverage = "unknown"
	}
	return o
}

// BuildOverviewCompare 把上一等长周期的聚合行汇总成可比块（仅四个 KPI 维度）。
// 返回 nil 表示上一周期完全无数据（rows 为空）→ 调用方据此置 HasPrevious=false，
// 前端不展示百分比（无可比基准），区别于"上一周期有数据但某指标为 0"。
// Cost 沿用 aggCost 的"缺价即未知"语义（nil），与本周期 Overview.Cost 一致。
func BuildOverviewCompare(rows []model.DailyUsage, prices map[string]model.ModelPrice) *model.OverviewCompare {
	if len(rows) == 0 {
		return nil
	}
	c := &model.OverviewCompare{}
	for _, r := range rows {
		c.Requests += r.Requests
		c.Tokens += r.Tokens.Total
		c.Failed += r.FailedRequests
	}
	if cost, known := aggCost(rows, prices); known {
		c.Cost = &cost
	}
	return c
}

// BuildAccounts 按账号汇总用量榜（保持首次出现顺序，调用方可再排序）。
func BuildAccounts(rows []model.DailyUsage, prices map[string]model.ModelPrice) []model.AccountUsage {
	type acc struct {
		requests, failed int64
		tokens           aggregateTokens
		cost             costAccumulator
	}
	m := map[string]*acc{}
	order := []string{}
	for _, r := range rows {
		a := m[r.Source]
		if a == nil {
			a = &acc{cost: newCostAccumulator()}
			m[r.Source] = a
			order = append(order, r.Source)
		}
		a.requests += r.Requests
		a.failed += r.FailedRequests
		price, priceKnown := prices[r.Model]
		a.tokens.add(r, price.Provider)
		a.cost.add(r, price, priceKnown)
	}
	out := make([]model.AccountUsage, 0, len(order))
	for _, s := range order {
		a := m[s]
		au := model.AccountUsage{
			Source: s, Requests: a.requests, Tokens: a.tokens.total, Failed: a.failed,
			TokenBreakdown:           a.tokens.breakdown(),
			AccountingQualitySummary: a.tokens.quality(),
		}
		if a.cost.known && au.CostCoverage != "unknown" {
			au.Cost = &a.cost.total
		} else if !a.cost.known {
			au.CostCoverage = "unknown"
		}
		out = append(out, au)
	}
	return out
}

// BuildKeys 按脱敏 api_key 指纹汇总用量榜（与账号榜并列的独立维度，保持首次出现顺序）。
// 指标口径与 BuildAccounts 完全一致（DRY）；KeyMask 取同指纹下首个非空掩码做展示
// （'none' 桶通常掩码为 '(no key)' 或空，回退到 fingerprint 以免界面空白）。
func BuildKeys(rows []model.DailyUsage, prices map[string]model.ModelPrice) []model.KeyUsage {
	type acc struct {
		mask             string
		requests, failed int64
		tokens           aggregateTokens
		cost             costAccumulator
	}
	m := map[string]*acc{}
	order := []string{}
	for _, r := range rows {
		a := m[r.KeyFingerprint]
		if a == nil {
			a = &acc{cost: newCostAccumulator()}
			m[r.KeyFingerprint] = a
			order = append(order, r.KeyFingerprint)
		}
		if a.mask == "" && r.KeyMask != "" { // 同指纹掩码一致，取首个非空即可
			a.mask = r.KeyMask
		}
		a.requests += r.Requests
		a.failed += r.FailedRequests
		price, priceKnown := prices[r.Model]
		a.tokens.add(r, price.Provider)
		a.cost.add(r, price, priceKnown)
	}
	out := make([]model.KeyUsage, 0, len(order))
	for _, fp := range order {
		a := m[fp]
		mask := a.mask
		if mask == "" { // 掩码缺失兜底，避免前端展示空白
			mask = fp
		}
		ku := model.KeyUsage{
			Fingerprint: fp, KeyMask: mask, Requests: a.requests, Tokens: a.tokens.total, Failed: a.failed,
			TokenBreakdown:           a.tokens.breakdown(),
			AccountingQualitySummary: a.tokens.quality(),
		}
		if a.cost.known && ku.CostCoverage != "unknown" {
			ku.Cost = &a.cost.total
		} else if !a.cost.known {
			ku.CostCoverage = "unknown"
		}
		out = append(out, ku)
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
		quality := qualityForRows(d.rows, prices)
		tp := model.TrendPoint{Date: k, Requests: d.requests, Tokens: d.tokens, Failed: d.failed, CostCoverage: quality.CostCoverage}
		if c, known := aggCost(d.rows, prices); known && quality.CostCoverage != "unknown" {
			tp.Cost = &c
		} else if !known {
			tp.CostCoverage = "unknown"
		}
		out = append(out, tp)
	}
	return out
}

// BuildModelBreakdown 按 模型×天 透视 total_tokens，并附「模型总量排行」。
//   - Models 按周期总 token 降序（相同则按 model 名字典序，保证确定性）；
//   - Daily 按日期升序，每天的 Tokens map 仅含当天有数据的模型；
//     （Models/Daily 服务于「每日 100% 堆叠柱」，恒按 token，与 metric 无关）
//   - Ranking 按 metric 口径降序：metric=="cost" 按成本降序，否则（默认）按 token 降序；
//     每项都带 token 与 cost 两个值（cost 缺价为 nil），前端切口径只改排序展示、无需重查。
//
// 成本逐模型经 aggCost(同一 pricing.Cost) 计算（DRY），任一行缺价则该模型成本未知。
func BuildModelBreakdown(rows []model.DailyUsage, prices map[string]model.ModelPrice, metric string) model.ModelBreakdown {
	if metric != "cost" { // 归一化：仅 token/cost 两种，默认 token
		metric = "token"
	}

	modelTotal := map[string]int64{}             // model -> 周期总 token（用于排序 Models / Ranking.Tokens）
	modelRows := map[string][]model.DailyUsage{} // model -> 该模型周期内全部行（用于逐模型算成本）
	dayTokens := map[string]map[string]int64{}   // date -> model -> 当天 total_tokens
	for _, r := range rows {
		modelTotal[r.Model] += r.Tokens.Total
		modelRows[r.Model] = append(modelRows[r.Model], r)
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

	// Ranking：逐模型算成本，按 metric 口径降序（token 口径直接复用 models 的顺序）。
	ranking := make([]model.ModelRankItem, 0, len(models))
	for _, name := range models {
		quality := qualityForRows(modelRows[name], prices)
		item := model.ModelRankItem{Model: name, Tokens: modelTotal[name], CostCoverage: quality.CostCoverage}
		if cost, known := aggCost(modelRows[name], prices); known && quality.CostCoverage != "unknown" {
			item.Cost = &cost
		} else if !known {
			item.CostCoverage = "unknown"
		}
		ranking = append(ranking, item)
	}
	if metric == "cost" {
		// 成本降序；缺价(nil)视为最小排末尾；成本相同则按 token 降序、再按名字典序，保证确定性。
		sort.SliceStable(ranking, func(i, j int) bool {
			ci, cj := ranking[i].Cost, ranking[j].Cost
			vi, vj := costSortKey(ci), costSortKey(cj)
			if vi != vj {
				return vi > vj
			}
			if ranking[i].Tokens != ranking[j].Tokens {
				return ranking[i].Tokens > ranking[j].Tokens
			}
			return ranking[i].Model < ranking[j].Model
		})
	}

	dates := make([]string, 0, len(dayTokens))
	for date := range dayTokens {
		dates = append(dates, date)
	}
	sort.Strings(dates) // YYYY-MM-DD 字典序即时间升序

	daily := make([]model.ModelDailyPoint, 0, len(dates))
	for _, date := range dates {
		daily = append(daily, model.ModelDailyPoint{Date: date, Tokens: dayTokens[date]})
	}
	return model.ModelBreakdown{Models: models, Daily: daily, Ranking: ranking, Metric: metric}
}

func qualityForRows(rows []model.DailyUsage, prices map[string]model.ModelPrice) model.AccountingQualitySummary {
	var tokens aggregateTokens
	for _, row := range rows {
		tokens.add(row, prices[row.Model].Provider)
	}
	return tokens.quality()
}

// costSortKey 把 *float64 成本映射为排序用数值：nil(未知) 取 -1 排到末尾，
// 已知成本恒 >=0，故未知一定小于任何已知成本。
func costSortKey(c *float64) float64 {
	if c == nil {
		return -1
	}
	return *c
}
