import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { ErrorAlert } from '../components/ErrorAlert'
import { confirmPasswordReset } from '../lib/apiClient'

export function PasswordResetConfirmPage() {
  const [searchParams] = useSearchParams()
  const [id, setID] = useState(searchParams.get('id') ?? '')
  const [token, setToken] = useState(searchParams.get('token') ?? '')
  const [newPassword, setNewPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [completed, setCompleted] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError(null)
    setCompleted(false)
    try {
      await confirmPasswordReset(id.trim(), token.trim(), newPassword)
      setCompleted(true)
      setNewPassword('')
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-8">
      <div className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <h1 className="text-xl font-semibold">确认密码重置</h1>
        <p className="mt-1 text-sm text-slate-600">
          输入用户 ID、重置令牌和新密码，完成密码更新。
        </p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <label htmlFor="reset-confirm-id" className="block text-sm font-medium">
            用户 ID
          </label>
          <input
            id="reset-confirm-id"
            autoComplete="username"
            className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            value={id}
            onChange={(event) => setID(event.target.value)}
            required
          />

          <label htmlFor="reset-confirm-token" className="block text-sm font-medium">
            重置令牌
          </label>
          <textarea
            id="reset-confirm-token"
            className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            rows={4}
            value={token}
            onChange={(event) => setToken(event.target.value)}
            required
          />

          <label htmlFor="reset-confirm-password" className="block text-sm font-medium">
            新密码
          </label>
          <input
            id="reset-confirm-password"
            type="password"
            autoComplete="new-password"
            minLength={8}
            className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            value={newPassword}
            onChange={(event) => setNewPassword(event.target.value)}
            required
          />

          <ErrorAlert error={error} />

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-blue-600 px-4 py-2 text-white disabled:opacity-50"
          >
            {loading ? '提交中…' : '确认重置'}
          </button>
        </form>

        {completed ? (
          <div
            role="status"
            className="mt-4 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800"
          >
            密码已重置，请使用新密码重新登录。
          </div>
        ) : null}

        <div className="mt-4 flex flex-col gap-2 text-sm">
          <Link to="/password-reset/request" className="text-blue-700 hover:text-blue-900">
            重新发起密码重置
          </Link>
          <Link to="/login" className="text-slate-600 hover:text-slate-900">
            返回登录
          </Link>
        </div>
      </div>
    </div>
  )
}
