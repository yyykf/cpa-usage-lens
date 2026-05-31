import { useMemo, useState } from 'react'
import { ArrowDown, ArrowUp, TriangleAlert } from 'lucide-react'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Panel, PanelHeader, TokenBar, type TokenSegment } from './dashboard/Primitives'
import { Empty } from './dashboard/Empty'
import { TableSkeleton } from './dashboard/TableSkeleton'
import { CHART_COLORS, TOKEN_PART_COLORS } from '@/lib/charts'
import { formatInt, formatCost } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { AccountUsage } from '../types'

type SortKey = 'requests' | 'tokens' | 'cost' | 'failed'

function accountSegments(a: AccountUsage): TokenSegment[] {
  return [
    { label: '输入', value: a.inputTokens, color: TOKEN_PART_COLORS.input },
    { label: '输出', value: a.outputTokens, color: TOKEN_PART_COLORS.output },
    { label: '缓存读', value: a.cacheReadTokens, color: TOKEN_PART_COLORS.cacheRead },
    { label: '缓存写', value: a.cacheCreationTokens, color: TOKEN_PART_COLORS.cacheCreation },
  ]
}

export default function AccountTable({ accounts, loading }: { accounts: AccountUsage[]; loading: boolean }) {
  const [sortKey, setSortKey] = useState<SortKey>('cost')
  const [desc, setDesc] = useState(true)

  const sorted = useMemo(() => {
    const copy = [...accounts]
    copy.sort((a, b) => {
      // cost 为 null（缺价）视为最小，始终沉底。
      const av = sortKey === 'cost' ? (a.cost ?? -1) : a[sortKey]
      const bv = sortKey === 'cost' ? (b.cost ?? -1) : b[sortKey]
      return desc ? bv - av : av - bv
    })
    return copy
  }, [accounts, sortKey, desc])

  const clickSort = (key: SortKey) => {
    if (key === sortKey) {
      setDesc((d) => !d)
    } else {
      setSortKey(key)
      setDesc(true)
    }
  }

  const sortLabel = `按${({ requests: '请求', tokens: 'Token', cost: '成本', failed: '失败' } as const)[sortKey]}${desc ? '降序 ↓' : '升序 ↑'}`

  return (
    <Panel>
      <PanelHeader
        title="各账号用量"
        barColor={CHART_COLORS.cost}
        right={<span className="font-mono text-[11px] tracking-wide text-faint">{sortLabel}</span>}
      />
      <div className="px-1.5 pb-2 pt-1">
        {loading ? (
          <TableSkeleton rows={4} cols={5} />
        ) : sorted.length === 0 ? (
          <Empty className="py-10" />
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="font-mono text-[10.5px] uppercase tracking-[0.12em] text-faint">
                  账号 / Token 构成
                </TableHead>
                <SortHead label="请求" active={sortKey === 'requests'} desc={desc} onClick={() => clickSort('requests')} />
                <SortHead label="Token" active={sortKey === 'tokens'} desc={desc} onClick={() => clickSort('tokens')} />
                <SortHead label="成本" active={sortKey === 'cost'} desc={desc} onClick={() => clickSort('cost')} />
                <SortHead label="失败" active={sortKey === 'failed'} desc={desc} onClick={() => clickSort('failed')} />
              </TableRow>
            </TableHeader>
            <TableBody>
              {sorted.map((a) => (
                <TableRow key={a.source} className="border-border-soft">
                  <TableCell className="whitespace-normal py-3.5">
                    <div className="flex flex-col gap-1.5">
                      <span className="font-sans text-[13.5px] text-foreground">{a.source}</span>
                      <TokenBar segments={accountSegments(a)} variant="mini" />
                    </div>
                  </TableCell>
                  <TableCell className="py-3.5 text-right font-num text-[13.5px] text-foreground">{formatInt(a.requests)}</TableCell>
                  <TableCell className="py-3.5 text-right font-num text-[13.5px] text-foreground">{formatInt(a.tokens)}</TableCell>
                  <TableCell className="py-3.5 text-right font-num text-[13.5px]">
                    {a.cost === null ? (
                      <span className="text-muted-foreground">未知</span>
                    ) : (
                      <span className="text-[hsl(var(--data-cost))]">{formatCost(a.cost)}</span>
                    )}
                  </TableCell>
                  <TableCell className="py-3.5 text-right font-num text-[13.5px]">
                    {a.failed > 0 ? (
                      <span className="inline-flex items-center justify-end gap-1.5 text-destructive">
                        {formatInt(a.failed)}
                        <Badge variant="destructive" className="gap-1 px-1.5 font-mono text-[10px]">
                          <TriangleAlert className="size-2.5" />
                          err
                        </Badge>
                      </span>
                    ) : (
                      <span className="text-faint">0</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </Panel>
  )
}

function SortHead({
  label,
  active,
  desc,
  onClick,
}: {
  label: string
  active: boolean
  desc: boolean
  onClick: () => void
}) {
  return (
    <TableHead className="text-right">
      <button
        type="button"
        onClick={onClick}
        className={cn(
          'ml-auto inline-flex items-center gap-1 font-mono text-[10.5px] uppercase tracking-[0.12em] transition-colors',
          active ? 'text-foreground' : 'text-faint hover:text-muted-foreground',
        )}
      >
        {label}
        {active && (desc ? <ArrowDown className="size-3" /> : <ArrowUp className="size-3" />)}
      </button>
    </TableHead>
  )
}
