import { describe, expect, it } from 'vitest'
import {
  actionNeedsConfirmation,
  actionNeedsContent,
  availableActions,
  shouldLoadAssignableUsers,
  shouldOpenActionDialog,
} from './taskWorkflow'
import type { Task, User } from '../types/api'

const owner: User = {
  id: 'u_owner',
  name: 'Owner',
  role: 'owner',
  active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const worker: User = {
  id: 'u_alice',
  name: 'Alice',
  role: 'human',
  active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function task(partial: Partial<Task> & Pick<Task, 'status'>): Task {
  return {
    id: 'task_1',
    title: 'Demo',
    description: '',
    creator_id: 'u_owner',
    assignee_id: 'u_alice',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...partial,
  }
}

describe('taskWorkflow.availableActions', () => {
  it('shows assign/cancel/delete for creator on open task', () => {
    const actions = availableActions(task({ status: 'open', assignee_id: '' }), owner)
    expect(actions).toEqual(expect.arrayContaining(['assign', 'cancel', 'delete']))
  })

  it('shows start for assignee on assigned task', () => {
    const actions = availableActions(task({ status: 'assigned' }), worker)
    expect(actions).toEqual(['start'])
  })

  it('shows submit for assignee on in_progress task', () => {
    const actions = availableActions(task({ status: 'in_progress' }), worker)
    expect(actions).toEqual(['submit'])
  })

  it('shows approve/reject for creator on submitted task', () => {
    const actions = availableActions(task({ status: 'submitted' }), owner)
    expect(actions).toEqual(expect.arrayContaining(['approve', 'reject']))
  })

  it('prefers backend-provided available_actions when present', () => {
    const actions = availableActions(
      task({ status: 'assigned', available_actions: ['cancel', 'delete'] }),
      worker,
    )
    expect(actions).toEqual(['cancel', 'delete'])
  })
})

describe('taskWorkflow.shouldLoadAssignableUsers', () => {
  it('loads users only when assign action is available', () => {
    expect(shouldLoadAssignableUsers(task({ status: 'open', assignee_id: '' }), owner)).toBe(true)
    expect(shouldLoadAssignableUsers(task({ status: 'submitted' }), owner)).toBe(false)
    expect(shouldLoadAssignableUsers(task({ status: 'assigned' }), worker)).toBe(false)
  })
})

describe('taskWorkflow.dialog behavior', () => {
  it('requires content for submission and review actions', () => {
    expect(actionNeedsContent('submit')).toBe(true)
    expect(actionNeedsContent('reject')).toBe(true)
    expect(actionNeedsContent('approve')).toBe(true)
    expect(actionNeedsContent('cancel')).toBe(true)
    expect(actionNeedsContent('reactivate')).toBe(true)
    expect(actionNeedsContent('close')).toBe(false)
  })

  it('requires confirmation only for delete', () => {
    expect(actionNeedsConfirmation('delete')).toBe(true)
    expect(shouldOpenActionDialog('delete')).toBe(true)
    expect(shouldOpenActionDialog('close')).toBe(false)
  })
})
