import { useMemo } from 'react'
import { Empty } from './dashboard/Empty'
import { modelColor } from '@/lib/charts'
import { formatCompact, formatCostCompact, formatPercent } from '@/lib/format'
import type { ModelMetric, ModelRankItem } from '../types'

// 模型总量排行（水平条形）：按周期内总量降序，看「整段时间谁用得最多」。
// 用纯 div 画条（KISS，不引 recharts/新依赖）；条宽 = 值 / 最大值，占比 = 值 / 总和。
// 支持 Token / 成本 两种口径就地重排（每项 tokens 与 cost 都在，无需二次请求）。

// 取某项在当前口径下的「值」：token 口径用 tokens；cost 口径用 cost（缺价记为 null）。
function metricValue(item: ModelRankItem, metric: ModelMetric): number | null {
  return metric === 'token' ? item.tokens : item.cost
}

// 前端就地重排（与后端契约「排序细节」一致，保证切口径结果确定）：
//   token：按 tokens 降序，相同按 model 字典序；
//   cost：按 cost 降序，缺价(null)一律末尾，成本相同按 tokens 降序再字典序。
function sortByMetric(ranking: ModelRankItem[], metric: ModelMetric): ModelRankItem[] {
  const arr = [...ranking]
  arr.sort((a, b) => {
    if (metric === 'token') {
      if (b.tokens !== a.tokens) return b.tokens - a.tokens
      return a.model.localeCompare(b.model)
    }
    // cost 口径：缺价排末尾。
    const an = a.cost == null
    const bn = b.cost == null
    if (an !== bn) return an ? 1 : -1
    if (!an && !bn && a.cost !== b.cost) return (b.cost as number) - (a.cost as number)
    if (b.tokens !== a.tokens) return b.tokens - a.tokens
    return a.model.localeCompare(b.model)
  })
  return arr
}

export default function ModelRankChart({
  ranking,
  metric,
  loading,
}: {
  ranking: ModelRankItem[]
  metric: ModelMetric
  loading: boolean
}) {
  // 颜色按「总 token 降序」的稳定索引取（与每日堆叠柱图例一致）：先建 model → 色 映射。
  const colorOf = useMemo(() => {
    const byToken = [...ranking].sort((a, b) =>
      b.tokens !== a.tokens ? b.tokens - a.tokens : a.model.localeCompare(b.model),
    )
    return new Map(byToken.map((it, i) => [it.model, modelColor(i)]))
  }, [ranking])

  const rows = useMemo(() => sortByMetric(ranking, metric), [ranking, metric])

  // 条宽基准（最大值）与占比分母（总和）。缺价项按 0 计入，既不撑条也不影响他人占比分母。
  const values = rows.map((it) => metricValue(it, metric) ?? 0)
  const max = Math.max(...values, 0)
  const total = values.reduce((s, v) => s + v, 0)

  const isEmpty = rows.length === 0

  if (loading) {
    return <div className="h-[200px] w-full animate-pulse rounded-md bg-muted/60" />
  }
  if (isEmpty) {
    return <Empty className="h-[200px]" />
  }

  return (
    <div className="flex flex-col gap-3.5 px-2 pb-2 pt-1">
      {rows.map((it) => {
        const v = metricValue(it, metric)
        const known = v != null
        const width = known && max > 0 ? (v / max) * 100 : 0
        const figure =
          metric === 'token' ? formatCompact(it.tokens) : known ? formatCostCompact(v as number) : '未知'
        const pct = known ? formatPercent(v as number, total) : '—'

        return (
          <div key={it.model} className="grid grid-cols-[110px_1fr_120px] items-center gap-3.5 md:grid-cols-[140px_1fr_130px]">
            <span className="truncate font-mono text-[13px] text-foreground" title={it.model}>
              {it.model}
            </span>
            <div className="h-[18px] overflow-hidden rounded-[5px] bg-background">
              <div
                className="h-full rounded-[5px] transition-[width] duration-300"
                style={{ width: `${width}%`, background: colorOf.get(it.model) }}
              />
            </div>
            <span className="text-right font-num text-[12.5px]">
              <span className={known ? 'text-foreground' : 'text-faint'}>{figure}</span>
              <span className="ml-2 text-faint">{pct}</span>
            </span>
          </div>
        )
      })}
    </div>
  )
}
