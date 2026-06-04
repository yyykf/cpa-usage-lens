package report

import (
	"testing"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
)

func fp(v float64) *float64 { return &v }

func sampleRows() []model.DailyUsage {
	d1 := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	return []model.DailyUsage{
		{UsageDate: d1, Source: "a@x.com", Model: "gpt-5.4", Requests: 10, FailedRequests: 1, Tokens: model.Tokens{Total: 1000, Input: 600, Output: 400, Reasoning: 50, Cached: 100, CacheRead: 80, CacheCreation: 20}},
		{UsageDate: d1, Source: "a@x.com", Model: "claude", Requests: 5, Tokens: model.Tokens{Total: 500, Input: 300, Output: 200, Reasoning: 10, Cached: 40, CacheRead: 30, CacheCreation: 10}},
		{UsageDate: d2, Source: "b@x.com", Model: "gpt-5.4", Requests: 20, FailedRequests: 2, Tokens: model.Tokens{Total: 2000, Input: 1200, Output: 800, Reasoning: 100, Cached: 200, CacheRead: 150, CacheCreation: 50}},
	}
}

func prices() map[string]model.ModelPrice {
	return map[string]model.ModelPrice{
		"gpt-5.4": {InputCostPerToken: fp(1e-6), OutputCostPerToken: fp(2e-6)},
		"claude":  {InputCostPerToken: fp(3e-6), OutputCostPerToken: fp(6e-6)},
	}
}

func TestBuildOverview(t *testing.T) {
	o := BuildOverview(sampleRows(), prices())
	if o.Requests != 35 || o.Tokens != 3500 || o.Failed != 3 {
		t.Errorf("overview totals wrong: %+v", o)
	}
	if o.Cost == nil {
		t.Error("cost should be known")
	}
	// token 拆分应分别累加（不只是 total）
	if o.InputTokens != 2100 || o.OutputTokens != 1400 || o.ReasoningTokens != 160 {
		t.Errorf("overview token split wrong: %+v", o)
	}
	if o.CachedTokens != 340 || o.CacheReadTokens != 260 || o.CacheCreationTokens != 80 {
		t.Errorf("overview cache token split wrong: %+v", o)
	}
}

func TestBuildOverview_MissingPriceMeansUnknownCost(t *testing.T) {
	o := BuildOverview(sampleRows(), map[string]model.ModelPrice{})
	if o.Cost != nil {
		t.Error("cost should be unknown when prices missing")
	}
	if o.Requests != 35 {
		t.Error("requests should still aggregate even without prices")
	}
}

func TestBuildAccounts(t *testing.T) {
	accs := BuildAccounts(sampleRows(), prices())
	if len(accs) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accs))
	}
	if accs[0].Source != "a@x.com" || accs[0].Requests != 15 || accs[0].Tokens != 1500 {
		t.Errorf("account a wrong: %+v", accs[0])
	}
	// 账号 a 的 token 拆分应为前两行之和
	if accs[0].InputTokens != 900 || accs[0].OutputTokens != 600 || accs[0].ReasoningTokens != 60 {
		t.Errorf("account a token split wrong: %+v", accs[0])
	}
	if accs[0].CachedTokens != 140 || accs[0].CacheReadTokens != 110 || accs[0].CacheCreationTokens != 30 {
		t.Errorf("account a cache token split wrong: %+v", accs[0])
	}
}

