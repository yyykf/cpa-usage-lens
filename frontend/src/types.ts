// 与后端 internal/model DTO 严格对齐（JSON 字段名一致，camelCase）。

// token 拆分字段（后端在 overview / accounts 上新增的 6 个 int64）。
export interface TokenBreakdown {
  inputTokens: number
  outputTokens: number
  reasoningTokens: number
  cachedTokens: number
  cacheReadTokens: number
  cacheCreationTokens: number
}

// 环比对比的「上一等长周期」可比指标（仅 4 个 KPI 维度）。后端只给绝对值，百分比由前端算。
export interface PreviousPeriod {
  requests: number
  tokens: number
  cost: number | null // null = 上一周期存在缺价模型（成本未知）
  failed: number
}

export interface Overview extends TokenBreakdown {
  requests: number
  tokens: number // 总量（不变）
  cost: number | null // null = 存在缺价模型，显示"未知"
  failed: number
  hasPrevious: boolean // 上一周期是否有任何数据；false = 无可比基准（走兜底）
  previous: PreviousPeriod | null // hasPrevious=false 时为 null
}

export interface AccountUsage extends TokenBreakdown {
  source: string
  requests: number
  tokens: number
  cost: number | null
  failed: number
}

export interface TrendPoint {
  date: string // YYYY-MM-DD
  requests: number
  tokens: number
  cost: number | null
  failed: number
}

// GET /api/models 的单日数据点：tokens 仅含当天有数据的模型。
export interface ModelDailyPoint {
  date: string // YYYY-MM-DD
  tokens: Record<string, number>
}

// 模型用量排序口径：token（按总 token）/ cost（按总成本）。默认 token。
export type ModelMetric = 'token' | 'cost'

// 模型总量排行单项：tokens 与 cost 始终同时返回，切口径前端就地重排即可。
export interface ModelRankItem {
  model: string
  tokens: number
  cost: number | null // null = 该模型缺价（成本未知）
}

// GET /api/models 响应：models 按周期内总 token 降序；daily 按日期升序。
// ranking 按 metric 口径降序；metric 为后端实际生效（已归一化）的口径。
export interface ModelBreakdown {
  models: string[]
  daily: ModelDailyPoint[]
  metric: ModelMetric
  ranking: ModelRankItem[]
}

export interface CollectorHealth {
  status: 'running' | 'stale' | 'error'
  lastPollAt: string | null
  lagSeconds: number | null
  lastEventTs: string | null
  eventsIngested: number
  lastError: string
  hotBytes: number
  dailyBytes: number
}

export type Period = 'today' | '7d' | '30d' | 'custom'

export interface CustomRange {
  from: string // YYYY-MM-DD
  to: string
}
