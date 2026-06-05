// Package model 定义跨层共享的核心类型（CPA 队列事件、DB 行、API 响应 DTO）。
// 这是各模块与前端的数据契约：字段与 supabase migration、CPA v7.1.31 队列字段一一对应。
package model

import "time"

// Tokens 一次请求的 token 拆分，对齐 CPA v7.1.31 队列 tokens 对象。
type Tokens struct {
	Input         int64 `json:"input_tokens"`
	Output        int64 `json:"output_tokens"`
	Reasoning     int64 `json:"reasoning_tokens"`
	Cached        int64 `json:"cached_tokens"`
	CacheRead     int64 `json:"cache_read_tokens"`
	CacheCreation int64 `json:"cache_creation_tokens"`
	Total         int64 `json:"total_tokens"`
}

// UsageEvent 写入 request_events_hot 的一条精简明细
// （已剥离 api_key / response_headers / fail.body 等敏感或大字段）。
// KeyFingerprint/KeyMask 为客户端 api_key 的不可逆脱敏形态——明文绝不进本结构、绝不入库。
type UsageEvent struct {
	RequestID       string
	EventTS         time.Time
	Source          string
	AuthIndex       string
	Provider        string
	Model           string
	Alias           string
	Endpoint        string
	AuthType        string
	KeyFingerprint  string // sha256(明文 api_key) 全长小写 hex；非 key 认证/空 key 落 'none'
	KeyMask         string // 展示掩码（前缀…后4位）；非 key 认证/空 key 落 '(no key)'
	Tokens          Tokens
	LatencyMs       *int32
	TTFTMs          *int32
	Failed          bool
	FailStatusCode  *int32
	ReasoningEffort string
	ServiceTier     string
}

// DailyUsage 对应 daily_account_usage 一行（账号+模型+key+天 聚合）。
// KeyFingerprint/KeyMask 为脱敏的客户端 api_key 维度（key 是独立视角，与账号正交）。
type DailyUsage struct {
	UsageDate      time.Time
	Source         string
	Model          string
	KeyFingerprint string // sha256 指纹；'none' = 非 key 认证/未知归属
	KeyMask        string // 展示掩码，随指纹带出
	Requests       int64
	FailedRequests int64
	Tokens         Tokens
}

// ModelPrice 对应 model_prices 一行（LiteLLM 每 token USD 单价；nil = 缺该项价格）。
type ModelPrice struct {
	Model                     string
	InputCostPerToken         *float64
	OutputCostPerToken        *float64
	CacheReadCostPerToken     *float64
	CacheCreationCostPerToken *float64
	Currency                  string
	Source                    string
	UpdatedAt                 time.Time
}

// CollectorState 对应 collector_state 单行（采集器游标 + 健康）。
type CollectorState struct {
	LastPollAt     *time.Time
	LastEventTS    *time.Time
	LastRequestID  string
	EventsIngested int64
	LastError      string
	LastErrorAt    *time.Time
	UpdatedAt      time.Time
}

// ---------- API DTO（前端契约，JSON） ----------

// Overview 顶部总览（周期内汇总）。Cost 为 nil 表示存在缺价模型 → 前端显示"未知"。
// 在总 token 之外额外透出 token 拆分，供前端做拆分维度可视化。
// Previous 为与本周期紧邻且等长的上一周期汇总，供 KPI 环比角标使用；
// HasPrevious=false 表示上一周期完全无数据（无可比基准，前端不应展示百分比，
// 区别于"上一周期有数据但某指标为 0"）。后端只下发两段绝对值，
// 不在后端算百分比——避免除 0 得 ↑∞，具体兜底呈现由前端按设计稿决定。
type Overview struct {
	Requests            int64            `json:"requests"`
	Tokens              int64            `json:"tokens"`
	Cost                *float64         `json:"cost"`
	Failed              int64            `json:"failed"`
	InputTokens         int64            `json:"inputTokens"`
	OutputTokens        int64            `json:"outputTokens"`
	ReasoningTokens     int64            `json:"reasoningTokens"`
	CachedTokens        int64            `json:"cachedTokens"`
	CacheReadTokens     int64            `json:"cacheReadTokens"`
	CacheCreationTokens int64            `json:"cacheCreationTokens"`
	HasPrevious         bool             `json:"hasPrevious"`
	Previous            *OverviewCompare `json:"previous"`
}

