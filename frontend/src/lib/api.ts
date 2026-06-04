import type {
  Overview,
  AccountUsage,
  TrendPoint,
  CollectorHealth,
  ModelBreakdown,
  ModelMetric,
  Period,
  CustomRange,
} from '../types'

const BASE = import.meta.env.VITE_API_BASE ?? ''
const TOKEN_KEY = 'cpalens_token'

export const getToken = () => localStorage.getItem(TOKEN_KEY)
export const setToken = (t: string) => localStorage.setItem(TOKEN_KEY, t)
export const clearToken = () => localStorage.removeItem(TOKEN_KEY)
export const isAuthed = () => !!getToken()

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const tok = getToken()
  const res = await fetch(BASE + path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(tok ? { Authorization: `Bearer ${tok}` } : {}),
      ...(init?.headers ?? {}),
    },
  })
  if (res.status === 401) {
    clearToken()
    // App 的 authed 仅初次读 localStorage，token 过期后需整页刷新才能回到登录页
    if (typeof window !== 'undefined') window.location.reload()
    throw new Error('未授权，请重新登录')
  }
  if (!res.ok) {
    const msg = await res.json().catch(() => null)
    throw new Error(msg?.error ?? `请求失败 HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

// 把周期 + 自定义范围转成查询串。
// custom 周期必须带 from/to，否则后端报"无效周期参数"——缺失时回退到 7d 兜底。
export function periodQuery(period: Period, custom?: CustomRange): string {
  if (period === 'custom') {
    if (custom?.from && custom?.to) {
      return `period=custom&from=${encodeURIComponent(custom.from)}&to=${encodeURIComponent(custom.to)}`
    }
    return 'period=7d'
  }
  return `period=${period}`
}

export async function login(password: string): Promise<void> {
  const r = await req<{ token: string }>('/api/login', {
    method: 'POST',
    body: JSON.stringify({ password }),
  })
  setToken(r.token)
}

export const getOverview = (q: string) => req<Overview>(`/api/overview?${q}`)
export const getAccounts = (q: string) => req<AccountUsage[]>(`/api/accounts?${q}`)
export const getTrend = (q: string) => req<TrendPoint[]>(`/api/trend?${q}`)
// metric 决定 ranking 排序口径（默认 token）；ranking 每项同时含 tokens 与 cost，
// 前端切口径通常就地重排即可，无需带 metric 二次请求。
export const getModels = (q: string, metric?: ModelMetric) =>
  req<ModelBreakdown>(`/api/models?${q}${metric ? `&metric=${metric}` : ''}`)
export const getCollector = () => req<CollectorHealth>('/api/collector')
export const refreshPrices = () => req<{ status: string }>('/api/prices/refresh', { method: 'POST' })
