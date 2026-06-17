import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ActionDialog } from '../components/ActionDialog'
import { ErrorAlert } from '../components/ErrorAlert'
import { StatusStepper } from '../components/StatusStepper'
import {
  actionLabel,
  actionNeedsConfirmation,
  actionNeedsContent,
  availableActions,
  statusLabel,
} from '../domain/taskWorkflow'
import {
  approveTask,
  assignTask,
  cancelTask,
  closeTask,
  deleteTask,
  fetchTask,
  fetchTaskAuditLogs,
  fetchTaskRecords,
  fetchUsers,
  reactivateTask,
  rejectTask,
  startTask,
  submitTask,
  updateTask,
} from '../lib/apiClient'
import { getStoredUser } from '../lib/auth'
import type { AuditLog, Task, TaskAction, TaskRecord, User } from '../types/api'

async function fetchTaskPageData(taskId: string): Promise<{
  task: Task
  records: TaskRecord[]
  auditLogs: AuditLog[]
}> {
  const [task, records, auditLogs] = await Promise.all([
    fetchTask(taskId),
    fetchTaskRecords(taskId),
    fetchTaskAuditLogs(taskId),
  ])

  return { task, records, auditLogs }
}

export function TaskDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const currentUser = getStoredUser()
  const [task, setTask] = useState<Task | null>(null)
  const [records, setRecords] = useState<TaskRecord[]>([])
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [tab, setTab] = useState<'records' | 'audit'>('records')
  const [error, setError] = useState<unknown>(null)
  const [usersError, setUsersError] = useState<unknown>(null)
  const [loadedId, setLoadedId] = useState('')
  const [reloadVersion, setReloadVersion] = useState(0)
  const [editing, setEditing] = useState(false)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [assigneeId, setAssigneeId] = useState('')
  const [pendingAction, setPendingAction] = useState<TaskAction | null>(null)

  function requestReload() {
    setError(null)
    setUsersError(null)
    setLoadedId('')
    setReloadVersion((value) => value + 1)
  }

  useEffect(() => {
    if (!id) return
    let cancelled = false

    async function loadPage() {
      const [taskResult, usersResult] = await Promise.allSettled([
        fetchTaskPageData(id),
        fetchUsers(true),
      ])

      if (cancelled) return

      if (taskResult.status === 'fulfilled') {
        const { task: nextTask, records: nextRecords, auditLogs: nextAuditLogs } = taskResult.value
        setTask(nextTask)
        setRecords(nextRecords)
        setAuditLogs(nextAuditLogs)
        setTitle(nextTask.title)
        setDescription(nextTask.description)
        setError(null)
      } else {
        setTask(null)
        setRecords([])
        setAuditLogs([])
        setError(taskResult.reason)
      }

      if (usersResult.status === 'fulfilled') {
        setUsers(usersResult.value)
        setUsersError(null)
      } else {
        setUsers([])
        setUsersError(usersResult.reason)
      }

      setLoadedId(id)
    }

    loadPage().catch((err) => {
      if (cancelled) return
      setTask(null)
      setRecords([])
      setAuditLogs([])
      setUsers([])
      setError(err)
      setUsersError(null)
      setLoadedId(id)
    })

    return () => {
      cancelled = true
    }
  }, [id, reloadVersion])

  async function handleAction(action: TaskAction, content = '') {
    if (!task) return
    switch (action) {
      case 'assign':
        await assignTask(task.id, assigneeId)
        break
      case 'start':
        await startTask(task.id)
        break
      case 'submit':
        await submitTask(task.id, content)
        break
      case 'reject':
        await rejectTask(task.id, content)
        break
      case 'approve':
        await approveTask(task.id, content)
        break
      case 'close':
        await closeTask(task.id)
        break
      case 'cancel':
        await cancelTask(task.id, content)
        break
      case 'reactivate':
        await reactivateTask(task.id, content)
        break
      case 'delete':
        await deleteTask(task.id)
        break
    }
    if (action === 'delete') {
      navigate('/tasks', { replace: true })
      return
    }
    requestReload()
  }

  async function saveBasic() {
    if (!task) return
    await updateTask(task.id, title.trim(), description.trim())
    setEditing(false)
    requestReload()
  }

  const loading = !error && loadedId !== id

  if (loading) return <p className="text-slate-600">加载中…</p>
  if (!task || !currentUser) return <ErrorAlert error={error ?? new Error('任务不存在')} />

  const actions = availableActions(task, currentUser)

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm text-slate-500">#{task.id}</p>
          <h1 className="text-2xl font-semibold">{task.title}</h1>
          <p className="mt-2 text-slate-600">{task.description || '无描述'}</p>
        </div>
        <span className="rounded-full bg-slate-100 px-3 py-1 text-sm">
          {statusLabel(task.status)}
        </span>
      </div>

      <StatusStepper status={task.status} />
      <ErrorAlert error={error} />
      <ErrorAlert error={usersError} />

      <div className="rounded-xl border border-slate-200 bg-white p-4">
        <div className="flex flex-wrap items-center gap-2">
          {actions.map((action) => {
            if (action === 'assign') {
              return (
                <div key={action} className="flex items-center gap-2">
                  <select
                    className="rounded-md border border-slate-300 px-2 py-1.5 text-sm"
                    value={assigneeId}
                    onChange={(e) => setAssigneeId(e.target.value)}
                  >
                    <option value="">选择执行人</option>
                    {users
                      .filter((u) => u.id !== currentUser.id && u.active)
                      .map((u) => (
                        <option key={u.id} value={u.id}>
                          {u.name} ({u.id})
                        </option>
                      ))}
                  </select>
                  <button
                    type="button"
                    disabled={!assigneeId}
                    onClick={() => handleAction('assign').catch(setError)}
                    className="rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white disabled:opacity-50"
                  >
                    {actionLabel(action)}
                  </button>
                </div>
              )
            }

            return (
              <button
                key={action}
                type="button"
                onClick={() => {
                  if (actionNeedsContent(action) || actionNeedsConfirmation(action)) {
                    setPendingAction(action)
                  } else {
                    handleAction(action).catch(setError)
                  }
                }}
                className={`rounded-md px-3 py-1.5 text-sm ${
                  action === 'delete'
                    ? 'border border-red-300 text-red-700 hover:bg-red-50'
                    : 'bg-blue-600 text-white'
                }`}
              >
                {actionLabel(action)}
              </button>
            )
          })}
        </div>
      </div>

      {task.creator_id === currentUser.id ? (
        <div className="rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center justify-between">
            <h2 className="font-medium">基本信息</h2>
            <button
              type="button"
              onClick={() => setEditing((v) => !v)}
              className="text-sm text-blue-700"
            >
              {editing ? '取消' : '编辑'}
            </button>
          </div>
          {editing ? (
            <div className="mt-3 space-y-3">
              <input
                className="w-full rounded-md border border-slate-300 px-3 py-2"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
              />
              <textarea
                className="w-full rounded-md border border-slate-300 px-3 py-2"
                rows={4}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
              <button
                type="button"
                onClick={() => saveBasic().catch(setError)}
                className="rounded-md bg-blue-600 px-3 py-1.5 text-sm text-white"
              >
                保存
              </button>
            </div>
          ) : null}
        </div>
      ) : null}

      <div className="rounded-xl border border-slate-200 bg-white">
        <div className="flex border-b border-slate-200">
          <button
            type="button"
            onClick={() => setTab('records')}
            className={`px-4 py-3 text-sm ${tab === 'records' ? 'border-b-2 border-blue-600 font-medium' : 'text-slate-600'}`}
          >
            记录
          </button>
          <button
            type="button"
            onClick={() => setTab('audit')}
            className={`px-4 py-3 text-sm ${tab === 'audit' ? 'border-b-2 border-blue-600 font-medium' : 'text-slate-600'}`}
          >
            审计
          </button>
        </div>
        <div className="divide-y divide-slate-100 p-4 text-sm">
          {tab === 'records' ? (
            records.length === 0 ? (
              <p className="text-slate-500">暂无记录</p>
            ) : (
              records.map((record) => (
                <div key={record.id} className="py-3">
                  <p className="font-medium">
                    {record.type} · {record.author_id}
                  </p>
                  <p className="mt-1 text-slate-700">{record.content}</p>
                  <p className="mt-1 text-xs text-slate-500">
                    {new Date(record.created_at).toLocaleString()}
                  </p>
                </div>
              ))
            )
          ) : auditLogs.length === 0 ? (
            <p className="text-slate-500">暂无审计日志</p>
          ) : (
            auditLogs.map((log) => (
              <div key={log.id} className="py-3">
                <p className="font-medium">
                  {log.action} · {log.actor_id}
                </p>
                <p className="mt-1 text-slate-600">
                  {log.from_status} → {log.to_status}
                </p>
                <p className="mt-1 text-xs text-slate-500">
                  {new Date(log.created_at).toLocaleString()}
                </p>
              </div>
            ))
          )}
        </div>
      </div>

      <ActionDialog
        key={pendingAction ?? 'closed'}
        action={pendingAction}
        needsContent={pendingAction ? actionNeedsContent(pendingAction) : false}
        onClose={() => setPendingAction(null)}
        onConfirm={(content) => {
          if (!pendingAction) return Promise.resolve()
          return handleAction(pendingAction, content)
        }}
      />
    </div>
  )
}
