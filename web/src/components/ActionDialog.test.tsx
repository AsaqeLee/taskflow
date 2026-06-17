import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ActionDialog } from './ActionDialog'

describe('ActionDialog', () => {
  it('requires confirmation for delete without content field', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn().mockResolvedValue(undefined)
    const onClose = vi.fn()

    render(
      <ActionDialog
        action="delete"
        needsContent={false}
        onClose={onClose}
        onConfirm={onConfirm}
      />,
    )

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('确认执行此操作？')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '确认' }))
    expect(onConfirm).toHaveBeenCalledWith('')
  })

  it('blocks submit when content is required but empty', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn()

    render(
      <ActionDialog
        action="submit"
        needsContent
        onClose={() => undefined}
        onConfirm={onConfirm}
      />,
    )

    await user.click(screen.getByRole('button', { name: '确认' }))
    expect(screen.getByText('请填写说明内容')).toBeInTheDocument()
    expect(onConfirm).not.toHaveBeenCalled()
  })
})