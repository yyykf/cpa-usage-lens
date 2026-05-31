import { Skeleton } from '@/components/ui/skeleton'

// 表格加载占位：预留行高避免内容跳动。
export function TableSkeleton({ rows = 4, cols = 5 }: { rows?: number; cols?: number }) {
  return (
    <div className="flex flex-col gap-3 p-3">
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex items-center gap-4">
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-56" />
            <Skeleton className="h-[5px] w-[170px]" />
          </div>
          {Array.from({ length: cols - 1 }).map((_, c) => (
            <Skeleton key={c} className="h-3.5 w-14" />
          ))}
        </div>
      ))}
    </div>
  )
}
