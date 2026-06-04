// KPI 环比（与上一等长周期对比）计算与兜底，全站单一来源（DRY）。
//
// 后端只下发本期绝对值 + 上一周期绝对值（previous）+ hasPrevious 标记，不算百分比。
// 前端按契约决定呈现：仅当「有可比基准 且 上期分母 > 0（成本卡还需两侧均非 null）」时
// 才渲染 ▲/▼ 百分比角标；否则一律走兜底占位（绝不渲染 ↑∞ / NaN）。
//
// 颜色语义（来自设计稿改动清单）：
//   - 请求 / Token：中性灰（neutral），涨跌都不带价值判断；
//   - 成本：古铜（warn 涨 / good 跌）——涨是轻警示、跌是好事；
//   - 失败：红绿（bad 涨 / good 跌）——失败上升是坏事。

import type { Overview } from '../types'

// 角标语气 → 对应 .delta 样式类（设计稿同名 tone）。
export type DeltaTone = 'neutral' | 'good' | 'warn' | 'bad'

// 'none' = 无可比基准 / 分母为 0 / 成本未知，走占位；'pct' = 正常环比百分比。
// direction='flat' = 与上期持平（ratio===0），显示 0% 但不带 ▲/▼ 箭头、tone 中性，
// 避免把「无变化」渲染成「上升」语义。
export type DeltaResult =
  | { kind: 'none'; placeholder: string }
  | { kind: 'pct'; direction: 'up' | 'down' | 'flat'; percent: number; tone: DeltaTone }

// 指标语义：决定涨/跌各自的 tone。
//   - neutralMetric：请求 / Token，涨跌都 neutral；
//   - costMetric：成本，涨 warn / 跌 good；
//   - badMetric：失败，涨 bad / 跌 good。
type MetricKind = 'neutral' | 'cost' | 'bad'

function toneFor(kind: MetricKind, direction: 'up' | 'down'): DeltaTone {
  if (kind === 'neutral') return 'neutral'
  if (kind === 'cost') return direction === 'up' ? 'warn' : 'good'
  // bad（失败）：涨红、跌绿。
  return direction === 'up' ? 'bad' : 'good'
}

// 通用环比：current/previous 为数值指标（请求/Token/失败）。
// 上期无数据或分母 <= 0 → 兜底占位。
function computeDelta(
  current: number,
  previous: number | null | undefined,
  hasPrevious: boolean,
  kind: MetricKind,
  placeholder: string,
): DeltaResult {
  if (!hasPrevious || previous == null || previous <= 0) {
    return { kind: 'none', placeholder }
  }
  const ratio = ((current - previous) / previous) * 100
  // 取绝对值四舍五入到整数百分比（与设计稿 "▲ 12%" 一致）。
  const percent = Math.round(Math.abs(ratio))
  // 持平：ratio===0（或四舍五入后为 0%）单独走 flat，tone 中性、不带箭头，
  // 不把「无变化」当成「上升」。toneFor 只接受 up/down，故 flat 不经它。
  if (percent === 0) {
    return { kind: 'pct', direction: 'flat', percent: 0, tone: 'neutral' }
  }
  const direction = ratio > 0 ? 'up' : 'down'
  return { kind: 'pct', direction, percent, tone: toneFor(kind, direction) }
}

// 占位文案：无基准（上一周期完全没数据）显示「新」；分母为 0 / 成本未知显示「—」。
const PH_NEW = '新'
const PH_DASH = '—'

// 4 个 KPI 的环比角标计算入口。previous 缺失字段时各自兜底。
export interface OverviewDeltas {
  requests: DeltaResult
  tokens: DeltaResult
  cost: DeltaResult
  failed: DeltaResult
}

export function overviewDeltas(o: Overview): OverviewDeltas {
  const prev = o.previous
  // 无可比基准（整个上一周期没数据）：四项都显示「新」。
  const noBase = !o.hasPrevious || prev == null
  const basePlaceholder = noBase ? PH_NEW : PH_DASH

  // 成本卡特殊：本期或上期 cost 任一为 null → 成本未知，走「—」占位，不参与百分比。
  const costResult: DeltaResult =
    noBase || o.cost == null || prev?.cost == null
      ? { kind: 'none', placeholder: noBase ? PH_NEW : PH_DASH }
      : computeDelta(o.cost, prev.cost, o.hasPrevious, 'cost', PH_DASH)

  return {
    requests: computeDelta(o.requests, prev?.requests, o.hasPrevious, 'neutral', basePlaceholder),
    tokens: computeDelta(o.tokens, prev?.tokens, o.hasPrevious, 'neutral', basePlaceholder),
    cost: costResult,
    failed: computeDelta(o.failed, prev?.failed, o.hasPrevious, 'bad', basePlaceholder),
  }
}
