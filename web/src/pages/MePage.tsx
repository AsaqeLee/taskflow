import { useEffect, useState } from 'react'
import { ErrorAlert } from '../components/ErrorAlert'
import { fetchMe } from '../lib/apiClient'
import type { User } from '../types/api'

export function MePage() {
  const [user, setUser] = useState<User | null>(null)
  const [error, setError] = useState<unknown>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchMe()
      .then(setUser)
      .catch(setError)
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <p className="text-slate-600">加载中…</p>
  }

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
      ) : null}
    </div>
  )
}
