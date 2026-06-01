// KPI 迷你折线（纯 SVG，无需 recharts）。把一组数值画成一条趋势线。
// 数据点 <2 时退化为一条水平基线，仍保持视觉占位、不难看。
export function Sparkline({
  values,
  color = 'hsl(var(--data-tokens))',
  width = 150,
  height = 22,
}: {
  values: number[]
  color?: string
  width?: number
  height?: number
}) {
  const pad = 3
  const w = width
  const h = height
  const pts = values.length >= 2 ? values : [values[0] ?? 0, values[0] ?? 0]
  const max = Math.max(...pts)
  const min = Math.min(...pts)
  const span = max - min || 1
  const stepX = (w - pad * 2) / (pts.length - 1)

  const coords = pts.map((v, i) => {
    const x = pad + i * stepX
    const y = pad + (1 - (v - min) / span) * (h - pad * 2)
    return `${x.toFixed(1)},${y.toFixed(1)}`
  })

  return (
    <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`} aria-hidden className="overflow-visible">
      <polyline points={coords.join(' ')} fill="none" stroke={color} strokeWidth={1.6} strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