// BuildKeys：按脱敏指纹聚合，同指纹跨账号/模型合并，掩码随指纹带出，成本沿用账号榜口径。
func TestBuildKeys(t *testing.T) {
	d1 := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	rows := []model.DailyUsage{
		// key fp1 跨两账号/两模型/两天 → 应合并成一行
		{UsageDate: d1, Source: "a@x.com", Model: "gpt-5.4", KeyFingerprint: "fp1", KeyMask: "sk-aaaa…1111", Requests: 10, FailedRequests: 1, Tokens: model.Tokens{Total: 1000, Input: 600, Output: 400}},
		{UsageDate: d1, Source: "b@x.com", Model: "claude", KeyFingerprint: "fp1", KeyMask: "sk-aaaa…1111", Requests: 5, Tokens: model.Tokens{Total: 500, Input: 300, Output: 200}},
		{UsageDate: d2, Source: "a@x.com", Model: "gpt-5.4", KeyFingerprint: "fp1", KeyMask: "sk-aaaa…1111", Requests: 5, Tokens: model.Tokens{Total: 500, Input: 500}},
		// 另一把 key
		{UsageDate: d2, Source: "a@x.com", Model: "gpt-5.4", KeyFingerprint: "fp2", KeyMask: "sk-bbbb…2222", Requests: 20, FailedRequests: 2, Tokens: model.Tokens{Total: 2000, Input: 1200, Output: 800}},
	}
	keys := BuildKeys(rows, prices())
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	// 保持首次出现顺序：fp1 先
	if keys[0].Fingerprint != "fp1" || keys[0].KeyMask != "sk-aaaa…1111" {
		t.Errorf("key0 identity wrong: %+v", keys[0])
	}
	// fp1 跨行合并：requests 10+5+5=20，tokens 1000+500+500=2000，failed=1
	if keys[0].Requests != 20 || keys[0].Tokens != 2000 || keys[0].Failed != 1 {
		t.Errorf("fp1 aggregation wrong: %+v", keys[0])
	}
	if keys[0].InputTokens != 1400 || keys[0].OutputTokens != 600 {
		t.Errorf("fp1 token split wrong: %+v", keys[0])
	}
	if keys[0].Cost == nil {
		t.Error("fp1 cost should be known (all models priced)")
	}
	if keys[1].Fingerprint != "fp2" || keys[1].Requests != 20 || keys[1].Failed != 2 {
		t.Errorf("fp2 wrong: %+v", keys[1])
	}
}

// 缺价模型 → 该 key 成本未知（nil），但请求/token 仍照常聚合（与账号榜一致）。
func TestBuildKeys_MissingPriceUnknownCost(t *testing.T) {
	d := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	rows := []model.DailyUsage{
		{UsageDate: d, Source: "a", Model: "noprice", KeyFingerprint: "fp1", KeyMask: "sk-x…1", Requests: 3, Tokens: model.Tokens{Total: 100, Input: 100}},
	}
	keys := BuildKeys(rows, map[string]model.ModelPrice{})
	if len(keys) != 1 || keys[0].Cost != nil {
		t.Errorf("missing price should yield nil cost: %+v", keys)
	}
	if keys[0].Requests != 3 {
		t.Error("requests should aggregate even without prices")
	}
}

// 非 key 认证桶（fp='none'）掩码为空时回退到指纹，避免前端展示空白。
func TestBuildKeys_NoneBucketMaskFallback(t *testing.T) {
	d := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	rows := []model.DailyUsage{
		{UsageDate: d, Source: "a", Model: "gpt-5.4", KeyFingerprint: "none", KeyMask: "", Requests: 1, Tokens: model.Tokens{Total: 10, Input: 10}},
	}
	keys := BuildKeys(rows, prices())
	if len(keys) != 1 || keys[0].Fingerprint != "none" {
		t.Fatalf("expected single none bucket: %+v", keys)
	}
	if keys[0].KeyMask != "none" { // 掩码空 → 回退指纹
		t.Errorf("empty mask should fall back to fingerprint, got %q", keys[0].KeyMask)
	}
}

func TestBuildTrend(t *testing.T) {
	tr := BuildTrend(sampleRows(), prices())
	if len(tr) != 2 {
		t.Fatalf("expected 2 days, got %d", len(tr))
	}
	if tr[0].Date != "2026-05-30" || tr[0].Requests != 15 {
		t.Errorf("day1 wrong: %+v", tr[0])
	}
	if tr[1].Date != "2026-05-31" || tr[1].Requests != 20 {
		t.Errorf("day2 wrong: %+v", tr[1])
	}
}

