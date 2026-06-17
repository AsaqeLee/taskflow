import { StatusStepper } from './StatusStepper'
import { statusLabel } from '../domain/taskWorkflow'
import type { Task } from '../types/api'

interface TaskDetailHeaderProps {
  task: Task
}

export function TaskDetailHeader({ task }: TaskDetailHeaderProps) {
  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <p className="text-sm text-slate-500">#{task.id}</p>
          <h1 className="text-xl font-semibold sm:text-2xl">{task.title}</h1>
          <p className="mt-2 text-slate-600">{task.description || '无描述'}</p>
        </div>
        <span className="w-fit rounded-full bg-slate-100 px-3 py-1 text-sm">
          {statusLabel(task.status)}
        </span>
      </div>
      <StatusStepper status={task.status} />
    </div>
  )
}