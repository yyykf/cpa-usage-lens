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
		{UsageDate: d1, Source: "a@x.com", Model: "gpt-5.4", Requests: 10, FailedRequests: 1, Tokens: model.Tokens{Total: 1000, Input: 600, Output: 400}},
		{UsageDate: d1, Source: "a@x.com", Model: "claude", Requests: 5, Tokens: model.Tokens{Total: 500, Input: 300, Output: 200}},
		{UsageDate: d2, Source: "b@x.com", Model: "gpt-5.4", Requests: 20, FailedRequests: 2, Tokens: model.Tokens{Total: 2000, Input: 1200, Output: 800}},
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
