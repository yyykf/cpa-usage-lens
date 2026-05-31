import { cn } from '@/lib/utils'

// 仪器编号（如 "01 — 周期总览"）：Fira Code 大写 + 字距 + 前缀短刻度线。去 AI 味关键元素。
export function Kicker({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        'mb-3 flex items-center gap-2.5 font-mono text-[11px] uppercase tracking-[0.2em] text-faint',
        className,
      )}
    >
      <span aria-hidden className="h-px w-3.5 bg-accent/50" />
      {children}
    </div>
  )
}

// 仪表台面板：碳黑渐变底 + 顶部 1px 高光线 + 内嵌阴影。承载各模块内容。
export function Panel({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-lg border border-border',
        'bg-linear-to-b from-card to-[hsl(var(--bg-1))]',
        'shadow-[inset_0_1px_0_hsl(var(--foreground)/0.025),0_18px_40px_-28px_hsl(0_0%_0%/0.9)]',
        // 顶部高光线
        'before:absolute before:inset-x-0 before:top-0 before:h-px',
        'before:bg-linear-to-r before:from-transparent before:via-foreground/[0.07] before:to-transparent',
        className,
      )}
    >
      {children}
    </div>
  )
}

// 模块标题（面板内）：前缀语义色竖条 + 标题 + 可选右侧（图例/排序说明）。
export function PanelHeader({
  title,
  barColor = 'hsl(var(--accent))',
  right,
  className,
}: {
  title: React.ReactNode
  barColor?: string
  right?: React.ReactNode
  className?: string
}) {
  return (
    <div className={cn('flex items-center justify-between px-5 pt-4 pb-1.5', className)}>
      <h3 className="flex items-center gap-2.5 text-sm font-semibold text-foreground">
        <span aria-hidden className="h-3.5 w-[3px] rounded-sm" style={{ background: barColor }} />
        {title}
      </h3>
      {right}
    </div>
  )
}

export interface TokenSegment {
  label: string
  value: number
  color: string
}

// token 构成水平占比条：段与段之间留深色间隙（靠间隙分辨相邻哑光色）。
// variant="bar" 大条（带图例由外部渲染）；variant="mini" 账号行内迷你条。
export function TokenBar({
  segments,
  variant = 'bar',
  className,
}: {
  segments: TokenSegment[]
  variant?: 'bar' | 'mini'
  className?: string
}) {
  const total = segments.reduce((s, seg) => s + seg.value, 0)
  const visible = segments.filter((seg) => seg.value > 0)
  const isMini = variant === 'mini'

  return (
    <div
      className={cn(
        'flex overflow-hidden rounded-md bg-background',
        isMini ? 'h-[5px] w-[170px] gap-px' : 'h-3 gap-0.5',
        className,
      )}
      role="img"
      aria-label="Token 构成占比"
    >
      {total > 0 ? (
        visible.map((seg) => (
          <span
            key={seg.label}
            className="h-full"
            style={{ width: `${(seg.value / total) * 100}%`, background: seg.color }}
          />
        ))
      ) : (
        <span className="h-full w-full bg-border-soft" />
      )}
    </div>
  )
}
