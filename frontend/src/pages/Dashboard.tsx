import { useState, useEffect, useCallback, useRef } from 'react'
import { LogOut } from 'lucide-react'
import { toast } from 'sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/sonner'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import PeriodSwitcher from '../components/PeriodSwitcher'
import StatRail from '../components/dashboard/StatRail'
import TokenComposition from '../components/dashboard/TokenComposition'
import TrendChart from '../components/TrendChart'
import CollectorHealthCard from '../components/CollectorHealth'
import ModelUsagePanel from '../components/ModelUsagePanel'
import AccountTable from '../components/AccountTable'
import KeyTable from '../components/KeyTable'
import { LiveBadge } from '../components/dashboard/LivePulse'
import RefreshSelector from '../components/dashboard/RefreshSelector'
import { Kicker } from '../components/dashboard/Primitives'
import { useAutoRefresh, refreshLabel } from '../hooks/useAutoRefresh'
import {
  getOverview,
  getAccounts,
  getKeys,
  getTrend,
  getModels,
  getCollector,
  refreshPrices,
  periodQuery,
  clearToken,
} from '../lib/api'
import type {
  Period,
  CustomRange,
  Overview,
  AccountUsage,
  KeyUsage,
  TrendPoint,
  ModelBreakdown,
  CollectorHealth as CollectorHealthData,
} from '../types'

const EMPTY_OVERVIEW: Overview = {
  requests: 0,
  tokens: 0,
  cost: null,
  failed: 0,
  inputTokens: 0,
  outputTokens: 0,
  reasoningTokens: 0,
  cachedTokens: 0,
  cacheReadTokens: 0,
  cacheCreationTokens: 0,
  hasPrevious: false,
  previous: null,
}

const EMPTY_MODELS: ModelBreakdown = { models: [], daily: [], metric: 'token', ranking: [] }

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

const PERIOD_LABEL: Record<Period, string> = {
  today: '今天',
  '7d': '近 7 天',
  '30d': '近 30 天',
  custom: '自定义范围',
}

