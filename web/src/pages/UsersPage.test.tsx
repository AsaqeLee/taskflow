import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as apiClient from '../lib/apiClient'
import { clearSession, saveSession } from '../lib/auth'
import { UsersPage } from './UsersPage'

function signInAs(role: 'owner' | 'human' | 'agent') {
  saveSession({
    user: {
      id: role === 'owner' ? 'u_owner' : 'u_human',
      name: role === 'owner' ? 'Owner' : 'Human',
      role,
      active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
    access_token: 'access',
    refresh_token: 'refresh',
    expires_in_seconds: 3600,
    refresh_expires_in_seconds: 86400,
    token_type: 'Bearer',
  })
}

describe('UsersPage', () => {
  afterEach(() => {
    clearSession()
    vi.restoreAllMocks()
  })

  it('blocks non-owner users', async () => {
    signInAs('human')
    vi.spyOn(apiClient, 'fetchUsers').mockResolvedValue([])

    render(<UsersPage />)

    expect(await screen.findByRole('alert')).toHaveTextContent('仅 owner 可访问用户管理')
  })

  it('creates user through owner form', async () => {
    signInAs('owner')
    const user = userEvent.setup()

    vi.spyOn(apiClient, 'fetchUsers').mockResolvedValue([])
    const createSpy = vi.spyOn(apiClient, 'createUser').mockResolvedValue({
      id: 'u_new',
      name: 'New User',
      role: 'human',
      active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })

    render(<UsersPage />)

    await user.type(screen.getByLabelText('用户 ID'), 'u_new')
    await user.type(screen.getByLabelText('姓名'), 'New User')
    await user.type(screen.getByLabelText('初始密码'), 'Strong-pass-123')
    await user.click(screen.getByRole('button', { name: '创建用户' }))

    expect(createSpy).toHaveBeenCalledWith('u_new', 'New User', 'human', 'Strong-pass-123')
    expect(await screen.findByRole('status')).toHaveTextContent('用户已创建')
  })

  it('disables another user', async () => {
    signInAs('owner')
    const user = userEvent.setup()

    vi.spyOn(apiClient, 'fetchUsers').mockResolvedValue([
      {
        id: 'u_owner',
        name: 'Owner',
        role: 'owner',
        active: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
      {
        id: 'u_worker',
        name: 'Worker',
        role: 'human',
        active: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-02T00:00:00Z',
      },
    ])
    vi.spyOn(apiClient, 'fetchUserAPIKeys').mockResolvedValue([])
    const disableSpy = vi.spyOn(apiClient, 'disableUser').mockResolvedValue({
      id: 'u_worker',
      name: 'Worker',
      role: 'human',
      active: false,
      disabled_by: 'u_owner',
      disabled_at: '2026-01-03T00:00:00Z',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-03T00:00:00Z',
    })

    render(<UsersPage />)

    await user.click(await screen.findByRole('button', { name: '禁用账号' }))

    expect(disableSpy).toHaveBeenCalledWith('u_worker')
    expect(await screen.findByRole('status')).toHaveTextContent('已禁用用户 u_worker')
  })

  it('creates and displays api key for selected user', async () => {
    signInAs('owner')
    const user = userEvent.setup()

    vi.spyOn(apiClient, 'fetchUsers').mockResolvedValue([
      {
        id: 'u_agent',
        name: 'Hermes Agent',
        role: 'agent',
        active: true,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    ])
    vi.spyOn(apiClient, 'fetchUserAPIKeys').mockResolvedValue([])
    const createAPIKeySpy = vi.spyOn(apiClient, 'createUserAPIKey').mockResolvedValue({
      apiKey: {
        id: 'ak_001',
        user_id: 'u_agent',
        name: 'Hermes Prod',
        key_prefix: 'tfk_secret_1',
        created_at: '2026-01-01T00:00:00Z',
      },
      key: 'tfk_secret_123',
    })

    render(<UsersPage />)

    await user.type(await screen.findByLabelText('Key 名称'), 'Hermes Prod')
    await user.click(screen.getByRole('button', { name: '创建 API key' }))

    expect(createAPIKeySpy).toHaveBeenCalledWith('u_agent', 'Hermes Prod', undefined)
    expect(await screen.findByRole('status')).toHaveTextContent('已为 u_agent 创建 API key')
    expect(screen.getByText('新 API key（仅显示一次）')).toBeInTheDocument()
    expect(screen.getByText('tfk_secret_123')).toBeInTheDocument()
  })
})
