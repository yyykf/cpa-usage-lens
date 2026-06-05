package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
	"github.com/code4j/cpa-usage-lens/backend/internal/report"
	"github.com/code4j/cpa-usage-lens/backend/internal/timeutil"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	if !s.auth.CheckPassword(body.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "密码错误"})
		return
	}
	tok, err := s.auth.IssueToken(time.Now())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "签发 token 失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": tok})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	start, end, ok := s.resolveRangeOrFail(w, r)
	if !ok {
		return
	}
	rows, ok := s.queryDaily(w, r, start, end)
	if !ok {
		return
	}
	prices := s.prices.Prices()
	ov := report.BuildOverview(rows, prices)

	// 环比：查与本周期紧邻且等长的上一区间，汇总成可比块。
	prevStart, prevEnd := timeutil.PreviousRange(start, end)
	prevRows, ok := s.queryDaily(w, r, prevStart, prevEnd)
	if !ok {
		return
	}
	if cmp := report.BuildOverviewCompare(prevRows, prices); cmp != nil {
		ov.HasPrevious = true
		ov.Previous = cmp
	}
	writeJSON(w, http.StatusOK, ov)
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	rows, ok := s.queryRange(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, report.BuildAccounts(rows, s.prices.Prices()))
}

// handleKeys 返回「API key 用量榜」（按脱敏 key 指纹聚合，与账号榜并列的独立维度）。
// 复用与账号榜同一份周期聚合行 + 同一成本算法，口径完全对齐（DRY）。
func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request) {
	rows, ok := s.queryRange(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, report.BuildKeys(rows, s.prices.Prices()))
}

func (s *Server) handleTrend(w http.ResponseWriter, r *http.Request) {
	rows, ok := s.queryRange(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, report.BuildTrend(rows, s.prices.Prices()))
}

// handleModels 返回模型用量分布（每日堆叠柱 + 模型总量排行）。
// query 参数 metric=token|cost 决定排行口径（默认 token）；cost 口径用价格表实时算。
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	rows, ok := s.queryRange(w, r)
	if !ok {
		return
	}
	metric := r.URL.Query().Get("metric") // 归一化在 report.BuildModelBreakdown 内（非 cost 即 token）
	writeJSON(w, http.StatusOK, report.BuildModelBreakdown(rows, s.prices.Prices(), metric))
}

// queryRange 解析周期并查当前周期聚合行；出错时已写响应并返回 ok=false。
// accounts/trend/models 只需当前周期；overview 因需环比，改用 resolveRangeOrFail + queryDaily。
func (s *Server) queryRange(w http.ResponseWriter, r *http.Request) ([]model.DailyUsage, bool) {
	start, end, ok := s.resolveRangeOrFail(w, r)
	if !ok {
		return nil, false
	}
	return s.queryDaily(w, r, start, end)
}

// resolveRangeOrFail 解析周期为 [start, end)；解析失败时已写 400 响应并返回 ok=false。
func (s *Server) resolveRangeOrFail(w http.ResponseWriter, r *http.Request) (start, end time.Time, ok bool) {
	start, end, err := s.resolveRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效周期参数"})
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

// queryDaily 查 [start, end) 区间聚合行（time.Time 转为按时区的 YYYY-MM-DD）；
// 出错时已写响应并返回 ok=false。
func (s *Server) queryDaily(w http.ResponseWriter, r *http.Request, start, end time.Time) ([]model.DailyUsage, bool) {
	rows, err := s.store.QueryDailyUsage(r.Context(), timeutil.DateString(start, s.loc), timeutil.DateString(end, s.loc))
	if err != nil {
		log.Printf("查询用量失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return nil, false
	}
	return rows, true
}

func (s *Server) handleCollector(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hot, daily, err := s.store.Capacity(ctx)
	if err != nil {
		log.Printf("查询容量失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}
	h := model.CollectorHealth{HotBytes: hot, DailyBytes: daily, Status: "stale"}

	state, ok, err := s.store.GetCollectorState(ctx)
	if err != nil {
		log.Printf("查询采集器状态失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "查询失败"})
		return
	}
	if ok {
		now := time.Now()
		h.LastPollAt = state.LastPollAt
		h.LastEventTS = state.LastEventTS
		h.EventsIngested = state.EventsIngested
		h.LastError = state.LastError
		switch {
		case state.LastError != "" && state.LastErrorAt != nil && now.Sub(*state.LastErrorAt) < 5*time.Minute:
			h.Status = "error"
		case state.LastPollAt != nil && now.Sub(*state.LastPollAt) < time.Minute:
			h.Status = "running"
		}
		// 已恢复的旧错误（Status 已回到 running/stale）不再下发，
		// 避免前端把几小时前的瞬时错误当成"当前故障"常驻显示为红字；
		// collector_state.last_error 仍保留在库里，运维可直接查表追溯。
		if h.Status != "error" {
			h.LastError = ""
		}
		if state.LastEventTS != nil {
			lag := int64(now.Sub(*state.LastEventTS).Seconds())
			h.LagSeconds = &lag
		}
	}
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) handleRefreshPrices(w http.ResponseWriter, r *http.Request) {
	if err := s.refresher.Refresh(r.Context()); err != nil {
		log.Printf("刷新价格表失败: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "刷新价格失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
