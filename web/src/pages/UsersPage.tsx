import { useCallback, useMemo, useState, type FormEvent } from 'react'
import { EmptyState } from '../components/EmptyState'
import { ErrorAlert } from '../components/ErrorAlert'
import { LoadingState } from '../components/LoadingState'
import { useAsyncData } from '../hooks/useAsyncData'
import { getStoredUser } from '../lib/auth'
import {
  createUser,
  createUserAPIKey,
  disableUser,
  fetchUserAPIKeys,
  fetchUsers,
  revokeUserAPIKey,
  revokeUserSessions,
} from '../lib/apiClient'
import type { APIKey, UserRole } from '../types/api'

const roleOptions: Array<{ value: UserRole; label: string }> = [
  { value: 'human', label: 'Human' },
  { value: 'agent', label: 'Agent' },
  { value: 'owner', label: 'Owner' },
]

function formatDateTime(value?: string): string {
  if (!value) {
    return '未设置'
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleString()
}

function apiKeyStatus(key: APIKey): string {
  if (key.revoked_at) {
    return '已吊销'
  }

  if (key.expires_at && new Date(key.expires_at).getTime() < Date.now()) {
    return '已过期'
  }

  return '有效'
}

function normalizeExpiresAt(value: string): string | undefined {
  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }

  return new Date(trimmed).toISOString()
}

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
  const [apiKeyUserID, setAPIKeyUserID] = useState('')
  const [apiKeyName, setAPIKeyName] = useState('')
  const [apiKeyExpiresAt, setAPIKeyExpiresAt] = useState('')
  const [apiKeySubmitting, setAPIKeySubmitting] = useState(false)
  const [revealedAPIKey, setRevealedAPIKey] = useState<string | null>(null)

  const loadUsers = useCallback(() => fetchUsers(activeOnly), [activeOnly])
  const usersState = useAsyncData(`users:${activeOnly ? 'active' : 'all'}`, loadUsers)

  const sortedUsers = useMemo(() => {
    const users = usersState.data ?? []
    return [...users].sort((left, right) => left.id.localeCompare(right.id))
  }, [usersState.data])

  const effectiveAPIKeyUserID = useMemo(() => {
    if (sortedUsers.some((user) => user.id === apiKeyUserID)) {
      return apiKeyUserID
    }

    return sortedUsers[0]?.id ?? ''
  }, [apiKeyUserID, sortedUsers])

  const loadAPIKeys = useCallback(() => {
    if (!effectiveAPIKeyUserID) {
      return Promise.resolve([])
    }

    return fetchUserAPIKeys(effectiveAPIKeyUserID)
  }, [effectiveAPIKeyUserID])
  const apiKeysState = useAsyncData(`api-keys:${effectiveAPIKeyUserID || 'none'}`, loadAPIKeys)

  const managedUser = useMemo(
    () => sortedUsers.find((user) => user.id === effectiveAPIKeyUserID) ?? null,
    [effectiveAPIKeyUserID, sortedUsers],
  )

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

  async function handleCreateAPIKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!effectiveAPIKeyUserID) {
      return
    }

    setAPIKeySubmitting(true)
    setError(null)
    setNotice(null)

    try {
      const created = await createUserAPIKey(
        effectiveAPIKeyUserID,
        apiKeyName.trim(),
        normalizeExpiresAt(apiKeyExpiresAt),
      )

      setAPIKeyName('')
      setAPIKeyExpiresAt('')
      setRevealedAPIKey(created.key)
      setNotice(`已为 ${effectiveAPIKeyUserID} 创建 API key`)
      apiKeysState.reload()
    } catch (err) {
      setError(err)
    } finally {
      setAPIKeySubmitting(false)
    }
  }

  async function handleRevokeAPIKey(keyID: string) {
    if (!effectiveAPIKeyUserID) {
      return
    }

    setPendingAction(`api-key-revoke:${keyID}`)
    setError(null)
    setNotice(null)

    try {
      await revokeUserAPIKey(effectiveAPIKeyUserID, keyID)
      setNotice(`已吊销 API key ${keyID}`)
      apiKeysState.reload()
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
              创建内网账号、管理 API key、禁用账号、撤销刷新会话。
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

      <section className="rounded-xl border border-slate-200 bg-white p-6">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold">API Key 管理</h2>
            <p className="mt-1 text-sm text-slate-600">
              为 Hermes 等无人值守账号生成 Bearer key。新密钥只会展示一次。
            </p>
          </div>
          {managedUser ? (
            <button
              type="button"
              onClick={apiKeysState.reload}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-xs text-slate-700"
            >
              刷新密钥列表
            </button>
          ) : null}
        </div>

        {sortedUsers.length === 0 ? (
          <div className="mt-4">
            <EmptyState message="暂无可管理用户" />
          </div>
        ) : (
          <>
            <form className="mt-4 grid gap-4 md:grid-cols-2 lg:grid-cols-4" onSubmit={handleCreateAPIKey}>
              <label className="text-sm font-medium text-slate-700">
                目标用户
                <select
                  className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
                  value={effectiveAPIKeyUserID}
                  onChange={(event) => {
                    setAPIKeyUserID(event.target.value)
                    setRevealedAPIKey(null)
                  }}
                >
                  {sortedUsers.map((user) => (
                    <option key={user.id} value={user.id}>
                      {user.id} · {user.name} · {user.role}
                    </option>
                  ))}
                </select>
              </label>

              <label className="text-sm font-medium text-slate-700">
                Key 名称
                <input
                  className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
                  value={apiKeyName}
                  onChange={(event) => setAPIKeyName(event.target.value)}
                  placeholder="Hermes prod"
                  required
                />
              </label>

              <label className="text-sm font-medium text-slate-700">
                过期时间
                <input
                  type="datetime-local"
                  className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
                  value={apiKeyExpiresAt}
                  onChange={(event) => setAPIKeyExpiresAt(event.target.value)}
                />
              </label>

              <div className="flex items-end">
                <button
                  type="submit"
                  disabled={apiKeySubmitting || !effectiveAPIKeyUserID}
                  className="w-full rounded-md bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-60"
                >
                  {apiKeySubmitting ? '创建中…' : '创建 API key'}
                </button>
              </div>
            </form>

            {revealedAPIKey ? (
              <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                <p className="font-medium">新 API key（仅显示一次）</p>
                <code className="mt-2 block overflow-x-auto rounded-md bg-white px-3 py-2 font-mono text-xs text-slate-900">
                  {revealedAPIKey}
                </code>
              </div>
            ) : null}

            <div className="mt-4">
              {apiKeysState.status === 'loading' ? (
                <LoadingState />
              ) : apiKeysState.status === 'error' ? (
                <ErrorAlert error={apiKeysState.error} />
              ) : !managedUser ? (
                <EmptyState message="请选择用户" />
              ) : apiKeysState.data && apiKeysState.data.length > 0 ? (
                <div className="overflow-x-auto rounded-lg border border-slate-200">
                  <table className="min-w-[920px] text-left text-sm sm:min-w-full">
                    <thead className="bg-slate-50 text-slate-600">
                      <tr>
                        <th className="px-4 py-3">前缀</th>
                        <th className="px-4 py-3">名称</th>
                        <th className="px-4 py-3">状态</th>
                        <th className="px-4 py-3">上次使用</th>
                        <th className="px-4 py-3">过期时间</th>
                        <th className="px-4 py-3">创建时间</th>
                        <th className="px-4 py-3">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {apiKeysState.data.map((key) => {
                        const revoking = pendingAction === `api-key-revoke:${key.id}`

                        return (
                          <tr key={key.id} className="border-t border-slate-100 align-top">
                            <td className="px-4 py-3 font-mono text-xs text-slate-900">{key.key_prefix}</td>
                            <td className="px-4 py-3">{key.name}</td>
                            <td className="px-4 py-3">{apiKeyStatus(key)}</td>
                            <td className="px-4 py-3">{formatDateTime(key.last_used_at)}</td>
                            <td className="px-4 py-3">{formatDateTime(key.expires_at)}</td>
                            <td className="px-4 py-3">{formatDateTime(key.created_at)}</td>
                            <td className="px-4 py-3">
                              <button
                                type="button"
                                disabled={revoking || Boolean(key.revoked_at)}
                                onClick={() => {
                                  handleRevokeAPIKey(key.id).catch(() => undefined)
                                }}
                                className="rounded-md border border-red-300 px-3 py-1.5 text-xs text-red-700 disabled:opacity-60"
                              >
                                {revoking ? '吊销中…' : key.revoked_at ? '已吊销' : '吊销 Key'}
                              </button>
                            </td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </div>
              ) : (
                <EmptyState message={`用户 ${managedUser.id} 暂无 API key`} />
              )}
            </div>
          </>
        )}
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

      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
        <div className="border-b border-slate-100 px-6 py-4">
          <h2 className="text-lg font-semibold">现有用户</h2>
        </div>
        {usersState.status === 'loading' ? (
          <div className="p-6">
            <LoadingState />
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
          <div className="overflow-x-auto">
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
                      <td className="px-4 py-3">{formatDateTime(user.updated_at)}</td>
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
          </div>
        )}
      </section>
    </div>
  )
}
