import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { ErrorAlert } from '../components/ErrorAlert'
import { requestPasswordReset } from '../lib/apiClient'

interface RequestResult {
  id: string
  resetToken?: string
}

export function PasswordResetRequestPage() {
  const [id, setID] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const [result, setResult] = useState<RequestResult | null>(null)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError(null)
    setResult(null)
    try {
      const normalizedID = id.trim()
      const response = await requestPasswordReset(normalizedID)
      setResult({
        id: normalizedID,
        resetToken: response.reset_token,
      })
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }

  const confirmHref = result?.resetToken
    ? `/password-reset/confirm?${new URLSearchParams({
        id: result.id,
        token: result.resetToken,
      }).toString()}`
    : '/password-reset/confirm'

  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-8">
      <div className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <h1 className="text-xl font-semibold">发起密码重置</h1>
        <p className="mt-1 text-sm text-slate-600">
          输入用户 ID。系统会生成重置令牌并通过已配置渠道投递。
        </p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <label htmlFor="reset-request-id" className="block text-sm font-medium">
            用户 ID
          </label>
          <input
            id="reset-request-id"
            autoComplete="username"
            className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
            value={id}
            onChange={(event) => setID(event.target.value)}
            required
          />

          <ErrorAlert error={error} />

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-blue-600 px-4 py-2 text-white disabled:opacity-50"
          >
            {loading ? '提交中…' : '发送重置请求'}
          </button>
        </form>

        {result ? (
          <div
            role="status"
            className="mt-4 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800"
          >
            <p>已接受密码重置请求，请通过已配置投递渠道获取令牌。</p>
            {result.resetToken ? (
              <>
                <p className="mt-2 font-medium">当前环境直接返回了调试令牌：</p>
                <code className="mt-2 block break-all rounded bg-emerald-100 px-3 py-2 text-xs">
                  {result.resetToken}
                </code>
              </>
            ) : null}
          </div>
        ) : null}

        <div className="mt-4 flex flex-col gap-2 text-sm">
          <Link to={confirmHref} className="text-blue-700 hover:text-blue-900">
            前往确认重置
          </Link>
          <Link to="/login" className="text-slate-600 hover:text-slate-900">
            返回登录
          </Link>
        </div>
      </div>
    </div>
  )
}
