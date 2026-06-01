import { Inbox } from 'lucide-react'
import { cn } from '@/lib/utils'

// 仪表台空数据态：图标 + 文案，碳黑克制风格。
export function Empty({ message = '暂无数据', className }: { message?: string; className?: string }) {
  return (
    <div className={cn('flex flex-col items-center justify-center gap-2 text-muted-foreground', className)}>
      <Inbox className="size-6 text-faint" />
      <span className="font-mono text-xs tracking-wide">{message}</span>
    </div>
  )
}
