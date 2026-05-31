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

export interface Overview extends TokenBreakdown {
  requests: number
  tokens: number // 总量（不变）
  cost: number | null // null = 存在缺价模型，显示"未知"
  failed: number
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

// GET /api/models 响应：models 按周期内总 token 降序；daily 按日期升序。
export interface ModelBreakdown {
  models: string[]
  daily: ModelDailyPoint[]
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
