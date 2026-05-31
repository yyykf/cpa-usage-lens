import { useState } from 'react'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import type { Period, CustomRange } from '../types'

const QUICK: { value: Exclude<Period, 'custom'>; label: string }[] = [
  { value: 'today', label: '今天' },
  { value: '7d', label: '近7天' },
  { value: '30d', label: '近30天' },
]

// 本地日期 YYYY-MM-DD（按浏览器本地时区，与展示口径一致）。
function localDate(offsetDays = 0): string {
  const d = new Date()
  d.setDate(d.getDate() - offsetDays)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export default function PeriodSwitcher({
  period,
  custom,
  onChange,
}: {
  period: Period
  custom: CustomRange | null
  onChange: (period: Period, custom?: CustomRange) => void
}) {
  const [open, setOpen] = useState(false)
  // 草稿态：弹层里编辑，点"应用范围"才上抛。
  const [from, setFrom] = useState<string>(custom?.from ?? localDate(6))
  const [to, setTo] = useState<string>(custom?.to ?? localDate(0))

  const segClass = (active: boolean) =>
    cn(
      'rounded-md px-3.5 py-1.5 font-sans text-[13px] transition-all',
      active
        ? 'bg-primary font-medium text-primary-foreground shadow-[0_2px_14px_hsl(0_0%_100%/0.12),inset_0_1px_0_hsl(0_0%_100%/0.5)]'
        : 'text-muted-foreground hover:text-foreground',
    )

  const applyQuick = (q: Period) => {
    setOpen(false)
    onChange(q)
  }

  const applyCustom = () => {
    if (!from || !to) return
    // 保证 from <= to，顺手纠正用户的反向选择。
    const [lo, hi] = from <= to ? [from, to] : [to, from]
    setOpen(false)
    onChange('custom', { from: lo, to: hi })
  }

  // 快捷范围（弹层内）：直接填充草稿日期。
  const setQuickRange = (days: number, anchor: 'rolling' | 'thisMonth' | 'lastMonth') => {
    if (anchor === 'rolling') {
      setFrom(localDate(days - 1))
      setTo(localDate(0))
      return
    }
    const now = new Date()
    if (anchor === 'thisMonth') {
      const first = new Date(now.getFullYear(), now.getMonth(), 1)
      setFrom(toISO(first))
      setTo(localDate(0))
    } else {
      const first = new Date(now.getFullYear(), now.getMonth() - 1, 1)
      const last = new Date(now.getFullYear(), now.getMonth(), 0)
      setFrom(toISO(first))
      setTo(toISO(last))
    }
  }

  return (
    <div className="inline-flex items-center gap-3">
      <div className="inline-flex rounded-lg border border-border bg-card/90 p-[3px]">
        {QUICK.map((opt) => (
          <button key={opt.value} type="button" onClick={() => applyQuick(opt.value)} className={segClass(period === opt.value)}>
            {opt.label}
          </button>
        ))}
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <button type="button" className={segClass(period === 'custom')}>
              自定义
            </button>
          </PopoverTrigger>
          <PopoverContent align="end" sideOffset={8} className="w-[300px] gap-0 p-4">
            <h4 className="mb-3 font-mono text-[11px] uppercase tracking-[0.18em] text-faint">选择日期范围</h4>
            <div className="mb-3 flex gap-1.5">
              <QuickChip label="近 14 天" onClick={() => setQuickRange(14, 'rolling')} />
              <QuickChip label="本月" onClick={() => setQuickRange(0, 'thisMonth')} />
              <QuickChip label="上月" onClick={() => setQuickRange(0, 'lastMonth')} />
            </div>
            <div className="mb-2.5 flex flex-col gap-1.5">
              <label htmlFor="period-from" className="text-xs text-muted-foreground">
                开始
              </label>
              <input
                id="period-from"
                type="date"
                value={from}
                max={to || undefined}
                onChange={(e) => setFrom(e.target.value)}
                className="rounded-md border border-border bg-[hsl(var(--bg-1))] px-2.5 py-2 font-mono text-[13px] text-foreground outline-hidden focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/20 scheme-dark"
              />
            </div>
            <div className="mb-3 flex flex-col gap-1.5">
              <label htmlFor="period-to" className="text-xs text-muted-foreground">
                结束
              </label>
              <input
                id="period-to"
                type="date"
                value={to}
                min={from || undefined}
                onChange={(e) => setTo(e.target.value)}
                className="rounded-md border border-border bg-[hsl(var(--bg-1))] px-2.5 py-2 font-mono text-[13px] text-foreground outline-hidden focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/20 scheme-dark"
              />
            </div>
            <Button type="button" className="w-full" onClick={applyCustom} disabled={!from || !to}>
              应用范围
            </Button>
          </PopoverContent>
        </Popover>
      </div>
    </div>
  )
}

function toISO(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function QuickChip({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-md border border-border px-2 py-0.5 text-[11px] text-muted-foreground transition-colors hover:border-accent hover:text-foreground"
    >
      {label}
    </button>
  )
}
