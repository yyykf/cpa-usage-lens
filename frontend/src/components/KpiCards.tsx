import { Activity, Coins, DollarSign, AlertTriangle } from 'lucide-react'
import type { Overview } from '../types'

const CARD_CLASS =
  'rounded-2xl border border-border bg-card p-5 md:p-6 hover:border-primary/40 transition-colors'

export default function KpiCards({ overview }: { overview: Overview }) {
  const costLabel: string =
    overview.cost === null ? '未知' : `$${overview.cost.toFixed(4)}`

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <div className={CARD_CLASS}>
        <div className="flex items-start justify-between">
          <span className="text-muted-foreground text-sm">总请求</span>
          <Activity className="w-5 h-5 text-data-requests" />
        </div>
        <div className="mt-3 text-3xl font-num font-semibold text-foreground">
          {overview.requests.toLocaleString()}
        </div>
      </div>

      <div className={CARD_CLASS}>
        <div className="flex items-start justify-between">
          <span className="text-muted-foreground text-sm">总 Token</span>
          <Coins className="w-5 h-5 text-data-tokens" />
        </div>
        <div className="mt-3 text-3xl font-num font-semibold text-foreground">
          {overview.tokens.toLocaleString()}
        </div>
      </div>

      <div className={CARD_CLASS}>
        <div className="flex items-start justify-between">
          <span className="text-muted-foreground text-sm">总成本</span>
          <DollarSign className="w-5 h-5 text-data-cost" />
        </div>
        <div
          className={`mt-3 text-3xl font-num font-semibold ${
            overview.cost === null ? 'text-muted-foreground' : 'text-foreground'
          }`}
        >
          {costLabel}
        </div>
      </div>

      <div className={CARD_CLASS}>
        <div className="flex items-start justify-between">
          <span className="text-muted-foreground text-sm">失败数</span>
          <AlertTriangle className="w-5 h-5 text-data-failed" />
        </div>
        <div
          className={`mt-3 text-3xl font-num font-semibold ${
            overview.failed > 0 ? 'text-destructive' : 'text-foreground'
          }`}
        >
          {overview.failed.toLocaleString()}
        </div>
      </div>
    </div>
  )
}
