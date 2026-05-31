import { Panel, TokenBar, type TokenSegment } from './Primitives'
import { TOKEN_PART_COLORS } from '@/lib/charts'
import { formatInt } from '@/lib/format'
import type { Overview } from '../../types'

// Token 构成条：输入 / 输出 / 缓存读 / 缓存写 四段水平占比 + 图例带数值。
// 分母用四段之和（计费视角的明细拆分），与 design-system / mockup 一致。
export default function TokenComposition({ overview, loading }: { overview: Overview; loading: boolean }) {
  const segments: TokenSegment[] = [
    { label: '输入', value: overview.inputTokens, color: TOKEN_PART_COLORS.input },
    { label: '输出', value: overview.outputTokens, color: TOKEN_PART_COLORS.output },
    { label: '缓存读', value: overview.cacheReadTokens, color: TOKEN_PART_COLORS.cacheRead },
    { label: '缓存写', value: overview.cacheCreationTokens, color: TOKEN_PART_COLORS.cacheCreation },
  ]
  const total = segments.reduce((s, seg) => s + seg.value, 0)

  return (
    <Panel className="px-5 py-4">
      <div className="mb-3 flex items-center justify-between">
        <div className="text-[13px] font-medium text-foreground">Token 构成</div>
        <div className="font-num text-xs text-muted-foreground">{formatInt(total)} tokens</div>
      </div>

      {loading ? (
        <div className="h-3 w-full animate-pulse rounded-md bg-muted" />
      ) : (
        <TokenBar segments={segments} variant="bar" />
      )}

      <div className="mt-3 flex flex-wrap gap-x-[18px] gap-y-2">
        {segments.map((seg) => (
          <div key={seg.label} className="flex items-center gap-2 text-xs text-muted-foreground">
            <span aria-hidden className="size-[9px] rounded-sm" style={{ background: seg.color }} />
            {seg.label} <span className="font-num font-medium text-foreground">{loading ? '—' : formatInt(seg.value)}</span>
          </div>
        ))}
      </div>
    </Panel>
  )
}
