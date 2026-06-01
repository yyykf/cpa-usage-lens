package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
	"github.com/code4j/cpa-usage-lens/backend/internal/report"
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
	rows, ok := s.queryRange(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, report.BuildOverview(rows, s.prices.Prices()))
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	rows, ok := s.queryRange(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, report.BuildAccounts(rows, s.prices.Prices()))
}

func (s *Server) handleTrend(w http.ResponseWriter, r *http.Request) {
	rows, ok := s.queryRange(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, report.BuildTrend(rows, s.prices.Prices()))
}

// handleModels 返回模型用量分布（按 token，不涉及成本，故无需价格表）。
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	rows, ok := s.queryRange(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, report.BuildModelBreakdown(rows))
}

// queryRange 解析周期并查聚合行；出错时已写响应并返回 ok=false。
func (s *Server) queryRange(w http.ResponseWriter, r *http.Request) ([]model.DailyUsage, bool) {
	start, end, err := s.resolveRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效周期参数"})
		return nil, false
	}
	rows, err := s.store.QueryDailyUsage(r.Context(), start, end)
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
