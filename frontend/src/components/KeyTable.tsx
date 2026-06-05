import { useMemo, useState } from 'react'
import { ArrowDown, ArrowUp, TriangleAlert, KeyRound } from 'lucide-react'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Panel, PanelHeader, TokenBar } from './dashboard/Primitives'
import { Empty } from './dashboard/Empty'
import { TableSkeleton } from './dashboard/TableSkeleton'
import { CHART_COLORS } from '@/lib/charts'
import { tokenSegments } from '@/lib/tokens'
import { formatInt, formatCost } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { KeyUsage } from '../types'

type SortKey = 'requests' | 'tokens' | 'cost' | 'failed'

// 「API key 用量榜」：与账号榜正交、平级的独立维度（不是账号下钻）。
// 排序 / 列结构 / 成本缺价沉底 / 失败徽标 全部沿用账号榜口径（DRY），
// 仅把维度键从 source 换成脱敏 key（keyMask 展示、fingerprint 做唯一标识）。
export default function KeyTable({ keys, loading }: { keys: KeyUsage[]; loading: boolean }) {
  const [sortKey, setSortKey] = useState<SortKey>('cost')
  const [desc, setDesc] = useState(true)

  const sorted = useMemo(() => {
    const copy = [...keys]
    copy.sort((a, b) => {
      // cost 为 null（缺价）视为最小，始终沉底。
      const av = sortKey === 'cost' ? (a.cost ?? -1) : a[sortKey]
      const bv = sortKey === 'cost' ? (b.cost ?? -1) : b[sortKey]
      return desc ? bv - av : av - bv
    })
    return copy
  }, [keys, sortKey, desc])

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
        title="各 API key 用量"
        barColor={CHART_COLORS.tokens}
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
                  API key / Token 构成
                </TableHead>
                <SortHead label="请求" active={sortKey === 'requests'} desc={desc} onClick={() => clickSort('requests')} />
                <SortHead label="Token" active={sortKey === 'tokens'} desc={desc} onClick={() => clickSort('tokens')} />
                <SortHead label="成本" active={sortKey === 'cost'} desc={desc} onClick={() => clickSort('cost')} />
                <SortHead label="失败" active={sortKey === 'failed'} desc={desc} onClick={() => clickSort('failed')} />
              </TableRow>
            </TableHeader>
            <TableBody>
              {sorted.map((k) => (
                // fingerprint 全长唯一（'none' 桶也只有一行），稳定且不会撞 key。
                <TableRow key={k.fingerprint} className="border-border-soft">
                  <TableCell className="whitespace-normal py-3.5">
                    <div className="flex flex-col gap-1.5">
                      <KeyLabel fingerprint={k.fingerprint} keyMask={k.keyMask} />
                      <TokenBar segments={tokenSegments(k)} variant="mini" />
                    </div>
                  </TableCell>
                  <TableCell className="py-3.5 text-right font-num text-[13.5px] text-foreground">{formatInt(k.requests)}</TableCell>
                  <TableCell className="py-3.5 text-right font-num text-[13.5px] text-foreground">{formatInt(k.tokens)}</TableCell>
                  <TableCell className="py-3.5 text-right font-num text-[13.5px]">
                    {k.cost === null ? (
                      <span className="text-muted-foreground">未知</span>
                    ) : (
                      <span className="text-[hsl(var(--data-cost))]">{formatCost(k.cost)}</span>
                    )}
                  </TableCell>
                  <TableCell className="py-3.5 text-right font-num text-[13.5px]">
                    {k.failed > 0 ? (
                      <span className="inline-flex items-center justify-end gap-1.5 text-destructive">
                        {formatInt(k.failed)}
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

// key 维度标签：key 是字符串标识，用等宽字体呈现掩码更易读。
// 'none' 桶（非 key / oauth 认证）展示成中文「非 key 认证」，避免界面出现裸 "none"/"(no key)"。
function KeyLabel({ fingerprint, keyMask }: { fingerprint: string; keyMask: string }) {
  const isBucket = fingerprint === 'none'
  const label = isBucket ? '非 key 认证' : keyMask
  return (
    <span className="inline-flex items-center gap-1.5">
      <KeyRound aria-hidden className={cn('size-3', isBucket ? 'text-faint' : 'text-accent/70')} />
      <span className={cn('text-[13px]', isBucket ? 'font-sans text-muted-foreground' : 'font-mono text-foreground')}>
        {label}
      </span>
    </span>
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
