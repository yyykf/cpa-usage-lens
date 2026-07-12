// Token 构成归一化（跨 provider 通用，全站单一来源 DRY）。
//
// 缓存语义只有「读」「写」两类，但 OpenAI input 包含缓存、Claude input 不包含缓存，
// 且 CPA v7.2.67 还可能把同一份 read 同时写入 cached/cacheRead。后端基于
// LiteLLM provider 元数据统一给出 uncachedInputTokens/canonicalCacheReadTokens，
// 前端不再猜 provider，也不会把 alias 相加两次。
//
// 归一化四段，各段相加正好 = totalTokens：
//   输入   = uncachedInputTokens
//   缓存读 = canonicalCacheReadTokens
//   缓存写 = cacheCreationTokens
//   输出   = outputTokens
//
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
    { label: '输入', value: b.uncachedInputTokens, color: TOKEN_PART_COLORS.input },
    { label: '缓存读', value: b.canonicalCacheReadTokens, color: TOKEN_PART_COLORS.cacheRead },
    { label: '缓存写', value: b.cacheCreationTokens, color: TOKEN_PART_COLORS.cacheCreation },
    { label: '输出', value: b.outputTokens, color: TOKEN_PART_COLORS.output },
  ]
}
