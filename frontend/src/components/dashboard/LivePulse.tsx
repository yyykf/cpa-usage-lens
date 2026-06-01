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

// 顶栏"采集中"呼吸徽标。
export function LiveBadge({ label = '采集中' }: { label?: string }) {
  return (
    <span className="ml-1.5 inline-flex items-center gap-1.5 rounded-full border border-data-success/25 bg-data-success/8 px-2.5 py-1 font-mono text-[11px] tracking-wider text-data-success">
      <PulseDot tone="success" />
      {label}
    </span>
  )
}
