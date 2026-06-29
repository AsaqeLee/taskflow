import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as apiClient from '../lib/apiClient'
import { PasswordResetConfirmPage } from './PasswordResetConfirmPage'

describe('PasswordResetConfirmPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('prefills query parameters and confirms password reset', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(apiClient, 'confirmPasswordReset').mockResolvedValue({
      id: 'u_reset',
      name: 'Reset User',
      role: 'human',
      active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-02T00:00:00Z',
    })

    render(
      <MemoryRouter
        initialEntries={['/password-reset/confirm?id=u_reset&token=token-123']}
      >
        <PasswordResetConfirmPage />
      </MemoryRouter>,
    )

    expect(screen.getByDisplayValue('u_reset')).toBeInTheDocument()
    expect(screen.getByDisplayValue('token-123')).toBeInTheDocument()

    await user.type(screen.getByLabelText('新密码'), 'new-strong-pass')
    await user.click(screen.getByRole('button', { name: '确认重置' }))

    expect(confirmSpy).toHaveBeenCalledWith(
      'u_reset',
      'token-123',
      'new-strong-pass',
    )
    expect(await screen.findByRole('status')).toHaveTextContent(
      '密码已重置，请使用新密码重新登录。',
    )
  })
})
