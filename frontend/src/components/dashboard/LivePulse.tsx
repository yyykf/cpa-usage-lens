import { cn } from '@/lib/utils'

// 呼吸状态点：绿=采集中/健康，红=异常，灰=空闲。配合文字使用（不只靠颜色）。
export function PulseDot({
  tone = 'success',
  className,
}: {
  tone?: 'success' | 'error' | 'idle'
  className?: string
}) {
  const color =
    tone === 'success'
      ? 'hsl(var(--data-success))'
      : tone === 'error'
        ? 'hsl(var(--destructive))'
        : 'hsl(var(--muted-foreground))'
  return (
    <span
      aria-hidden
      className={cn('inline-block size-[7px] rounded-full', tone === 'success' && 'animate-pulse-ring', className)}
      style={{ background: color }}
    />
  )
}

// 顶栏自动刷新状态徽标。
// 只表达"前端是否在自动刷新"（开启=绿点脉冲+档位，关闭=灰点静止）；
// 真实采集健康交给底部 CollectorHealth 卡，避免顶栏再次"看似实时、实则不刷新"的误导。
export function LiveBadge({ active, intervalLabel }: { active: boolean; intervalLabel: string }) {
  return (
    <span
      className={cn(
        'ml-1.5 inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 font-mono text-[11px] tracking-wider',
        active
          ? 'border-data-success/25 bg-data-success/8 text-data-success'
          : 'border-border bg-muted/40 text-muted-foreground',
      )}
    >
      <PulseDot tone={active ? 'success' : 'idle'} />
      {active ? `自动刷新 · ${intervalLabel}` : '自动刷新 · 关闭'}
    </span>
  )
}
