// Package api 提供鉴权与数据查询 HTTP 接口。
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/code4j/cpa-usage-lens/backend/internal/model"
	"github.com/code4j/cpa-usage-lens/backend/internal/timeutil"
)

// DataStore 是 API 查询依赖（由 db.DB 实现）。
type DataStore interface {
	QueryDailyUsage(ctx context.Context, startDate, endDate string) ([]model.DailyUsage, error)
	Capacity(ctx context.Context) (hotBytes, dailyBytes int64, err error)
	GetCollectorState(ctx context.Context) (model.CollectorState, bool, error)
}

// Prices 提供当前价格表（由 pricing.Service 实现）。
type Prices interface {
	Prices() map[string]model.ModelPrice
}

// PriceRefresher 手动刷新价格（由 pricing.Service 实现）。
type PriceRefresher interface {
	Refresh(ctx context.Context) error
}

// Server 装配数据 API + 鉴权。
type Server struct {
	store     DataStore
	prices    Prices
	refresher PriceRefresher
	auth      *Authenticator
	loc       *time.Location
}

func NewServer(store DataStore, prices Prices, refresher PriceRefresher, auth *Authenticator, loc *time.Location) *Server {
	return &Server{store: store, prices: prices, refresher: refresher, auth: auth, loc: loc}
}

// Handler 返回装好路由的 http.Handler。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.Handle("GET /api/overview", s.requireAuth(s.handleOverview))
	mux.Handle("GET /api/accounts", s.requireAuth(s.handleAccounts))
	mux.Handle("GET /api/keys", s.requireAuth(s.handleKeys))
	mux.Handle("GET /api/trend", s.requireAuth(s.handleTrend))
	mux.Handle("GET /api/models", s.requireAuth(s.handleModels))
	mux.Handle("GET /api/collector", s.requireAuth(s.handleCollector))
	mux.Handle("POST /api/prices/refresh", s.requireAuth(s.handleRefreshPrices))
	return cors(mux)
}

// requireAuth 校验 Bearer token。
func (s *Server) requireAuth(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == "" || s.auth.ValidateToken(tok) != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		h(w, r)
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// resolveRange 从 query 解析周期 → [start, end) 半开区间（time.Time，按 loc 时区的"天"边界）。
// 返回 time.Time 而非字符串，便于上层据此推算"上一等长周期"（见 timeutil.PreviousRange）；
// 落到查询时再用 timeutil.DateString 转 YYYY-MM-DD。
func (s *Server) resolveRange(r *http.Request) (start, end time.Time, err error) {
	period := r.URL.Query().Get("period")
	if period == "custom" {
		return timeutil.CustomRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"), s.loc)
	}
	if period == "" {
		period = "7d"
	}
	start, end = timeutil.PeriodRange(period, time.Now(), s.loc)
	return start, end, nil
}
