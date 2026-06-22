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
    expect(onConfirm).toHaveBeenCalledWith({ content: '', metadata: undefined })
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

  it('submits structured metadata when provided', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn().mockResolvedValue(undefined)

    render(
      <ActionDialog
        action="submit"
        needsContent
        metadataFields={[
          { key: 'summary', label: '执行摘要', placeholder: '摘要' },
          { key: 'failure_reason', label: '失败原因', placeholder: '失败原因' },
        ]}
        onClose={() => undefined}
        onConfirm={onConfirm}
      />,
    )

    await user.type(screen.getByPlaceholderText('填写说明（必填）'), 'agent result')
    await user.type(screen.getByPlaceholderText('摘要'), 'structured summary')
    await user.type(screen.getByPlaceholderText('失败原因'), 'none')
    await user.click(screen.getByRole('button', { name: '确认' }))

    expect(onConfirm).toHaveBeenCalledWith({
      content: 'agent result',
      metadata: {
        summary: 'structured summary',
        failure_reason: 'none',
      },
    })
  })
})
