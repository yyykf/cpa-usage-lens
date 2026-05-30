// 与后端 internal/model DTO 严格对齐（JSON 字段名一致）。

export interface Overview {
  requests: number
  tokens: number
  cost: number | null // null = 存在缺价模型，显示"未知"
  failed: number
}

export interface AccountUsage {
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
