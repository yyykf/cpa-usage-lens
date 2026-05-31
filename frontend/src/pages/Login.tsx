import { useState } from 'react'
import { AlertCircle, Loader2, Lock } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { login } from '../lib/api'

export default function Login({ onSuccess }: { onSuccess: () => void }) {
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      await login(password)
      onSuccess()
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm gap-0 py-0 ring-1 ring-foreground/[0.08]">
        <CardHeader className="flex flex-col items-center gap-2 px-7 pt-8 pb-2 text-center">
          <div className="mb-1 grid size-11 place-items-center rounded-xl border border-[hsl(240_8%_18%)] bg-gradient-to-br from-[hsl(220_8%_10%)] to-[hsl(240_9%_4%)] shadow-[inset_0_1px_0_hsl(0_0%_100%/0.06),0_0_24px_hsl(186_31%_50%/0.16)]">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden>
              <path d="M3 17l5-6 4 4 4-7 5 8" stroke="hsl(var(--accent))" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              <circle cx="8" cy="11" r="1.6" fill="hsl(186 28% 57%)" />
              <circle cx="16" cy="8" r="1.6" fill="hsl(186 28% 57%)" />
            </svg>
          </div>
          <CardTitle className="font-mono text-base font-semibold uppercase tracking-[0.14em]">CPA Usage Lens</CardTitle>
          <CardDescription className="font-mono text-[11px] uppercase tracking-[0.2em] text-faint">
            账号用量分析 · 登录
          </CardDescription>
        </CardHeader>

        <CardContent className="px-7 pb-8 pt-4">
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="password" className="text-sm text-muted-foreground">
                登录密码
              </label>
              <div className="relative">
                <Lock className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-faint" />
                <Input
                  id="password"
                  type="password"
                  autoFocus
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  aria-invalid={error !== ''}
                  className="h-10 pl-8 font-mono"
                  placeholder="••••••••"
                />
              </div>
              {error !== '' && (
                <p className="flex items-center gap-1.5 text-sm text-destructive">
                  <AlertCircle className="size-4 shrink-0" />
                  {error}
                </p>
              )}
            </div>

            <Button type="submit" size="lg" disabled={loading} className="mt-1 h-10 w-full">
              {loading && <Loader2 className="size-4 animate-spin" />}
              {loading ? '登录中…' : '登录'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
