import type { Task, TaskAction, TaskStatus, User } from '../types/api'

const CREATOR_ACTIONS: Partial<Record<TaskStatus, TaskAction[]>> = {
  open: ['assign', 'cancel', 'delete'],
  assigned: ['cancel', 'delete'],
  in_progress: ['cancel'],
  submitted: ['approve', 'reject'],
  approved: ['close'],
  cancelled: ['reactivate'],
  completed: ['reactivate'],
}

const ASSIGNEE_ACTIONS: Partial<Record<TaskStatus, TaskAction[]>> = {
  assigned: ['start'],
  in_progress: ['submit'],
}

export const WORKFLOW_STEPS: TaskStatus[] = [
  'open',
  'assigned',
  'in_progress',
  'submitted',
  'approved',
  'completed',
]

export function isOwner(user: User): boolean {
  return user.role === 'owner'
}

export function isCreator(task: Task, user: User): boolean {
  return task.creator_id === user.id
}

export function isAssignee(task: Task, user: User): boolean {
  return task.assignee_id === user.id
}

export function availableActions(task: Task, user: User): TaskAction[] {
  if (task.available_actions) {
    return task.available_actions
  }

  const actions = new Set<TaskAction>()
  const status = task.status

  if (isCreator(task, user)) {
    for (const action of CREATOR_ACTIONS[status] ?? []) {
      actions.add(action)
    }
  }

  if (isAssignee(task, user)) {
    for (const action of ASSIGNEE_ACTIONS[status] ?? []) {
      actions.add(action)
    }
  }

  return Array.from(actions)
}

export function actionNeedsContent(action: TaskAction): boolean {
  return ['submit', 'reject', 'approve', 'cancel', 'reactivate'].includes(action)
}

export function actionNeedsConfirmation(action: TaskAction): boolean {
  return action === 'delete'
}

export function shouldOpenActionDialog(action: TaskAction): boolean {
  return actionNeedsContent(action) || actionNeedsConfirmation(action)
}

export function actionLabel(action: TaskAction): string {
  const labels: Record<TaskAction, string> = {
    assign: '分配',
    start: '开始',
    submit: '提交',
    reject: '驳回',
    approve: '审批通过',
    close: '关闭',
    cancel: '取消',
    reactivate: '重新激活',
    delete: '删除',
  }
  return labels[action] ?? action
}

export function statusLabel(status: TaskStatus): string {
  const labels: Record<TaskStatus, string> = {
    open: '待处理',
    assigned: '已分配',
    in_progress: '进行中',
    submitted: '已提交',
    approved: '已审批',
    completed: '已完成',
    cancelled: '已取消',
    deleted: '已删除',
  }
  return labels[status] ?? status
}
