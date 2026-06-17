import { ApiClientError } from '../lib/apiClient'

interface ErrorAlertProps {
  error: unknown
}

export function ErrorAlert({ error }: ErrorAlertProps) {
  if (!error) return null

  let message = '请求失败'
  let code = 'unknown_error'
  let requestId: string | undefined

  if (error instanceof ApiClientError) {
    message = error.message
    code = error.code
    requestId = error.requestId
  } else if (error instanceof Error) {
    message = error.message
  }

  return (
    <div
      role="alert"
      className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800"
    >
      <p className="font-medium">{message}</p>
      <p className="mt-1 text-red-600">
        错误码: {code}
        {requestId ? ` · request_id: ${requestId}` : ''}
      </p>
    </div>
  )
}
