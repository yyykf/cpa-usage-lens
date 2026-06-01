// 共享格式化工具（数字 / 字节 / 成本 / 相对时间），全站单一来源（DRY）。

// 千分位整数。
export function formatInt(n: number): string {
  return n.toLocaleString()
}

// 紧凑大数：1.2k / 3.4M / 1.1B（图表轴、KPI 副信息用）。
export function formatCompact(n: number): string {
  const abs = Math.abs(n)
  if (abs >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(1)}B`
  if (abs >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (abs >= 1_000) return `${(n / 1_000).toFixed(1)}k`
  return `${n}`
}

// 成本：null = 缺价显示"未知"；否则 $x.xxxx。
export function formatCost(cost: number | null): string {
  return cost === null ? '未知' : `$${cost.toFixed(4)}`
}

// 成本轴/紧凑成本：$3.4 / $1.2k。
export function formatCostCompact(cost: number): string {
  if (cost >= 1000) return `$${(cost / 1000).toFixed(1)}k`
  return `$${cost.toFixed(2)}`
}

// 字节 → B / KB / MB（整数不带小数点）。
export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  const kb = n / 1024
  if (kb < 1024) return `${kb % 1 === 0 ? kb : kb.toFixed(1)} KB`
  const mb = kb / 1024
  return `${mb % 1 === 0 ? mb : mb.toFixed(1)} MB`
}

// 距上次的秒数 → 人类可读："3s 前" / "5m 前" / "1h 20m 前"。
export function formatLag(sec: number): string {
  if (sec < 60) return `${sec}s 前`
  if (sec < 3600) return `${Math.floor(sec / 60)}m 前`
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return m > 0 ? `${h}h ${m}m 前` : `${h}h 前`
}

// "2026-05-31T15:21:08+08:00" → "15:21:08"（游标时间，取时分秒）。
export function formatClock(iso: string | null): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleTimeString('zh-CN', { hour12: false })
}

// "YYYY-MM-DD" → "M/D"（图表 X 轴短标签）。
export function formatDateShort(date: string): string {
  const parts = date.split('-')
  if (parts.length !== 3) return date
  return `${Number(parts[1])}/${Number(parts[2])}`
}

// 百分比（0-100），保留一位小数，去掉无意义的 .0。
export function formatPercent(value: number, total: number): string {
  if (total <= 0) return '0%'
  const p = (value / total) * 100
  return `${p % 1 === 0 ? p : p.toFixed(1)}%`
}
