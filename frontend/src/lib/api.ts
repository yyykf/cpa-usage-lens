import type { Overview, AccountUsage, TrendPoint, CollectorHealth, Period, CustomRange } from '../types'

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
    throw new Error('未授权，请重新登录')
  }
  if (!res.ok) {
    const msg = await res.json().catch(() => null)
    throw new Error(msg?.error ?? `请求失败 HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

// 把周期 + 自定义范围转成查询串
export function periodQuery(period: Period, custom?: CustomRange): string {
  if (period === 'custom' && custom) {
    return `period=custom&from=${custom.from}&to=${custom.to}`
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
export const getCollector = () => req<CollectorHealth>('/api/collector')
export const refreshPrices = () => req<{ status: string }>('/api/prices/refresh', { method: 'POST' })
