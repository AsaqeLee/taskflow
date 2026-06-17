import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { ErrorAlert } from '../components/ErrorAlert'
import { login } from '../lib/apiClient'

export function LoginPage() {
  const navigate = useNavigate()
  const [id, setId] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<unknown>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setLoading(true)
    setError(null)
    try {
      await login(id.trim(), password)
      navigate('/tasks', { replace: true })
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-sm rounded-xl border border-slate-200 bg-white p-6 shadow-sm"
      >
        <h1 className="text-xl font-semibold">登录 TaskFlow</h1>
        <p className="mt-1 text-sm text-slate-600">使用内网账号登录</p>

        <label htmlFor="login-id" className="mt-6 block text-sm font-medium">
          用户 ID
        </label>
        <input
          id="login-id"
          autoComplete="username"
          className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
          value={id}
          onChange={(e) => setId(e.target.value)}
          required
        />

        <label htmlFor="login-password" className="mt-4 block text-sm font-medium">
          密码
        </label>
        <input
          id="login-password"
          type="password"
          autoComplete="current-password"
          className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />

        <div className="mt-4">
          <ErrorAlert error={error} />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="mt-4 w-full rounded-md bg-blue-600 px-4 py-2 text-white disabled:opacity-50"
        >
          {loading ? '登录中…' : '登录'}
        </button>
      </form>
    </div>
  )
}
