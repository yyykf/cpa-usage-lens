package api

import (
	"encoding/json"
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

// queryRange 解析周期并查聚合行；出错时已写响应并返回 ok=false。
func (s *Server) queryRange(w http.ResponseWriter, r *http.Request) ([]model.DailyUsage, bool) {
	start, end, err := s.resolveRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效周期参数"})
		return nil, false
	}
	rows, err := s.store.QueryDailyUsage(r.Context(), start, end)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return nil, false
	}
	return rows, true
}

func (s *Server) handleCollector(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hot, daily, err := s.store.Capacity(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h := model.CollectorHealth{HotBytes: hot, DailyBytes: daily, Status: "stale"}

	state, ok, err := s.store.GetCollectorState(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		if state.LastEventTS != nil {
			lag := int64(now.Sub(*state.LastEventTS).Seconds())
			h.LagSeconds = &lag
		}
	}
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) handleRefreshPrices(w http.ResponseWriter, r *http.Request) {
	if err := s.refresher.Refresh(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
