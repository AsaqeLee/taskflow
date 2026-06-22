import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as apiClient from '../lib/apiClient'
import { TaskNewPage } from './TaskNewPage'

describe('TaskNewPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('creates a task and navigates to detail page', async () => {
    const user = userEvent.setup()
    vi.spyOn(apiClient, 'createTask').mockResolvedValue({
      id: 'task_new_001',
      title: 'Launch docs sync',
      description: 'details',
      status: 'open',
      creator_id: 'u_owner',
      assignee_id: '',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })

    render(
      <MemoryRouter initialEntries={['/tasks/new']}>
        <Routes>
          <Route path="/tasks/new" element={<TaskNewPage />} />
          <Route path="/tasks/:id" element={<div>detail page</div>} />
        </Routes>
      </MemoryRouter>,
    )

    await user.type(screen.getByLabelText('标题'), 'Launch docs sync')
    await user.type(screen.getByLabelText('描述'), 'details')
    await user.click(screen.getByRole('button', { name: '创建' }))

    expect(apiClient.createTask).toHaveBeenCalledWith('Launch docs sync', 'details')
    expect(await screen.findByText('detail page')).toBeInTheDocument()
  })
})
