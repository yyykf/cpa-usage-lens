import { useState } from 'react'
import {
  ComposedChart,
  Area,
  Line,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { Panel, PanelHeader } from './dashboard/Primitives'
import { Empty } from './dashboard/Empty'
import { CHART_COLORS } from '@/lib/charts'
import { formatCompact, formatCostCompact, formatInt, formatCost, formatDateShort } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { TrendPoint } from '../types'

type SeriesKey = 'tokens' | 'cost' | 'requests'

const SERIES: { key: SeriesKey; label: string; color: string; axis: string }[] = [
  { key: 'tokens', label: 'Token（左轴）', color: CHART_COLORS.tokens, axis: '左轴' },
  { key: 'cost', label: '成本（右轴）', color: CHART_COLORS.cost, axis: '右轴' },
  { key: 'requests', label: '请求', color: CHART_COLORS.requests, axis: '' },
]

export default function TrendChart({ data, loading }: { data: TrendPoint[]; loading: boolean }) {
  // 请求线默认隐藏（量级与 token 差异大），点击图例可 toggle。
  const [hidden, setHidden] = useState<Set<SeriesKey>>(new Set(['requests']))
  const toggle = (k: SeriesKey) =>
    setHidden((prev) => {
      const next = new Set(prev)
      next.has(k) ? next.delete(k) : next.add(k)
      return next
    })

  const legend = (
    <div className="flex flex-wrap gap-3.5 text-xs">
      {SERIES.map((s) => (
        <button
          key={s.key}
          type="button"
          onClick={() => toggle(s.key)}
          className={cn('flex items-center gap-1.5 transition-opacity', hidden.has(s.key) ? 'opacity-40' : 'opacity-100')}
        >
          <span aria-hidden className="h-0.5 w-2.5 rounded-sm" style={{ background: s.color }} />
          <span className="text-muted-foreground">{s.label}</span>
        </button>
      ))}
    </div>
  )

  return (
    <Panel className="h-full">
      <PanelHeader title="每日趋势" barColor={CHART_COLORS.tokens} right={legend} />
      <div className="px-2 pb-3 pt-1.5">
        {loading ? (
          <div className="h-[250px] w-full animate-pulse rounded-md bg-muted/60" />
        ) : data.length === 0 ? (
          <Empty className="h-[250px]" />
        ) : (
          <ResponsiveContainer width="100%" height={250}>
            <ComposedChart data={data} margin={{ top: 12, right: 16, bottom: 4, left: 4 }}>
              <defs>
                <linearGradient id="trend-token-fill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={CHART_COLORS.tokens} stopOpacity={0.22} />
                  <stop offset="100%" stopColor={CHART_COLORS.tokens} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke={CHART_COLORS.grid} strokeDasharray="3 5" vertical={false} />
              <XAxis
                dataKey="date"
                tickFormatter={formatDateShort}
                tick={{ fill: CHART_COLORS.axis, fontSize: 10, fontFamily: 'Fira Code' }}
                tickLine={false}
                axisLine={{ stroke: CHART_COLORS.grid }}
                minTickGap={8}
              />
              <YAxis
                yAxisId="token"
                tickFormatter={(v: number) => formatCompact(v)}
                tick={{ fill: CHART_COLORS.axis, fontSize: 10, fontFamily: 'Fira Code' }}
                tickLine={false}
                axisLine={false}
                width={44}
              />
              <YAxis
                yAxisId="cost"
                orientation="right"
                tickFormatter={(v: number) => formatCostCompact(v)}
                tick={{ fill: 'hsl(var(--data-cost))', fontSize: 10, fontFamily: 'Fira Code' }}
                tickLine={false}
                axisLine={false}
                width={48}
              />
              {!hidden.has('requests') && (
                <YAxis yAxisId="req" orientation="right" hide />
              )}
              <Tooltip content={<TrendTooltip hidden={hidden} />} cursor={{ stroke: CHART_COLORS.grid }} />
              {!hidden.has('tokens') && (
                <Area
                  yAxisId="token"
                  type="monotone"
                  dataKey="tokens"
                  stroke={CHART_COLORS.tokens}
                  strokeWidth={2.2}
                  fill="url(#trend-token-fill)"
                  dot={false}
                  activeDot={{ r: 3.4, fill: 'hsl(var(--background))', stroke: CHART_COLORS.tokens, strokeWidth: 2 }}
                />
              )}
              {!hidden.has('cost') && (
                <Line
                  yAxisId="cost"
                  type="monotone"
                  dataKey="cost"
                  stroke={CHART_COLORS.cost}
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 2.6, fill: CHART_COLORS.cost }}
                  connectNulls
                />
              )}
              {!hidden.has('requests') && (
                <Line
                  yAxisId="req"
                  type="monotone"
                  dataKey="requests"
                  stroke={CHART_COLORS.requests}
                  strokeWidth={1.6}
                  dot={false}
                  activeDot={{ r: 2.6, fill: CHART_COLORS.requests }}
                />
              )}
            </ComposedChart>
          </ResponsiveContainer>
        )}
      </div>
    </Panel>
  )
}

interface TooltipPayloadItem {
  dataKey: string
  value: number | null
  payload: TrendPoint
}

function TrendTooltip({ active, payload, hidden }: { active?: boolean; payload?: TooltipPayloadItem[]; hidden: Set<SeriesKey> }) {
  if (!active || !payload?.length) return null
  const point = payload[0].payload
  const rows: { key: SeriesKey; label: string; color: string; text: string }[] = []
  if (!hidden.has('tokens')) rows.push({ key: 'tokens', label: 'Token', color: CHART_COLORS.tokens, text: formatInt(point.tokens) })
  if (!hidden.has('cost')) rows.push({ key: 'cost', label: '成本', color: CHART_COLORS.cost, text: formatCost(point.cost) })
  if (!hidden.has('requests')) rows.push({ key: 'requests', label: '请求', color: CHART_COLORS.requests, text: formatInt(point.requests) })

  return (
    <div className="min-w-36 rounded-lg border border-border bg-popover px-3 py-2 shadow-xl">
      <div className="mb-1.5 font-mono text-xs text-foreground">{point.date}</div>
      <div className="flex flex-col gap-1">
        {rows.map((r) => (
          <div key={r.key} className="flex items-center justify-between gap-4 text-xs">
            <span className="flex items-center gap-1.5 text-muted-foreground">
              <span aria-hidden className="size-2 rounded-sm" style={{ background: r.color }} />
              {r.label}
            </span>
            <span className="font-num text-foreground">{r.text}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
