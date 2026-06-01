import { Panel, TokenBar } from './Primitives'
import { tokenSegments } from '@/lib/tokens'
import { formatInt } from '@/lib/format'
import type { Overview } from '../../types'

// Token 构成条：输入 / 缓存读 / 缓存写 / 输出 四段水平占比 + 图例带数值。
// 四段经 tokenSegments 跨 provider 归一化（OpenAI 的 cachedTokens 计入缓存读）；
// 分母用四段之和（= totalTokens），与 design-system / mockup 一致。
export default function TokenComposition({ overview, loading }: { overview: Overview; loading: boolean }) {
  const segments = tokenSegments(overview)
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
