import { Activity, AlertCircle, Database, RefreshCw } from 'lucide-react'
import type { CollectorHealth } from '../types'

interface CollectorHealthProps {
  health: CollectorHealth
  onRefreshPrices?: () => void
}

// 把字节数格式化为 B / KB / MB（保留一位小数，整数不带小数点）
function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  const kb = n / 1024
  if (kb < 1024) return `${kb % 1 === 0 ? kb : kb.toFixed(1)} KB`
  const mb = kb / 1024
  return `${mb % 1 === 0 ? mb : mb.toFixed(1)} MB`
}

const STATUS_CONFIG: Record<
  CollectorHealth['status'],
  { dot: string; label: string }
> = {
  running: { dot: 'bg-data-success', label: '采集中' },
  error: { dot: 'bg-destructive', label: '异常' },
  stale: { dot: 'bg-muted-foreground', label: '空闲' },
}

export default function CollectorHealth({
  health,
  onRefreshPrices,
}: CollectorHealthProps) {
  const status = STATUS_CONFIG[health.status]
  const lag: string =
    health.lagSeconds === null ? '—' : `${health.lagSeconds}s 前`

  return (
    <div className="rounded-2xl border border-border bg-card p-5 md:p-6 hover:border-primary/40 transition-colors">
      <div className="mb-4 flex items-center gap-2">
        <Activity className="w-5 h-5 text-muted-foreground" />
        <h3 className="text-base font-semibold text-foreground">采集器健康</h3>
      </div>

      <div className="mb-4 flex items-center gap-2">
        <span
          className={`inline-block w-2.5 h-2.5 rounded-full ${status.dot}`}
        />
        <span className="text-sm text-foreground">{status.label}</span>
      </div>

      <dl className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <dt className="text-sm text-muted-foreground">采集延迟</dt>
          <dd className="font-num text-sm text-foreground">{lag}</dd>
        </div>
        <div className="flex items-center justify-between gap-3">
          <dt className="text-sm text-muted-foreground">已采集</dt>
          <dd className="font-num text-sm text-foreground">
            {health.eventsIngested.toLocaleString()} 条
          </dd>
        </div>
        <div className="flex items-center justify-between gap-3">
          <dt className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <Database className="w-4 h-4" />
            数据库占用
          </dt>
          <dd className="font-num text-sm text-foreground">
            明细 {formatBytes(health.hotBytes)} / 聚合{' '}
            {formatBytes(health.dailyBytes)}
          </dd>
        </div>
      </dl>

      {health.lastError !== '' && (
        <div
          className="mt-4 flex items-center gap-1.5 text-destructive text-sm truncate"
          title={health.lastError}
        >
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span className="truncate">{health.lastError}</span>
        </div>
      )}

      <button
        type="button"
        onClick={onRefreshPrices}
        className="mt-5 inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground hover:border-primary/40 hover:text-primary transition-colors"
      >
        <RefreshCw className="w-4 h-4" />
        刷新价格表
      </button>
    </div>
  )
}
