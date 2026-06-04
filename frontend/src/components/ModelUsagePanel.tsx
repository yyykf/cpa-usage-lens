import { useState } from 'react'
import { Panel, PanelHeader } from './dashboard/Primitives'
import ModelRankChart from './ModelRankChart'
import ModelStackChart from './ModelStackChart'
import { cn } from '@/lib/utils'
import type { ModelBreakdown, ModelMetric } from '../types'

// 模型用量卡：同一张卡里切换两种视图（互补）——
//   「排行」= 模型总量排行（水平条，看整段时间谁用得最多）；
//   「每日占比」= 现有每日 100% 堆叠柱（看每天占比变化，复用 ModelStackChart，恒按 token）。
// 「排行」视图下另有 Token / 成本 口径切换（默认 Token），就地重排（每项两值都在，无需二次请求）。
//
// 视图切换段在两个视图都常驻（排行卡的 header / 堆叠柱卡的 headerExtra），保证能来回切。

type View = 'ranking' | 'daily'

export default function ModelUsagePanel({ data, loading }: { data: ModelBreakdown; loading: boolean }) {
  const [view, setView] = useState<View>('ranking')
  // 口径默认跟随后端实际生效口径（通常 token）；用户在前端切换即就地重排。
  const [metric, setMetric] = useState<ModelMetric>(data.metric ?? 'token')

  const isRanking = view === 'ranking'

  const viewSeg = (
    <div className="flex items-center gap-2">
      <span className="font-mono text-[10.5px] tracking-[0.12em] text-faint">视图</span>
      <Seg
        options={[
          { value: 'ranking', label: '排行' },
          { value: 'daily', label: '每日占比' },
        ]}
        value={view}
        onChange={setView}
      />
    </div>
  )

  // 「每日占比」分支复用现有 ModelStackChart（自带标题/图例/空态），仅把视图切换段注入其 header。
  if (!isRanking) {
    return <ModelStackChart data={data} loading={loading} headerExtra={viewSeg} />
  }

  return (
    <Panel>
      <PanelHeader
        title={
          <span className="flex items-center gap-2">
            模型总量排行
            <span className="rounded-[5px] border border-accent/30 bg-accent/[0.12] px-1.5 font-mono text-[10px] font-semibold text-accent">
              NEW
            </span>
          </span>
        }
        barColor="hsl(var(--m1))"
        right={
          <div className="flex flex-wrap items-center gap-2.5">
            {viewSeg}
            <span className="font-mono text-[10.5px] tracking-[0.12em] text-faint">口径</span>
            <Seg
              options={[
                { value: 'token', label: 'Token' },
                { value: 'cost', label: '成本' },
              ]}
              value={metric}
              onChange={setMetric}
            />
          </div>
        }
      />
      <div className="px-2 pb-3 pt-1.5">
        <ModelRankChart ranking={data.ranking} metric={metric} loading={loading} />
      </div>
    </Panel>
  )
}

// 段切换（本卡专用）：对齐设计稿 .seg（边框 + 等宽，激活态青色填充）。
function Seg<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string }[]
  value: T
  onChange: (value: T) => void
}) {
  return (
    <div className="inline-flex overflow-hidden rounded-lg border border-border">
      {options.map((opt) => {
        const on = opt.value === value
        return (
          <button
            key={opt.value}
            type="button"
            onClick={() => onChange(opt.value)}
            className={cn(
              'px-2.5 py-[5px] font-mono text-xs transition-colors',
              on ? 'bg-accent/[0.14] text-accent' : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {opt.label}
          </button>
        )
      })}
    </div>
  )
}
