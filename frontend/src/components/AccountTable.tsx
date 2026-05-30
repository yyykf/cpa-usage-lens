import type { AccountUsage } from '../types'

export default function AccountTable({ accounts }: { accounts: AccountUsage[] }) {
  // 先按 cost 降序排序，null 视为 -1 排最后；复制一份避免修改入参
  const sorted: AccountUsage[] = [...accounts].sort(
    (a, b) => (b.cost ?? -1) - (a.cost ?? -1),
  )

  return (
    <div className="rounded-2xl border border-border bg-card p-5 md:p-6 hover:border-primary/40 transition-colors">
      <h2 className="text-foreground text-base font-medium mb-4">各账号用量</h2>

      {sorted.length === 0 ? (
        <div className="text-muted-foreground text-center py-8">暂无数据</div>
      ) : (
        <table className="w-full">
          <thead>
            <tr className="text-muted-foreground text-xs uppercase tracking-wide border-b border-border">
              <th className="text-left font-medium py-2">账号</th>
              <th className="text-right font-medium py-2">请求数</th>
              <th className="text-right font-medium py-2">Token</th>
              <th className="text-right font-medium py-2">成本</th>
              <th className="text-right font-medium py-2">失败数</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((account: AccountUsage) => (
              <tr
                key={account.source}
                className="hover:bg-muted/50 border-b border-border/50 transition-colors"
              >
                <td className="text-left text-foreground py-2">{account.source}</td>
                <td className="text-right font-num text-foreground py-2">
                  {account.requests.toLocaleString()}
                </td>
                <td className="text-right font-num text-foreground py-2">
                  {account.tokens.toLocaleString()}
                </td>
                <td className="text-right font-num py-2">
                  {account.cost === null ? (
                    <span className="text-muted-foreground">未知</span>
                  ) : (
                    <span className="text-foreground">{'$' + account.cost.toFixed(4)}</span>
                  )}
                </td>
                <td
                  className={
                    'text-right font-num py-2 ' +
                    (account.failed > 0 ? 'text-destructive' : 'text-foreground')
                  }
                >
                  {account.failed.toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