export default function Dashboard({ onLogout }: { onLogout: () => void }) {
  const [period, setPeriod] = useState<Period>('7d')
  const [custom, setCustom] = useState<CustomRange | null>(null)
  const [overview, setOverview] = useState<Overview | null>(null)
  const [accounts, setAccounts] = useState<AccountUsage[]>([])
  const [keys, setKeys] = useState<KeyUsage[]>([])
  const [trend, setTrend] = useState<TrendPoint[]>([])
  const [models, setModels] = useState<ModelBreakdown>(EMPTY_MODELS)
  const [collector, setCollector] = useState<CollectorHealthData | null>(null)
  const [loading, setLoading] = useState<boolean>(true)

  // 请求序号：每次 loadData 自增，回包后只有仍是最新序号才 setState。
  // 防止慢的旧响应覆盖新数据（切周期瞬间闪旧数据、或多轮轮询重叠时的根因）。
  const reqIdRef = useRef(0)
  // 轮询 in-flight 锁：仅约束自动刷新（silent），上一轮没结束就跳过本 tick、不堆积。
  const inFlightRef = useRef(false)
  // silent 失败去重：记上一次 silent 是否已处于失败态，避免 API 临时挂时每 tick 刷屏。
  const silentErroredRef = useRef(false)

  // silent: 自动刷新轮询时为 true，不触发骨架屏（避免每隔几秒全屏闪一下），仅静默替换数据。
  const loadData = useCallback(
    async (silent = false): Promise<void> => {
      // 自动刷新：上一轮还没回来就跳过本轮，避免慢网络/5s 档多轮重叠。
      // 手动操作（非 silent）不受此锁约束——用户主动刷新优先。
      if (silent && inFlightRef.current) return

      const reqId = ++reqIdRef.current
      // 锁只服务于 silent 的「跳过重叠轮询」，故只由 silent 路径置位/复位；
      // 手动请求不碰它，避免手动在途时错误复位、削弱后续轮询的跳过判断。
      if (silent) inFlightRef.current = true
      try {
        if (!silent) setLoading(true)
        const q = periodQuery(period, custom ?? undefined)
        const [o, a, k, t, m, c] = await Promise.all([
          getOverview(q),
          getAccounts(q),
          getKeys(q),
          getTrend(q),
          getModels(q),
          getCollector(),
        ])
        // 仅当本次仍是最新请求时才落数据：更晚发起的请求（如切周期）已赢，丢弃旧结果。
        if (reqId !== reqIdRef.current) return
        setOverview(o)
        setAccounts(a)
        setKeys(k)
        setTrend(t)
        setModels(m)
        setCollector(c)
        silentErroredRef.current = false
      } catch (e) {
        // 旧请求的失败也不该打扰用户（已有更新的请求在途）。
        if (reqId !== reqIdRef.current) return
        // silent 失败去重：仅在「上一轮还正常、这轮首次转失败」时弹一次；
        // 持续失败不重复弹，恢复成功后复位，下次再失败可再提示。
        if (silent) {
          if (!silentErroredRef.current) {
            silentErroredRef.current = true
            toast.error(e instanceof Error ? e.message : '自动刷新失败')
          }
        } else {
          toast.error(e instanceof Error ? e.message : '加载失败')
        }
      } finally {
        if (silent) inFlightRef.current = false
        if (!silent) setLoading(false)
      }
    },
    [period, custom],
  )

  useEffect(() => {
    void loadData()
  }, [loadData])

  // 自动刷新：按选中档位静默重拉全部数据（复用 loadData），切换/卸载自动清理定时器。
  const { interval: refreshInterval, setInterval: setRefreshInterval } = useAutoRefresh(() => {
    void loadData(true)
  })

  const handlePeriodChange = (next: Period, range?: CustomRange) => {
    setCustom(range ?? null)
    setPeriod(next)
  }

  const handleLogout = (): void => {
    clearToken()
    onLogout()
  }

  const handleRefreshPrices = async (): Promise<void> => {
    try {
      await refreshPrices()
      await loadData()
      toast.success('价格表已刷新')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '刷新价格表失败')
    }
  }

  const periodSubtitle =
    period === 'custom' && custom ? `${custom.from} ~ ${custom.to}` : PERIOD_LABEL[period]

  return (
    <TooltipProvider delayDuration={200}>
      <div className="min-h-screen">
        <div className="mx-auto max-w-[1320px] px-4 pb-16 md:px-7">
          <header className="flex flex-col gap-4 py-5 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-3.5">
              <div className="grid size-[34px] place-items-center rounded-[9px] border border-[hsl(240_8%_18%)] bg-linear-to-br from-[hsl(220_8%_10%)] to-[hsl(240_9%_4%)] shadow-[inset_0_1px_0_hsl(0_0%_100%/0.06),0_0_20px_hsl(186_31%_50%/0.14)]">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
                  <path d="M3 17l5-6 4 4 4-7 5 8" stroke="hsl(var(--accent))" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                  <circle cx="8" cy="11" r="1.6" fill="hsl(186 28% 57%)" />
                  <circle cx="16" cy="8" r="1.6" fill="hsl(186 28% 57%)" />
                </svg>
              </div>
              <div>
                <h1 className="font-mono text-[15px] font-semibold uppercase tracking-[0.14em] text-foreground">CPA Usage Lens</h1>
                <div className="font-mono text-[11px] uppercase tracking-[0.22em] text-faint">Usage · Cost · Models</div>
              </div>
              <LiveBadge active={refreshInterval !== 0} intervalLabel={refreshLabel(refreshInterval)} />
            </div>

            <div className="flex items-center gap-3">
              <PeriodSwitcher period={period} custom={custom} onChange={handlePeriodChange} />
              <RefreshSelector value={refreshInterval} onChange={setRefreshInterval} />
              <Button type="button" variant="outline" size="sm" onClick={handleLogout}>
                <LogOut className="size-4" />
                退出
              </Button>
            </div>
          </header>

          <Separator className="mb-6 bg-linear-to-r from-transparent via-accent/40 to-transparent" />

          <Kicker>01 — 周期总览 · {periodSubtitle}</Kicker>
          <div className="mb-3.5">
            <StatRail overview={overview ?? EMPTY_OVERVIEW} trend={trend} loading={loading} />
          </div>

          <div className="mb-6">
            <TokenComposition overview={overview ?? EMPTY_OVERVIEW} loading={loading} />
          </div>

          <div className="mb-3.5 grid grid-cols-1 gap-3.5 lg:grid-cols-3">
            <div className="lg:col-span-2">
              <TrendChart data={trend} loading={loading} />
            </div>
            <div className="lg:col-span-1">
              <CollectorHealthCard health={collector ?? EMPTY_COLLECTOR} loading={loading} onRefreshPrices={handleRefreshPrices} />
            </div>
          </div>

          <Kicker>02 — 模型用量 · {periodSubtitle}</Kicker>
          <div className="mb-3.5">
            <ModelUsagePanel data={models} loading={loading} />
          </div>

          <Kicker>03 — 用量榜 · {periodSubtitle}</Kicker>
          <div className="grid grid-cols-1 gap-3.5 lg:grid-cols-2">
            <AccountTable accounts={accounts} loading={loading} />
            <KeyTable keys={keys} loading={loading} />
          </div>
        </div>
      </div>
      <Toaster position="bottom-right" />
    </TooltipProvider>
  )
}
