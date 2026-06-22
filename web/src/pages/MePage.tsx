import { EmptyState } from '../components/EmptyState'
import { ErrorAlert } from '../components/ErrorAlert'
import { LoadingState } from '../components/LoadingState'
import { useAsyncData } from '../hooks/useAsyncData'
import { fetchMe } from '../lib/apiClient'

export function MePage() {
  const meState = useAsyncData('me', fetchMe)

  if (meState.status === 'idle' || meState.status === 'loading') {
    return <LoadingState />
  }

  const user = meState.data
  const error = meState.error

  return (
    <div className="mx-auto max-w-xl space-y-4">
      <h1 className="text-2xl font-semibold">我的信息</h1>
      <ErrorAlert error={error} />
      {user ? (
        <dl className="rounded-xl border border-slate-200 bg-white p-6 text-sm">
          <div className="grid grid-cols-1 gap-2 py-2 sm:grid-cols-3">
            <dt className="text-slate-500">ID</dt>
            <dd className="sm:col-span-2">{user.id}</dd>
          </div>
          <div className="grid grid-cols-1 gap-2 py-2 sm:grid-cols-3">
            <dt className="text-slate-500">姓名</dt>
            <dd className="sm:col-span-2">{user.name}</dd>
          </div>
          <div className="grid grid-cols-1 gap-2 py-2 sm:grid-cols-3">
            <dt className="text-slate-500">角色</dt>
            <dd className="sm:col-span-2">{user.role}</dd>
          </div>
          <div className="grid grid-cols-1 gap-2 py-2 sm:grid-cols-3">
            <dt className="text-slate-500">状态</dt>
            <dd className="sm:col-span-2">{user.active ? '活跃' : '已禁用'}</dd>
          </div>
        </dl>
      ) : (
        <EmptyState message="暂无用户信息" />
      )}
    </div>
  )
}
