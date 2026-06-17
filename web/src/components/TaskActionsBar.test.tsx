import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { TaskActionsBar } from './TaskActionsBar'
import type { Task, User } from '../types/api'

const owner: User = {
  id: 'u_owner',
  name: 'Owner',
  role: 'owner',
  active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const alice: User = {
  id: 'u_alice',
  name: 'Alice',
  role: 'human',
  active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const openTask: Task = {
  id: 'task_1',
  title: 'Demo',
  description: '',
  status: 'open',
  creator_id: 'u_owner',
  assignee_id: '',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('TaskActionsBar', () => {
  it('renders assign dropdown with active users', () => {
    render(
      <TaskActionsBar
        task={openTask}
        currentUser={owner}
        users={[owner, alice]}
        assigneeId=""
        onAssigneeChange={() => undefined}
        onAction={() => undefined}
      />,
    )

    expect(screen.getByLabelText('选择执行人')).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Alice (u_alice)' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: /Owner/ })).not.toBeInTheDocument()
  })

  it('enables assign after selecting assignee', async () => {
    const user = userEvent.setup()
    const onAction = vi.fn()
    let assigneeId = ''

    const { rerender } = render(
      <TaskActionsBar
        task={openTask}
        currentUser={owner}
        users={[alice]}
        assigneeId={assigneeId}
        onAssigneeChange={(value) => {
          assigneeId = value
        }}
        onAction={onAction}
      />,
    )

    expect(screen.getByRole('button', { name: '分配' })).toBeDisabled()

    await user.selectOptions(screen.getByLabelText('选择执行人'), 'u_alice')
    rerender(
      <TaskActionsBar
        task={openTask}
        currentUser={owner}
        users={[alice]}
        assigneeId={assigneeId}
        onAssigneeChange={(value) => {
          assigneeId = value
        }}
        onAction={onAction}
      />,
    )

    const assignButton = screen.getByRole('button', { name: '分配' })
    expect(assignButton).not.toBeDisabled()
    await user.click(assignButton)
    expect(onAction).toHaveBeenCalledWith('assign')
  })

  it('shows permission hint when no actions available', () => {
    const outsider: User = { ...alice, id: 'u_bob', name: 'Bob' }
    const assignedTask: Task = { ...openTask, status: 'assigned', assignee_id: 'u_alice' }

    render(
      <TaskActionsBar
        task={assignedTask}
        currentUser={outsider}
        users={[alice]}
        assigneeId=""
        onAssigneeChange={() => undefined}
        onAction={() => undefined}
      />,
    )

    expect(screen.getByText('当前状态下你没有可执行的操作')).toBeInTheDocument()
  })
})