func TestBuildModelBreakdown(t *testing.T) {
	d1 := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	// 输入日期故意逆序 + 同天同模型跨账号，验证内部会合并并按日期升序输出
	rows := []model.DailyUsage{
		{UsageDate: d2, Source: "a@x.com", Model: "gpt", Tokens: model.Tokens{Total: 300}},
		{UsageDate: d1, Source: "a@x.com", Model: "gpt", Tokens: model.Tokens{Total: 500}},
		{UsageDate: d1, Source: "b@x.com", Model: "gpt", Tokens: model.Tokens{Total: 300}},
		{UsageDate: d1, Source: "a@x.com", Model: "claude", Tokens: model.Tokens{Total: 200}},
		{UsageDate: d2, Source: "a@x.com", Model: "gemini", Tokens: model.Tokens{Total: 400}},
	}
	mb := BuildModelBreakdown(rows, nil, "")

	// Models 按周期总 token 降序：gpt(1100) > gemini(400) > claude(200)
	if got := mb.Models; len(got) != 3 || got[0] != "gpt" || got[1] != "gemini" || got[2] != "claude" {
		t.Fatalf("models order wrong: %v", got)
	}

	// Daily 按日期升序，且每天仅含当天有数据的模型
	if len(mb.Daily) != 2 {
		t.Fatalf("expected 2 days, got %d", len(mb.Daily))
	}
	if mb.Daily[0].Date != "2026-05-30" || mb.Daily[1].Date != "2026-05-31" {
		t.Fatalf("daily not date-ascending: %+v", mb.Daily)
	}
	// d1：gpt 跨账号合并为 800，claude 200，无 gemini
	day1 := mb.Daily[0].Tokens
	if day1["gpt"] != 800 || day1["claude"] != 200 {
		t.Errorf("day1 pivot wrong: %v", day1)
	}
	if _, ok := day1["gemini"]; ok {
		t.Errorf("day1 should not contain gemini: %v", day1)
	}
	// d2：gpt 300，gemini 400，无 claude
	day2 := mb.Daily[1].Tokens
	if day2["gpt"] != 300 || day2["gemini"] != 400 {
		t.Errorf("day2 pivot wrong: %v", day2)
	}
	if _, ok := day2["claude"]; ok {
		t.Errorf("day2 should not contain claude: %v", day2)
	}
}

// 环比块：有数据时汇总四个 KPI 维度；成本沿用"缺价即未知"。
func TestBuildOverviewCompare(t *testing.T) {
	cmp := BuildOverviewCompare(sampleRows(), prices())
	if cmp == nil {
		t.Fatal("compare should be non-nil when rows present")
	}
	if cmp.Requests != 35 || cmp.Tokens != 3500 || cmp.Failed != 3 {
		t.Errorf("compare totals wrong: %+v", cmp)
	}
	if cmp.Cost == nil {
		t.Error("compare cost should be known")
	}
}

// 上一周期完全无数据 → 返回 nil（前端据此置 HasPrevious=false，不展示百分比）。
func TestBuildOverviewCompare_EmptyMeansNil(t *testing.T) {
	if cmp := BuildOverviewCompare(nil, prices()); cmp != nil {
		t.Errorf("empty rows should yield nil compare, got %+v", cmp)
	}
	if cmp := BuildOverviewCompare([]model.DailyUsage{}, prices()); cmp != nil {
		t.Errorf("empty slice should yield nil compare, got %+v", cmp)
	}
}

// 上一周期有数据但缺价 → 块非 nil（有可比基准），但 Cost 为 nil（成本未知）。
func TestBuildOverviewCompare_MissingPriceUnknownCost(t *testing.T) {
	cmp := BuildOverviewCompare(sampleRows(), map[string]model.ModelPrice{})
	if cmp == nil {
		t.Fatal("compare should be non-nil even when prices missing")
	}
	if cmp.Cost != nil {
		t.Error("compare cost should be unknown when prices missing")
	}
	if cmp.Requests != 35 {
		t.Error("requests should still aggregate even without prices")
	}
}

