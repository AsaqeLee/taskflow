import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as apiClient from '../lib/apiClient'
import type { Task } from '../types/api'
import { TaskListPage } from './TaskListPage'

function buildTask(id: string, title: string, status: Task['status']): Task {
  return {
    id,
    title,
    description: '',
    status,
    creator_id: 'u_owner',
    assignee_id: 'u_worker',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }
}

describe('TaskListPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders empty state after loading with no tasks', async () => {
    vi.spyOn(apiClient, 'fetchTasks').mockResolvedValue({
      tasks: [],
      total: 0,
      page: 1,
      page_size: 20,
    })

    render(
      <MemoryRouter>
        <TaskListPage />
      </MemoryRouter>,
    )

    expect(await screen.findByText('暂无任务')).toBeInTheDocument()
  })

  it('submits filters and loads the next page', async () => {
    const user = userEvent.setup()
    const firstTask = buildTask('task_alpha_001', 'Alpha first', 'assigned')
    const secondTask = buildTask('task_alpha_002', 'Alpha second', 'assigned')
    const fetchSpy = vi.spyOn(apiClient, 'fetchTasks').mockImplementation(async (query = {}) => {
      if (query.q === 'alpha' && query.status === 'assigned' && query.page === 2) {
        return {
          tasks: [secondTask],
          total: 21,
          page: 2,
          page_size: 20,
        }
      }
      if (query.q === 'alpha' && query.status === 'assigned') {
        return {
          tasks: [firstTask],
          total: 21,
          page: 1,
          page_size: 20,
        }
      }
      return {
        tasks: [],
        total: 0,
        page: 1,
        page_size: 20,
      }
    })

    render(
      <MemoryRouter>
        <TaskListPage />
      </MemoryRouter>,
    )

    expect(await screen.findByText('暂无任务')).toBeInTheDocument()

    await user.type(screen.getByLabelText('搜索'), 'alpha')
    await user.selectOptions(screen.getByLabelText('状态'), 'assigned')
    await user.click(screen.getByRole('button', { name: '查询' }))

    expect(await screen.findByText('Alpha first')).toBeInTheDocument()
    await waitFor(() => {
      expect(fetchSpy).toHaveBeenLastCalledWith({
        q: 'alpha',
        status: 'assigned',
        page: 1,
        page_size: 20,
      })
    })

    await user.click(screen.getByRole('button', { name: '下一页' }))

    expect(await screen.findByText('Alpha second')).toBeInTheDocument()
    await waitFor(() => {
      expect(fetchSpy).toHaveBeenLastCalledWith({
        q: 'alpha',
        status: 'assigned',
        page: 2,
        page_size: 20,
      })
    })
  })
})
