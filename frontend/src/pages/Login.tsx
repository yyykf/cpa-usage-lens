import { useState } from 'react'
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
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="max-w-sm w-full rounded-2xl border border-border bg-card p-5 md:p-6 hover:border-primary/40 transition-colors">
        <h1 className="text-xl font-semibold text-foreground">CPA Usage Lens</h1>
        <p className="text-muted-foreground text-sm mt-1">账号用量分析</p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label htmlFor="password" className="block text-sm text-muted-foreground mb-1.5">
              登录密码
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="bg-muted border border-border rounded-lg px-3 py-2 w-full text-foreground"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="bg-primary text-white rounded-lg w-full py-2 disabled:opacity-60 transition-colors"
          >
            {loading ? '登录中…' : '登录'}
          </button>

          {error && <p className="text-destructive text-sm">{error}</p>}
        </form>
      </div>
    </div>
  )
}
