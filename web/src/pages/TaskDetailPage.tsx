import { useCallback, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ActionDialog } from '../components/ActionDialog'
import { ErrorAlert } from '../components/ErrorAlert'
import { LoadingState } from '../components/LoadingState'
import { TaskActionsBar } from '../components/TaskActionsBar'
import { TaskBasicInfoPanel } from '../components/TaskBasicInfoPanel'
import { TaskDetailHeader } from '../components/TaskDetailHeader'
import { TaskTimelineTabs } from '../components/TaskTimelineTabs'
import { actionNeedsContent, shouldOpenActionDialog } from '../domain/taskWorkflow'
import { useAsyncData } from '../hooks/useAsyncData'
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

interface TaskPageData {
  task: Task
  records: TaskRecord[]
  auditLogs: AuditLog[]
}

async function loadTaskPage(taskId: string): Promise<TaskPageData> {
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
  const [tab, setTab] = useState<'records' | 'audit'>('records')
  const [editing, setEditing] = useState(false)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [assigneeId, setAssigneeId] = useState('')
  const [pendingAction, setPendingAction] = useState<TaskAction | null>(null)
  const [actionError, setActionError] = useState<unknown>(null)

  const taskLoader = useCallback(async () => {
    const data = await loadTaskPage(id)
    setTitle(data.task.title)
    setDescription(data.task.description)
    return data
  }, [id])

  const usersLoader = useCallback(() => fetchUsers(true), [])

  const taskState = useAsyncData<TaskPageData>(`task:${id}`, taskLoader)
  const usersState = useAsyncData<User[]>(`users:${id}`, usersLoader)

  async function handleAction(action: TaskAction, content = '') {
    const task = taskState.data?.task
    if (!task) return
    setActionError(null)

    try {
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
          navigate('/tasks', { replace: true })
          return
      }
      taskState.reload()
    } catch (err) {
      setActionError(err)
      throw err
    }
  }

  async function saveBasic() {
    const task = taskState.data?.task
    if (!task) return
    setActionError(null)
    await updateTask(task.id, title.trim(), description.trim())
    setEditing(false)
    taskState.reload()
  }

  if (!id) {
    return <ErrorAlert error={new Error('缺少任务 ID')} />
  }

  if (taskState.status === 'loading' || taskState.status === 'idle') {
    return <LoadingState />
  }

  if (taskState.status === 'error' || !taskState.data || !currentUser) {
    return <ErrorAlert error={taskState.error ?? new Error('任务不存在')} />
  }

  const { task, records, auditLogs } = taskState.data

  return (
    <div className="space-y-6">
      <TaskDetailHeader task={task} />
      <ErrorAlert error={actionError} />
      <ErrorAlert error={usersState.error} />

      <TaskActionsBar
        task={task}
        currentUser={currentUser}
        users={usersState.data ?? []}
        assigneeId={assigneeId}
        onAssigneeChange={setAssigneeId}
        onAction={(action) => {
          if (shouldOpenActionDialog(action)) {
            setPendingAction(action)
            return
          }
          handleAction(action).catch(() => undefined)
        }}
      />

      <TaskBasicInfoPanel
        task={task}
        canEdit={task.creator_id === currentUser.id}
        editing={editing}
        title={title}
        description={description}
        onToggleEdit={() => setEditing((value) => !value)}
        onTitleChange={setTitle}
        onDescriptionChange={setDescription}
        onSave={() => saveBasic().catch(setActionError)}
      />

      <TaskTimelineTabs
        tab={tab}
        records={records}
        auditLogs={auditLogs}
        onTabChange={setTab}
      />

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