import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { availableActions } from './taskWorkflow'
import type { Task, TaskAction, TaskStatus, User } from '../types/api'

type ActionMatrixCase = {
  name: string
  status: TaskStatus
  actor: 'creator' | 'assignee' | 'other' | 'creator_and_assignee'
  actions: TaskAction[]
}

type ActionMatrixFile = {
  cases: ActionMatrixCase[]
}

const creator: User = {
  id: 'u_creator',
  name: 'Creator',
  role: 'owner',
  active: true,
  created_at: '2026-08-13T00:00:00Z',
  updated_at: '2026-08-13T00:00:00Z',
}

const assignee: User = {
  id: 'u_assignee',
  name: 'Assignee',
  role: 'human',
  active: true,
  created_at: '2026-08-13T00:00:00Z',
  updated_at: '2026-08-13T00:00:00Z',
}

const other: User = {
  id: 'u_other',
  name: 'Other',
  role: 'human',
  active: true,
  created_at: '2026-08-13T00:00:00Z',
  updated_at: '2026-08-13T00:00:00Z',
}

function loadContract(): ActionMatrixCase[] {
  const path = resolve(process.cwd(), '../testdata/task_action_matrix.json')
  const file = JSON.parse(readFileSync(path, 'utf8')) as ActionMatrixFile
  return file.cases
}

function taskFor(tc: ActionMatrixCase): { task: Task; user: User } {
  let user = creator
  let assigneeId = assignee.id

  if (tc.status === 'open' && tc.actor !== 'assignee' && tc.actor !== 'creator_and_assignee') {
    assigneeId = ''
  }

  switch (tc.actor) {
    case 'creator':
      user = creator
      break
    case 'assignee':
      user = assignee
      break
    case 'other':
      user = other
      break
    case 'creator_and_assignee':
      user = creator
      assigneeId = creator.id
      break
  }

  return {
    user,
    task: {
      id: 'task_contract',
      title: 'Contract task',
      description: 'testdata',
      status: tc.status,
      creator_id: creator.id,
      assignee_id: assigneeId,
      created_at: '2026-08-13T00:00:00Z',
      updated_at: '2026-08-13T00:00:00Z',
    },
  }
}

describe('taskWorkflow fallback matches shared FE/BE contract', () => {
  const cases = loadContract()

  it('has contract cases', () => {
    expect(cases.length).toBeGreaterThan(0)
  })

  it.each(cases)('$name', (tc) => {
    const { task, user } = taskFor(tc)
    expect(availableActions(task, user)).toEqual(tc.actions)
  })
})