// 默认（空 metric）按 token 降序，Ranking 与 Models 顺序一致，且每项 token/cost 双值。
func TestBuildModelBreakdown_RankingDefaultToken(t *testing.T) {
	mb := BuildModelBreakdown(sampleRows(), prices(), "")
	if mb.Metric != "token" {
		t.Errorf("default metric should normalize to token, got %q", mb.Metric)
	}
	// gpt-5.4 总 3000 token > claude 500 token
	if len(mb.Ranking) != 2 || mb.Ranking[0].Model != "gpt-5.4" || mb.Ranking[1].Model != "claude" {
		t.Fatalf("ranking token order wrong: %+v", mb.Ranking)
	}
	if mb.Ranking[0].Tokens != 3000 || mb.Ranking[1].Tokens != 500 {
		t.Errorf("ranking tokens wrong: %+v", mb.Ranking)
	}
	if mb.Ranking[0].Cost == nil || mb.Ranking[1].Cost == nil {
		t.Error("ranking cost should be present when priced")
	}
}

// cost 口径按成本降序：claude 单价更高，总成本可超过 token 更多的 gpt。
func TestBuildModelBreakdown_RankingByCost(t *testing.T) {
	d := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	// gpt: 1000 token 全 input @1e-6 = 0.001；claude: 500 input @3e-6 = 0.0015 → claude 成本更高
	rows := []model.DailyUsage{
		{UsageDate: d, Source: "a", Model: "gpt-5.4", Requests: 1, Tokens: model.Tokens{Total: 1000, Input: 1000}},
		{UsageDate: d, Source: "a", Model: "claude", Requests: 1, Tokens: model.Tokens{Total: 500, Input: 500}},
	}
	mb := BuildModelBreakdown(rows, prices(), "cost")
	if mb.Metric != "cost" {
		t.Errorf("metric should be cost, got %q", mb.Metric)
	}
	// 成本降序：claude(0.0015) > gpt(0.001)，尽管 gpt 的 token 更多
	if mb.Ranking[0].Model != "claude" || mb.Ranking[1].Model != "gpt-5.4" {
		t.Fatalf("ranking cost order wrong: %+v", mb.Ranking)
	}
}

// cost 口径下缺价模型(cost=nil)排到末尾，已知成本者在前。
func TestBuildModelBreakdown_RankingCostMissingPriceLast(t *testing.T) {
	d := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	rows := []model.DailyUsage{
		{UsageDate: d, Source: "a", Model: "noprice", Tokens: model.Tokens{Total: 9999, Input: 9999}}, // 无价格
		{UsageDate: d, Source: "a", Model: "gpt-5.4", Tokens: model.Tokens{Total: 100, Input: 100}},   // 有价格
	}
	mb := BuildModelBreakdown(rows, prices(), "cost")
	if mb.Ranking[0].Model != "gpt-5.4" {
		t.Fatalf("priced model should rank before unpriced under cost: %+v", mb.Ranking)
	}
	if mb.Ranking[1].Model != "noprice" || mb.Ranking[1].Cost != nil {
		t.Errorf("unpriced model should be last with nil cost: %+v", mb.Ranking)
	}
}

// 未知 metric 归一化为 token（默认口径），不报错。
func TestBuildModelBreakdown_UnknownMetricFallsBackToToken(t *testing.T) {
	mb := BuildModelBreakdown(sampleRows(), prices(), "garbage")
	if mb.Metric != "token" {
		t.Errorf("unknown metric should fall back to token, got %q", mb.Metric)
	}
}

// 总 token 相同的模型按名字典序排序，保证输出确定性。
func TestBuildModelBreakdown_TieBrokenByName(t *testing.T) {
	d := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	rows := []model.DailyUsage{
		{UsageDate: d, Source: "a@x.com", Model: "zeta", Tokens: model.Tokens{Total: 100}},
		{UsageDate: d, Source: "a@x.com", Model: "alpha", Tokens: model.Tokens{Total: 100}},
	}
	mb := BuildModelBreakdown(rows, nil, "")
	if len(mb.Models) != 2 || mb.Models[0] != "alpha" || mb.Models[1] != "zeta" {
		t.Errorf("tie should break by name asc: %v", mb.Models)
	}
}
