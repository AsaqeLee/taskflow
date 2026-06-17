import { WORKFLOW_STEPS, statusLabel } from '../domain/taskWorkflow'
import type { TaskStatus } from '../types/api'

interface StatusStepperProps {
  status: TaskStatus
}

export function StatusStepper({ status }: StatusStepperProps) {
  if (status === 'cancelled' || status === 'deleted') {
    return (
      <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
        当前状态: {statusLabel(status)}
      </div>
    )
  }

  const currentIndex = WORKFLOW_STEPS.indexOf(status)

  return (
    <ol className="flex flex-wrap gap-2">
      {WORKFLOW_STEPS.map((step, index) => {
        const active = index === currentIndex
        const done = index < currentIndex
        return (
          <li
            key={step}
            className={`rounded-full px-3 py-1 text-xs font-medium ${
              active
                ? 'bg-blue-600 text-white'
                : done
                  ? 'bg-blue-100 text-blue-800'
                  : 'bg-slate-100 text-slate-500'
            }`}
          >
            {statusLabel(step)}
          </li>
        )
      })}
    </ol>
  )
}