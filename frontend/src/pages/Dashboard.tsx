import { useState, useEffect } from 'react'
import { LogOut } from 'lucide-react'
import PeriodSwitcher from '../components/PeriodSwitcher'
import KpiCards from '../components/KpiCards'
import TrendChart from '../components/TrendChart'
import CollectorHealth from '../components/CollectorHealth'
import AccountTable from '../components/AccountTable'
import {
  getOverview,
  getAccounts,
  getTrend,
  getCollector,
  refreshPrices,
  periodQuery,
  clearToken,
} from '../lib/api'
import type {
  Period,
  Overview,
  AccountUsage,
  TrendPoint,
  CollectorHealth as CollectorHealthData,
} from '../types'

const EMPTY_OVERVIEW: Overview = { requests: 0, tokens: 0, cost: null, failed: 0 }

const EMPTY_COLLECTOR: CollectorHealthData = {
  status: 'error',
  lastPollAt: null,
  lagSeconds: null,
  lastEventTs: null,
  eventsIngested: 0,
  lastError: '',
  hotBytes: 0,
  dailyBytes: 0,
}

export default function Dashboard({ onLogout }: { onLogout: () => void }) {
  const [period, setPeriod] = useState<Period>('7d')
  const [overview, setOverview] = useState<Overview | null>(null)
  const [accounts, setAccounts] = useState<AccountUsage[]>([])
  const [trend, setTrend] = useState<TrendPoint[]>([])
  const [collector, setCollector] = useState<CollectorHealthData | null>(null)
  const [loading, setLoading] = useState<boolean>(false)
  const [error, setError] = useState<string>('')

  const loadData = async (): Promise<void> => {
    try {
      setLoading(true)
      setError('')
      const q = periodQuery(period)
      const [o, a, t, c] = await Promise.all([
        getOverview(q),
        getAccounts(q),
        getTrend(q),
        getCollector(),
      ])
      setOverview(o)
      setAccounts(a)
      setTrend(t)
      setCollector(c)
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [period])

  const handleLogout = (): void => {
    clearToken()
    onLogout()
  }

  const handleRefreshPrices = async (): Promise<void> => {
    await refreshPrices()
    await loadData()
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 md:px-6 py-6">
        <header className="flex justify-between items-center mb-6">
          <div className="flex items-baseline gap-3">
            <h1 className="text-xl md:text-2xl font-semibold text-foreground">
              CPA Usage Lens
            </h1>
            {loading && (
              <span className="text-sm text-muted-foreground">加载中…</span>
            )}
          </div>
          <div className="flex gap-3 items-center">
            <PeriodSwitcher period={period} onChange={setPeriod} />
            <button
              type="button"
              onClick={handleLogout}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-muted-foreground hover:text-foreground hover:border-primary/40 transition-colors"
            >
              <LogOut className="w-4 h-4" />
              退出
            </button>
          </div>
        </header>

        {error && (
          <div className="mb-4 rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {error}
          </div>
        )}

        <div className="mb-4">
          <KpiCards overview={overview ?? EMPTY_OVERVIEW} />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <div className="lg:col-span-2">
            <TrendChart data={trend} />
          </div>
          <div className="lg:col-span-1">
            <CollectorHealth
              health={collector ?? EMPTY_COLLECTOR}
              onRefreshPrices={handleRefreshPrices}
            />
          </div>
          <div className="lg:col-span-3">
            <AccountTable accounts={accounts} />
          </div>
        </div>
      </div>
    </div>
  )
}
