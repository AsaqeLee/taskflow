import { actionLabel, availableActions } from '../domain/taskWorkflow'
import type { Task, TaskAction, User } from '../types/api'

interface TaskActionsBarProps {
  task: Task
  currentUser: User
  users: User[]
  assigneeId: string
  onAssigneeChange: (assigneeId: string) => void
  onAction: (action: TaskAction) => void
}

export function TaskActionsBar({
  task,
  currentUser,
  users,
  assigneeId,
  onAssigneeChange,
  onAction,
}: TaskActionsBarProps) {
  const actions = availableActions(task, currentUser)

  if (actions.length === 0) {
    return (
      <div className="rounded-xl border border-slate-200 bg-white p-4 text-sm text-slate-500">
        当前状态下你没有可执行的操作
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
        {actions.map((action) => {
          if (action === 'assign') {
            return (
              <div key={action} className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <select
                  aria-label="选择执行人"
                  className="w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm sm:w-auto"
                  value={assigneeId}
                  onChange={(e) => onAssigneeChange(e.target.value)}
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
                  onClick={() => onAction('assign')}
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
              onClick={() => onAction(action)}
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
  )
}