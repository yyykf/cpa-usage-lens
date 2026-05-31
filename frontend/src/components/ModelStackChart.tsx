import { useMemo } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { Panel, PanelHeader } from './dashboard/Primitives'
import { Empty } from './dashboard/Empty'
import { CHART_COLORS, modelColor } from '@/lib/charts'
import { formatInt, formatDateShort, formatPercent } from '@/lib/format'
import type { ModelBreakdown } from '../types'

// 把 daily 的 { date, tokens:{model:n} } 摊平成 { date, [model]: n }，缺失模型补 0（堆叠需要每行字段齐全）。
function flatten(data: ModelBreakdown): Record<string, number | string>[] {
  return data.daily.map((d) => {
    const row: Record<string, number | string> = { date: d.date }
    for (const m of data.models) row[m] = d.tokens[m] ?? 0
    return row
  })
}

export default function ModelStackChart({ data, loading }: { data: ModelBreakdown; loading: boolean }) {
  const rows = useMemo(() => flatten(data), [data])
  const models = data.models
  const colorOf = useMemo(() => new Map(models.map((m, i) => [m, modelColor(i)])), [models])
  const isEmpty = models.length === 0 || rows.length === 0

  const legend = (
    <div className="flex max-w-[60%] flex-wrap justify-end gap-x-3.5 gap-y-1.5 text-xs">
      {models.map((m) => (
        <div key={m} className="flex items-center gap-1.5 text-muted-foreground">
          <span aria-hidden className="size-2.5 rounded-sm" style={{ background: colorOf.get(m) }} />
          <span className="max-w-[120px] truncate" title={m}>
            {m}
          </span>
        </div>
      ))}
    </div>
  )

  return (
    <Panel>
      <PanelHeader
        title={
          <span className="flex items-center gap-2">
            每日模型用量
            <span className="font-mono text-[11px] tracking-[0.1em] text-faint">100% · BY TOKEN</span>
          </span>
        }
        barColor={CHART_COLORS.tokens}
        right={!loading && !isEmpty ? legend : undefined}
      />
      <div className="px-2 pb-3 pt-1.5">
        {loading ? (
          <div className="h-[220px] w-full animate-pulse rounded-md bg-muted/60" />
        ) : isEmpty ? (
          <Empty className="h-[220px]" />
        ) : (
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={rows} stackOffset="expand" margin={{ top: 8, right: 12, bottom: 4, left: 4 }} barCategoryGap="22%">
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
                tickFormatter={(v: number) => `${Math.round(v * 100)}%`}
                tick={{ fill: CHART_COLORS.axis, fontSize: 10, fontFamily: 'Fira Code' }}
                tickLine={false}
                axisLine={false}
                width={40}
                ticks={[0, 0.5, 1]}
              />
              <Tooltip content={<ModelTooltip models={models} colorOf={colorOf} />} cursor={{ fill: 'hsl(var(--foreground) / 0.04)' }} />
              {models.map((m, i) => (
                <Bar
                  key={m}
                  dataKey={m}
                  stackId="models"
                  fill={colorOf.get(m)}
                  stroke="hsl(var(--background))"
                  strokeWidth={2}
                  // 顶部段（最后声明）给柱顶圆角。
                  radius={i === models.length - 1 ? [3, 3, 0, 0] : undefined}
                  isAnimationActive={false}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>
    </Panel>
  )
}

interface TooltipItem {
  payload: Record<string, number | string>
}

function ModelTooltip({
  active,
  payload,
  label,
  models,
  colorOf,
}: {
  active?: boolean
  payload?: TooltipItem[]
  label?: string
  models: string[]
  colorOf: Map<string, string>
}) {
  if (!active || !payload?.length) return null
  const row = payload[0].payload
  const total = models.reduce((s, m) => s + (Number(row[m]) || 0), 0)
  // 只显示当天有数据的模型，按 token 降序。
  const items = models
    .map((m) => ({ model: m, value: Number(row[m]) || 0 }))
    .filter((it) => it.value > 0)
    .sort((a, b) => b.value - a.value)

  return (
    <div className="min-w-[180px] rounded-lg border border-border bg-popover px-3 py-2 shadow-xl">
      <div className="mb-1.5 font-mono text-xs text-foreground">{label}</div>
      <div className="flex flex-col gap-1">
        {items.map((it) => (
          <div key={it.model} className="flex items-center justify-between gap-4 text-[11px]">
            <span className="flex items-center gap-1.5 text-muted-foreground">
              <span aria-hidden className="size-2 rounded-sm" style={{ background: colorOf.get(it.model) }} />
              <span className="max-w-[110px] truncate" title={it.model}>
                {it.model}
              </span>
            </span>
            <span className="flex items-center gap-2 font-num">
              <span className="text-faint">{formatInt(it.value)}</span>
              <span className="w-10 text-right text-foreground">{formatPercent(it.value, total)}</span>
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
