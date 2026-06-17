import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { LoginPage } from './LoginPage'
import * as apiClient from '../lib/apiClient'

describe('LoginPage', () => {
  it('submits credentials and calls login API', async () => {
    const user = userEvent.setup()
    const loginSpy = vi.spyOn(apiClient, 'login').mockResolvedValue({
      user: {
        id: 'u_owner',
        name: 'Owner',
        role: 'owner',
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

    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    )

    await user.type(screen.getByLabelText('用户 ID'), 'u_owner')
    await user.type(screen.getByLabelText('密码'), 'secret-pass')
    await user.click(screen.getByRole('button', { name: '登录' }))

    expect(loginSpy).toHaveBeenCalledWith('u_owner', 'secret-pass')
    loginSpy.mockRestore()
  })
})