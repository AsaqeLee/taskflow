import { useCallback, useMemo, useState, type FormEvent } from 'react'
import { EmptyState } from '../components/EmptyState'
import { ErrorAlert } from '../components/ErrorAlert'
import { LoadingState } from '../components/LoadingState'
import { useAsyncData } from '../hooks/useAsyncData'
import { getStoredUser } from '../lib/auth'
import {
  createUser,
  disableUser,
  fetchUsers,
  revokeUserSessions,
} from '../lib/apiClient'
import type { UserRole } from '../types/api'

const roleOptions: Array<{ value: UserRole; label: string }> = [
  { value: 'human', label: 'Human' },
  { value: 'agent', label: 'Agent' },
  { value: 'owner', label: 'Owner' },
]

export function UsersPage() {
  const currentUser = getStoredUser()
  const [activeOnly, setActiveOnly] = useState(false)
  const [createID, setCreateID] = useState('')
  const [createName, setCreateName] = useState('')
  const [createRole, setCreateRole] = useState<UserRole>('human')
  const [createPassword, setCreatePassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [pendingAction, setPendingAction] = useState<string | null>(null)
  const [error, setError] = useState<unknown>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const loadUsers = useCallback(() => fetchUsers(activeOnly), [activeOnly])
  const usersState = useAsyncData(`users:${activeOnly ? 'active' : 'all'}`, loadUsers)

  const sortedUsers = useMemo(() => {
    const users = usersState.data ?? []
    return [...users].sort((left, right) => left.id.localeCompare(right.id))
  }, [usersState.data])

  if (!currentUser) {
    return <ErrorAlert error={new Error('缺少当前登录用户信息')} />
  }

  if (currentUser.role !== 'owner') {
    return <ErrorAlert error={new Error('仅 owner 可访问用户管理')} />
  }

  async function handleCreateUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError(null)
    setNotice(null)
    try {
      await createUser(createID.trim(), createName.trim(), createRole, createPassword)
      setCreateID('')
      setCreateName('')
      setCreateRole('human')
      setCreatePassword('')
      setNotice('用户已创建')
      usersState.reload()
    } catch (err) {
      setError(err)
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDisableUser(userID: string) {
    setPendingAction(`disable:${userID}`)
    setError(null)
    setNotice(null)
    try {
      await disableUser(userID)
      setNotice(`已禁用用户 ${userID}`)
      usersState.reload()
    } catch (err) {
      setError(err)
    } finally {
      setPendingAction(null)
    }
  }

  async function handleRevokeSessions(userID: string) {
    setPendingAction(`revoke:${userID}`)
    setError(null)
    setNotice(null)
    try {
      await revokeUserSessions(userID)
      setNotice(`已撤销 ${userID} 的刷新会话`)
    } catch (err) {
      setError(err)
    } finally {
      setPendingAction(null)
    }
  }

  return (
    <div className="space-y-6">
      <section className="rounded-xl border border-slate-200 bg-white p-6">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-2xl font-semibold">用户管理</h1>
            <p className="mt-1 text-sm text-slate-600">
              创建内网账号、禁用账号、撤销刷新会话。
            </p>
          </div>
          <label className="inline-flex items-center gap-2 text-sm text-slate-700">
            <input
              type="checkbox"
              checked={activeOnly}
              onChange={(event) => setActiveOnly(event.target.checked)}
            />
            仅显示活跃用户
          </label>
        </div>
      </section>

      <section className="rounded-xl border border-slate-200 bg-white p-6">
        <h2 className="text-lg font-semibold">新建用户</h2>
        <form className="mt-4 grid gap-4 md:grid-cols-2" onSubmit={handleCreateUser}>
          <label className="text-sm font-medium text-slate-700">
            用户 ID
            <input
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
              value={createID}
              onChange={(event) => setCreateID(event.target.value)}
              required
            />
          </label>
          <label className="text-sm font-medium text-slate-700">
            姓名
            <input
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
              value={createName}
              onChange={(event) => setCreateName(event.target.value)}
              required
            />
          </label>
          <label className="text-sm font-medium text-slate-700">
            角色
            <select
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
              value={createRole}
              onChange={(event) => setCreateRole(event.target.value as UserRole)}
            >
              {roleOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium text-slate-700">
            初始密码
            <input
              type="password"
              minLength={12}
              className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
              value={createPassword}
              onChange={(event) => setCreatePassword(event.target.value)}
              required
            />
          </label>
          <div className="md:col-span-2">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white disabled:opacity-60"
            >
              {submitting ? '创建中…' : '创建用户'}
            </button>
          </div>
        </form>
      </section>

      <ErrorAlert error={error} />
      <ErrorAlert error={usersState.error} />
      {notice ? (
        <div
          role="status"
          className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800"
        >
          {notice}
        </div>
      ) : null}

      <section className="overflow-x-auto rounded-xl border border-slate-200 bg-white">
        {usersState.status === 'idle' || usersState.status === 'loading' ? (
          <div className="p-6">
            <LoadingState label="加载用户中…" />
          </div>
        ) : usersState.status === 'error' ? (
          <div className="p-6">
            <EmptyState message="用户列表加载失败" />
          </div>
        ) : sortedUsers.length === 0 ? (
          <div className="p-6">
            <EmptyState message="暂无用户" />
          </div>
        ) : (
          <table className="min-w-[840px] text-left text-sm sm:min-w-full">
            <thead className="bg-slate-50 text-slate-600">
              <tr>
                <th className="px-4 py-3">ID</th>
                <th className="px-4 py-3">姓名</th>
                <th className="px-4 py-3">角色</th>
                <th className="px-4 py-3">状态</th>
                <th className="px-4 py-3">更新时间</th>
                <th className="px-4 py-3">操作</th>
              </tr>
            </thead>
            <tbody>
              {sortedUsers.map((user) => {
                const disabling = pendingAction === `disable:${user.id}`
                const revoking = pendingAction === `revoke:${user.id}`
                return (
                  <tr key={user.id} className="border-t border-slate-100 align-top">
                    <td className="px-4 py-3 font-medium text-slate-900">{user.id}</td>
                    <td className="px-4 py-3">{user.name}</td>
                    <td className="px-4 py-3">{user.role}</td>
                    <td className="px-4 py-3">
                      {user.active ? '活跃' : `已禁用${user.disabled_by ? ` / ${user.disabled_by}` : ''}`}
                    </td>
                    <td className="px-4 py-3">{new Date(user.updated_at).toLocaleString()}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          disabled={revoking || disabling}
                          onClick={() => {
                            handleRevokeSessions(user.id).catch(() => undefined)
                          }}
                          className="rounded-md border border-slate-300 px-3 py-1.5 text-xs text-slate-700 disabled:opacity-60"
                        >
                          {revoking ? '撤销中…' : '撤销会话'}
                        </button>
                        {user.active && user.id !== currentUser.id ? (
                          <button
                            type="button"
                            disabled={disabling || revoking}
                            onClick={() => {
                              handleDisableUser(user.id).catch(() => undefined)
                            }}
                            className="rounded-md border border-red-300 px-3 py-1.5 text-xs text-red-700 disabled:opacity-60"
                          >
                            {disabling ? '禁用中…' : '禁用账号'}
                          </button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </section>
    </div>
  )
}