// OverviewCompare 上一等长周期的可比指标（仅四个 KPI 维度，不含 token 拆分）。
// Cost 为 nil 表示上一周期成本未知（缺价模型）→ 前端对成本环比走"未知"兜底。
type OverviewCompare struct {
	Requests int64    `json:"requests"`
	Tokens   int64    `json:"tokens"`
	Cost     *float64 `json:"cost"`
	Failed   int64    `json:"failed"`
}

// AccountUsage 账号用量榜一行（核心模块）。同样透出 token 拆分。
type AccountUsage struct {
	Source              string   `json:"source"`
	Requests            int64    `json:"requests"`
	Tokens              int64    `json:"tokens"`
	Cost                *float64 `json:"cost"`
	Failed              int64    `json:"failed"`
	InputTokens         int64    `json:"inputTokens"`
	OutputTokens        int64    `json:"outputTokens"`
	ReasoningTokens     int64    `json:"reasoningTokens"`
	CachedTokens        int64    `json:"cachedTokens"`
	CacheReadTokens     int64    `json:"cacheReadTokens"`
	CacheCreationTokens int64    `json:"cacheCreationTokens"`
}

// KeyUsage API key 用量榜一行（与账号榜并列的独立维度，按脱敏 key 聚合）。
// Fingerprint 为聚合主键（前端唯一标识/做 React key）；KeyMask 供界面展示（如 sk-…2216）。
// 指标口径（请求/失败/token 拆分/成本）与 AccountUsage 完全对齐（DRY）。
type KeyUsage struct {
	Fingerprint         string   `json:"fingerprint"`
	KeyMask             string   `json:"keyMask"`
	Requests            int64    `json:"requests"`
	Tokens              int64    `json:"tokens"`
	Cost                *float64 `json:"cost"`
	Failed              int64    `json:"failed"`
	InputTokens         int64    `json:"inputTokens"`
	OutputTokens        int64    `json:"outputTokens"`
	ReasoningTokens     int64    `json:"reasoningTokens"`
	CachedTokens        int64    `json:"cachedTokens"`
	CacheReadTokens     int64    `json:"cacheReadTokens"`
	CacheCreationTokens int64    `json:"cacheCreationTokens"`
}

// TrendPoint 每日趋势一个点（Date 为按配置时区的 YYYY-MM-DD）。
type TrendPoint struct {
	Date     string   `json:"date"`
	Requests int64    `json:"requests"`
	Tokens   int64    `json:"tokens"`
	Cost     *float64 `json:"cost"`
	Failed   int64    `json:"failed"`
}

// CollectorHealth 采集器健康 + 数据库真实容量（绝对值，不显示百分比）。
type CollectorHealth struct {
	Status         string     `json:"status"` // running | stale | error
	LastPollAt     *time.Time `json:"lastPollAt"`
	LagSeconds     *int64     `json:"lagSeconds"`
	LastEventTS    *time.Time `json:"lastEventTs"`
	EventsIngested int64      `json:"eventsIngested"`
	LastError      string     `json:"lastError"`
	HotBytes       int64      `json:"hotBytes"`
	DailyBytes     int64      `json:"dailyBytes"`
}

// ModelBreakdown 模型用量分布。
// Models/Daily 给前端「每日 100% 堆叠柱」；Ranking 给「模型总量排行」（水平条形）。
type ModelBreakdown struct {
	Models  []string          `json:"models"`  // 周期内出现过的模型，按总 token 降序（决定堆叠顺序/图例/配色索引）
	Daily   []ModelDailyPoint `json:"daily"`   // 每天一项，日期升序
	Ranking []ModelRankItem   `json:"ranking"` // 周期内各模型总量排行，按当前口径(metric)降序
	Metric  string            `json:"metric"`  // 本次排行口径："token" | "cost"（决定 Ranking 的排序依据）
}

// ModelRankItem 模型总量排行的一项。
// Tokens 始终给（周期内该模型总 token）；Cost 为 nil 表示该模型缺价 → 成本未知。
// 两个口径的值都返回，前端切换 token/cost 仅改排序与展示，无需二次请求。
type ModelRankItem struct {
	Model  string   `json:"model"`
	Tokens int64    `json:"tokens"`
	Cost   *float64 `json:"cost"`
}

// ModelDailyPoint 模型分布的某一天（按模型透视的 token）。
type ModelDailyPoint struct {
	Date   string           `json:"date"`   // YYYY-MM-DD（按配置时区的"天"）
	Tokens map[string]int64 `json:"tokens"` // model -> 当天 total_tokens（仅含当天有数据的模型）
}
