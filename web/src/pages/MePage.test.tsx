import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as apiClient from '../lib/apiClient'
import { MePage } from './MePage'

describe('MePage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows an error alert when loading profile fails', async () => {
    vi.spyOn(apiClient, 'fetchMe').mockRejectedValue(new Error('load failed'))

    render(<MePage />)

    expect(await screen.findByRole('alert')).toHaveTextContent('load failed')
  })
})
