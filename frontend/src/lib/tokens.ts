// Token 构成归一化（跨 provider 通用，全站单一来源 DRY）。
//
// 缓存语义只有「读」「写」两类，但不同 provider 用不同字段表达：
//   - OpenAI 风格：缓存命中走 cachedTokens（本质=缓存读），无缓存写；
//     且 inputTokens 已【包含】cachedTokens，需减掉才是"未命中缓存的常规输入"。
//   - Anthropic 风格：缓存读=cacheReadTokens、缓存写=cacheCreationTokens，
//     inputTokens 【不含】缓存（其 cachedTokens=0，input - cached = input 不受影响）。
//
// 归一化四段，各段相加正好 = totalTokens：
//   输入   = max(0, inputTokens - cachedTokens)
//   缓存读 = cachedTokens + cacheReadTokens
//   缓存写 = cacheCreationTokens
//   输出   = outputTokens
//
// 验证（两组真实数据相加均 = total）：
//   OpenAI    input=1059422 output=16552 cached=457472 cacheRead=0  cacheCreation=0
//             → 601950 + 457472 + 0 + 16552 = 1075974 ✓
//   Anthropic input=100     output=20    cached=0      cacheRead=50 cacheCreation=30
//             → 100 + 50 + 30 + 20 = 200 ✓

import { TOKEN_PART_COLORS } from '@/lib/charts'
import type { TokenBreakdown } from '../types'

export interface TokenSegmentValue {
  label: string
  value: number
  color: string
}

// 把 6 个原始 token 字段归一化成「输入 / 缓存读 / 缓存写 / 输出」四段。
// 顺序固定（与图例/迷你条一致）；各段之和 = totalTokens。
export function tokenSegments(b: TokenBreakdown): TokenSegmentValue[] {
  return [
    { label: '输入', value: Math.max(0, b.inputTokens - b.cachedTokens), color: TOKEN_PART_COLORS.input },
    { label: '缓存读', value: b.cachedTokens + b.cacheReadTokens, color: TOKEN_PART_COLORS.cacheRead },
    { label: '缓存写', value: b.cacheCreationTokens, color: TOKEN_PART_COLORS.cacheCreation },
    { label: '输出', value: b.outputTokens, color: TOKEN_PART_COLORS.output },
  ]
}
