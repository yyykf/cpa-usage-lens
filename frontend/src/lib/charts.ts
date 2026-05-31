// 图表用颜色（recharts 的 stroke/fill 需颜色字符串，统一走 CSS 变量，不硬编码 hex）。
// 与 design-system §2 数据语义色 / 模型色阶一一对应。

export const CHART_COLORS = {
  requests: 'hsl(var(--data-requests))', // 银灰
  tokens: 'hsl(var(--data-tokens))', // 哑光青
  cost: 'hsl(var(--data-cost))', // 古铜
  failed: 'hsl(var(--data-failed))', // 哑光红
  success: 'hsl(var(--data-success))', // 哑光绿
  grid: 'hsl(var(--border-soft))', // 网格线
  axis: 'hsl(var(--faint))', // 轴刻度文字
  background: 'hsl(var(--background))', // 段间深色间隙 / 描边
} as const

// token 构成四段（输入 / 输出 / 缓存读 / 缓存写）固定语义色。
// 缓存写用 faint（最弱）——它占比小且属"额外开销"，弱化呈现。
export const TOKEN_PART_COLORS = {
  input: 'hsl(var(--m1))',
  output: 'hsl(var(--m2))',
  cacheRead: 'hsl(var(--m3))',
  cacheCreation: 'hsl(var(--faint))',
} as const

// 模型分布色阶（青→黄绿，4 档；>4 个模型时按序循环复用最后几档的插值）。
export const MODEL_COLOR_SCALE = [
  'hsl(var(--m1))',
  'hsl(var(--m2))',
  'hsl(var(--m3))',
  'hsl(var(--m4))',
] as const

// 第 5+ 个模型在 4 档之外，用 faint 系灰阶兜底（保证仍可区分、不喧宾夺主）。
const OVERFLOW_COLORS = [
  'hsl(var(--faint))',
  'hsl(var(--muted-foreground))',
]

// 按模型在排序数组中的索引取色（稳定映射：同一模型在图例/柱/迷你条颜色一致）。
export function modelColor(index: number): string {
  if (index < MODEL_COLOR_SCALE.length) return MODEL_COLOR_SCALE[index]
  return OVERFLOW_COLORS[(index - MODEL_COLOR_SCALE.length) % OVERFLOW_COLORS.length]
}
