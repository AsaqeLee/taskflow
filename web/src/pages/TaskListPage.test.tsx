import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as apiClient from '../lib/apiClient'
import { TaskListPage } from './TaskListPage'

describe('TaskListPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders empty state after loading with no tasks', async () => {
    vi.spyOn(apiClient, 'fetchTasks').mockResolvedValue([])

    render(
      <MemoryRouter>
        <TaskListPage />
      </MemoryRouter>,
    )

    expect(await screen.findByText('暂无任务')).toBeInTheDocument()
  })
})
