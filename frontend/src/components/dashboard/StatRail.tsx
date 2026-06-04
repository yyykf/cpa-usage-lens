import { cn } from '@/lib/utils'
import { Panel } from './Primitives'
import { Sparkline } from './Sparkline'
import { DeltaBadge } from './DeltaBadge'
import { CHART_COLORS } from '@/lib/charts'
import { formatInt, formatCost, formatPercent } from '@/lib/format'
import { overviewDeltas, type DeltaResult } from '@/lib/delta'
import type { Overview, TrendPoint } from '../../types'

// 横向指标台：4 个指标并排，竖线分隔（破除四宫格 KPI 的 AI 味）。
// 每个指标在绝对值旁带「环比上一等长周期」角标（无基准/成本未知走兜底占位）。
export default function StatRail({
  overview,
  trend,
  loading,
}: {
  overview: Overview
  trend: TrendPoint[]
  loading: boolean
}) {
  const tokenSeries = trend.map((p) => p.tokens)
  const failRate = formatPercent(overview.failed, overview.requests)
  const deltas = overviewDeltas(overview)

  return (
    <Panel className="grid grid-cols-2 lg:grid-cols-4">
      <Stat
        dotColor={CHART_COLORS.requests}
        label="总请求"
        value={formatInt(overview.requests)}
        delta={deltas.requests}
        loading={loading}
        meta={<span className="text-faint">周期内累计</span>}
      />
      <Stat
        dotColor={CHART_COLORS.tokens}
        label="总 Token"
        value={formatInt(overview.tokens)}
        valueColor="text-foreground"
        delta={deltas.tokens}
        loading={loading}
        meta={tokenSeries.length > 0 ? <Sparkline values={tokenSeries} color={CHART_COLORS.tokens} /> : undefined}
      />
      <Stat
        dotColor={CHART_COLORS.cost}
        label="总成本"
        value={formatCost(overview.cost)}
        valueColor={overview.cost === null ? 'text-muted-foreground' : 'text-[hsl(var(--data-cost))]'}
        delta={deltas.cost}
        loading={loading}
        meta={<span className="text-faint">含缓存折扣</span>}
      />
      <Stat
        dotColor={CHART_COLORS.failed}
        label="失败数"
        value={formatInt(overview.failed)}
        valueColor={overview.failed > 0 ? 'text-destructive' : 'text-foreground'}
        delta={deltas.failed}
        loading={loading}
        meta={<span className={cn(overview.failed > 0 ? 'text-destructive/80' : 'text-faint')}>失败率 {failRate}</span>}
      />
    </Panel>
  )
}

function Stat({
  dotColor,
  label,
  value,
  valueColor = 'text-foreground',
  delta,
  meta,
  loading,
}: {
  dotColor: string
  label: string
  value: string
  valueColor?: string
  delta?: DeltaResult
  meta?: React.ReactNode
  loading: boolean
}) {
  return (
    <div className="relative px-5 py-5 not-first:before:absolute not-first:before:inset-y-[18px] not-first:before:left-0 not-first:before:w-px not-first:before:bg-border-soft">
      <div className="flex items-center gap-2 font-mono text-[11.5px] uppercase tracking-[0.16em] text-muted-foreground">
        <span aria-hidden className="size-1.5 rounded-sm" style={{ background: dotColor }} />
        {label}
      </div>
      {loading ? (
        <div className="mt-3 h-[30px] w-24 animate-pulse rounded bg-muted" />
      ) : (
        <div className="mt-3 flex items-baseline gap-2.5">
          <span className={cn('font-num text-[30px] font-semibold leading-none tracking-tight', valueColor)}>{value}</span>
          {delta && <DeltaBadge delta={delta} />}
        </div>
      )}
      <div className="mt-2.5 flex h-[22px] items-center font-mono text-xs">{!loading && meta}</div>
    </div>
  )
}
