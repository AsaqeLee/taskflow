import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as apiClient from '../lib/apiClient'
import { PasswordResetRequestPage } from './PasswordResetRequestPage'

describe('PasswordResetRequestPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('submits a password reset request and shows accepted state', async () => {
    const user = userEvent.setup()
    const requestSpy = vi
      .spyOn(apiClient, 'requestPasswordReset')
      .mockResolvedValue({ status: 'accepted' })

    render(
      <MemoryRouter>
        <PasswordResetRequestPage />
      </MemoryRouter>,
    )

    await user.type(screen.getByLabelText('用户 ID'), 'u_reset')
    await user.click(screen.getByRole('button', { name: '发送重置请求' }))

    expect(requestSpy).toHaveBeenCalledWith('u_reset')
    expect(await screen.findByRole('status')).toHaveTextContent(
      '已接受密码重置请求',
    )
  })

  it('shows returned dev reset token and confirm link', async () => {
    const user = userEvent.setup()

    vi.spyOn(apiClient, 'requestPasswordReset').mockResolvedValue({
      status: 'accepted',
      reset_token: 'dev-reset-token',
    })

    render(
      <MemoryRouter>
        <PasswordResetRequestPage />
      </MemoryRouter>,
    )

    await user.type(screen.getByLabelText('用户 ID'), 'u_dev')
    await user.click(screen.getByRole('button', { name: '发送重置请求' }))

    expect(await screen.findByText('dev-reset-token')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '前往确认重置' })).toHaveAttribute(
      'href',
      '/password-reset/confirm?id=u_dev&token=dev-reset-token',
    )
  })
})
