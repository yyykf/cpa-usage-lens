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
	mb := BuildModelBreakdown(rows)

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

// 总 token 相同的模型按名字典序排序，保证输出确定性。
func TestBuildModelBreakdown_TieBrokenByName(t *testing.T) {
	d := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	rows := []model.DailyUsage{
		{UsageDate: d, Source: "a@x.com", Model: "zeta", Tokens: model.Tokens{Total: 100}},
		{UsageDate: d, Source: "a@x.com", Model: "alpha", Tokens: model.Tokens{Total: 100}},
	}
	mb := BuildModelBreakdown(rows)
	if len(mb.Models) != 2 || mb.Models[0] != "alpha" || mb.Models[1] != "zeta" {
		t.Errorf("tie should break by name asc: %v", mb.Models)
	}
}
