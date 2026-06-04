import { useState } from 'react'
import { Check, ChevronDown, RefreshCw } from 'lucide-react'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import { REFRESH_OPTIONS, refreshLabel, type RefreshInterval } from '@/hooks/useAutoRefresh'

// 顶栏自动刷新档位选择器：关闭 / 5s / 10s / 30s / 60s。
// 样式对齐设计稿 .refresh（青色边框 + 图标 + 等宽值），克制辉光。
export default function RefreshSelector({
  value,
  onChange,
}: {
  value: RefreshInterval
  onChange: (value: RefreshInterval) => void
}) {
  const [open, setOpen] = useState(false)
  const active = value !== 0

  const pick = (next: RefreshInterval) => {
    setOpen(false)
    onChange(next)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="自动刷新间隔"
          className={cn(
            'inline-flex items-center gap-2 rounded-lg border px-3 py-[7px] text-[12.5px] transition-colors',
            active
              ? 'border-accent/40 bg-accent/[0.07] text-foreground'
              : 'border-border bg-card/90 text-muted-foreground',
          )}
        >
          <RefreshCw className={cn('size-[13px]', active ? 'text-accent' : 'text-faint')} />
          <span>自动刷新</span>
          <span className="text-faint">·</span>
          <span className={cn('font-mono font-semibold', active ? 'text-accent' : 'text-muted-foreground')}>
            {refreshLabel(value)}
          </span>
          <ChevronDown className="size-3 text-faint" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" sideOffset={8} className="w-[150px] gap-0 p-1.5">
        {REFRESH_OPTIONS.map((opt) => {
          const on = opt === value
          return (
            <button
              key={opt}
              type="button"
              onClick={() => pick(opt)}
              className={cn(
                'flex w-full items-center justify-between rounded-md px-3 py-2 font-mono text-[12.5px] transition-colors',
                on ? 'bg-accent/[0.14] text-accent' : 'text-muted-foreground hover:bg-muted hover:text-foreground',
              )}
            >
              <span>{refreshLabel(opt)}</span>
              {on && <Check className="size-3.5" />}
            </button>
          )
        })}
      </PopoverContent>
    </Popover>
  )
}
