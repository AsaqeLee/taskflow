import { describe, expect, it } from 'vitest'
import {
  actionNeedsConfirmation,
  actionNeedsContent,
  availableActions,
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
  it('shows assign/cancel/delete to creator on open task', () => {
    const actions = availableActions(task({ status: 'open', assignee_id: '' }), owner)
    expect(actions).toEqual(expect.arrayContaining(['assign', 'cancel', 'delete']))
  })

  it('shows start to assignee on assigned task', () => {
    const actions = availableActions(task({ status: 'assigned' }), worker)
    expect(actions).toEqual(['start'])
  })

  it('shows submit to assignee on in_progress task', () => {
    const actions = availableActions(task({ status: 'in_progress' }), worker)
    expect(actions).toEqual(['submit'])
  })

  it('shows approve/reject to creator on submitted task', () => {
    const actions = availableActions(task({ status: 'submitted' }), owner)
    expect(actions).toEqual(expect.arrayContaining(['approve', 'reject']))
  })

  it('prefers backend-provided actions when present', () => {
    const actions = availableActions(
      task({ status: 'open', available_actions: ['assign'] }),
      worker,
    )
    expect(actions).toEqual(['assign'])
  })

  it('returns empty actions for unrelated user', () => {
    const outsider: User = { ...worker, id: 'u_bob' }
    const actions = availableActions(task({ status: 'assigned' }), outsider)
    expect(actions).toEqual([])
  })
})

describe('taskWorkflow action helpers', () => {
  it('requires content for submit/reject/approve/cancel/reactivate', () => {
    expect(actionNeedsContent('submit')).toBe(true)
    expect(actionNeedsContent('close')).toBe(false)
  })

  it('requires confirmation dialog for delete', () => {
    expect(actionNeedsConfirmation('delete')).toBe(true)
    expect(shouldOpenActionDialog('delete')).toBe(true)
    expect(shouldOpenActionDialog('close')).toBe(false)
  })
})
