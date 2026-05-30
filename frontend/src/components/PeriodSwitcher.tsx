import type { Period } from '../types'

const OPTIONS: { value: Period; label: string }[] = [
  { value: 'today', label: '今天' },
  { value: '7d', label: '近7天' },
  { value: '30d', label: '近30天' },
  { value: 'custom', label: '自定义' },
]

export default function PeriodSwitcher({
  period,
  onChange,
}: {
  period: Period
  onChange: (p: Period) => void
}) {
  return (
    <div className="inline-flex gap-1 rounded-lg bg-muted p-1">
      {OPTIONS.map((option) => {
        const isActive = period === option.value
        return (
          <button
            key={option.value}
            type="button"
            onClick={() => onChange(option.value)}
            className={`px-3 py-1.5 rounded-md text-sm transition-colors ${
              isActive
                ? 'bg-primary text-white'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}
