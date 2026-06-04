import { cn } from '@/lib/utils'
import type { DeltaResult, DeltaTone } from '@/lib/delta'

// 环比角标：对齐设计稿 .delta（圆角 + 等宽 + ▲/▼ 百分比 + 4 档 tone 配色）。
// kind='none' 时显示克制的占位（「新」/「—」），不带 tone 背景，避免渲染假百分比。

const TONE_CLASS: Record<DeltaTone, string> = {
  neutral: 'text-muted-foreground bg-muted-foreground/10',
  good: 'text-[hsl(var(--data-success))] bg-[hsl(var(--data-success)/0.12)]',
  warn: 'text-[hsl(var(--data-cost))] bg-[hsl(var(--data-cost)/0.12)]',
  bad: 'text-[hsl(var(--data-failed))] bg-[hsl(var(--data-failed)/0.12)]',
}

export function DeltaBadge({ delta }: { delta: DeltaResult }) {
  if (delta.kind === 'none') {
    return (
      <span
        className="rounded-md px-1.5 py-0.5 font-mono text-xs font-semibold text-faint"
        title="无可比基准或成本未知"
      >
        {delta.placeholder}
      </span>
    )
  }

  // flat（持平）：显示 0% 不带 ▲/▼，避免「无变化」被渲染成涨/跌。
  const arrow = delta.direction === 'flat' ? '' : delta.direction === 'up' ? '▲' : '▼'
  return (
    <span className={cn('rounded-md px-1.5 py-0.5 font-mono text-xs font-semibold', TONE_CLASS[delta.tone])}>
      {arrow ? `${arrow} ` : ''}
      {delta.percent}%
    </span>
  )
}
