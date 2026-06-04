import { useEffect, useRef, useState, useCallback } from 'react'

// 自动刷新档位（秒）。0 = 关闭轮询。默认 30s。
export const REFRESH_OPTIONS = [0, 5, 10, 30, 60] as const
export type RefreshInterval = (typeof REFRESH_OPTIONS)[number]

export const DEFAULT_REFRESH: RefreshInterval = 30

const STORAGE_KEY = 'cpa.autoRefresh'

// 档位 → 顶栏展示文案（关闭态特殊处理）。
export function refreshLabel(value: RefreshInterval): string {
  return value === 0 ? '关闭' : `${value}s`
}

// 从 localStorage 读回上次偏好；无偏好或非法值回落默认 30s。
function readStored(): RefreshInterval {
  if (typeof window === 'undefined') return DEFAULT_REFRESH
  const raw = window.localStorage.getItem(STORAGE_KEY)
  // 无偏好（首次，raw=null）→ 默认 30s。必须先排除：Number(null)=0 恰是合法「关闭」档，
  // 否则首次会被误判成用户主动选了「关闭」，永远落不到默认值。
  if (raw === null) return DEFAULT_REFRESH
  const parsed = Number(raw)
  return (REFRESH_OPTIONS as readonly number[]).includes(parsed) ? (parsed as RefreshInterval) : DEFAULT_REFRESH
}

/**
 * 自动刷新轮询。
 * - 记忆用户偏好到 localStorage；
 * - 选中某档后按该间隔调用 onTick；选「关闭」(0) 停止；
 * - 档位切换 / 组件卸载时清理定时器，避免泄漏或重复定时器。
 *
 * onTick 用 ref 保存最新引用，避免它每次渲染变化都重建定时器（否则会和 loadData 的
 * 依赖一起把节奏打乱）。
 */
export function useAutoRefresh(onTick: () => void): {
  interval: RefreshInterval
  setInterval: (value: RefreshInterval) => void
} {
  const [interval, setIntervalState] = useState<RefreshInterval>(readStored)

  const onTickRef = useRef(onTick)
  useEffect(() => {
    onTickRef.current = onTick
  }, [onTick])

  const setInterval = useCallback((value: RefreshInterval): void => {
    setIntervalState(value)
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(STORAGE_KEY, String(value))
    }
  }, [])

  useEffect(() => {
    if (interval === 0) return
    const id = window.setInterval(() => {
      onTickRef.current()
    }, interval * 1000)
    return () => window.clearInterval(id)
  }, [interval])

  return { interval, setInterval }
}
