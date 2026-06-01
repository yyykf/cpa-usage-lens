import { useState } from 'react'
import { AlertCircle, RefreshCw } from 'lucide-react'
import { Panel, Kicker } from './dashboard/Primitives'
import { PulseDot } from './dashboard/LivePulse'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { formatBytes, formatLag, formatClock, formatInt } from '@/lib/format'
import type { CollectorHealth } from '../types'

interface Props {
  health: CollectorHealth
  loading: boolean
  onRefreshPrices?: () => void | Promise<void>
}

const STATUS: Record<CollectorHealth['status'], { tone: 'success' | 'error' | 'idle'; label: string }> = {
  running: { tone: 'success', label: '采集中 · 健康' },
  error: { tone: 'error', label: '异常' },
  stale: { tone: 'idle', label: '空闲' },
}

export default function CollectorHealthCard({ health, loading, onRefreshPrices }: Props) {
  const [refreshing, setRefreshing] = useState(false)
  const status = STATUS[health.status]
  const lag = health.lagSeconds === null ? '—' : formatLag(health.lagSeconds)

  const handleRefresh = async () => {
    if (!onRefreshPrices) return
    try {
      setRefreshing(true)
      await onRefreshPrices()
    } finally {
      setRefreshing(false)
    }
  }

  return (
    <Panel className="flex h-full flex-col px-5 pb-5 pt-4">
      <Kicker className="mb-3.5">02 — 采集器</Kicker>

      <div className="mb-4 flex items-center gap-2.5 text-sm font-medium text-foreground">
        <PulseDot tone={status.tone} />
        <span className={cn(status.tone === 'error' && 'text-destructive')}>{loading ? '加载中…' : status.label}</span>
      </div>

      <dl className="flex flex-col">
        <Row k="采集延迟" loading={loading}>
          <span className={cn(status.tone === 'success' && 'text-data-success')}>{lag}</span>
        </Row>
        <Row k="已采集" loading={loading}>
          {formatInt(health.eventsIngested)} 条
        </Row>
        <Row k="游标时间" loading={loading}>
          {formatClock(health.lastEventTs)}
        </Row>
        <Row k="数据库占用" loading={loading}>
          <span className="inline-flex items-baseline gap-1.5">
            明细 {formatBytes(health.hotBytes)}
            <span className="text-faint">/</span>
            聚合 {formatBytes(health.dailyBytes)}
          </span>
        </Row>
      </dl>

      {!loading && health.lastError !== '' && (
        <div className="mt-3 flex items-center gap-1.5 truncate text-xs text-destructive" title={health.lastError}>
          <AlertCircle className="size-3.5 shrink-0" />
          <span className="truncate">{health.lastError}</span>
        </div>
      )}

      <div className="mt-auto pt-4">
        <Button type="button" variant="outline" className="w-full" onClick={handleRefresh} disabled={refreshing}>
          <RefreshCw className={cn('size-4', refreshing && 'animate-spin')} />
          {refreshing ? '刷新中…' : '刷新价格表'}
        </Button>
      </div>
    </Panel>
  )
}

function Row({ k, children, loading }: { k: string; children: React.ReactNode; loading: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 border-t border-border-soft py-2.5">
      <dt className="text-[12.5px] text-muted-foreground">{k}</dt>
      <dd className="font-num text-[13px] text-foreground">
        {loading ? <span className="inline-block h-3.5 w-16 animate-pulse rounded bg-muted align-middle" /> : children}
      </dd>
    </div>
  )
}
