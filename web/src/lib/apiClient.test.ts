import { afterEach, describe, expect, it, vi } from 'vitest'
import { clearSession, saveSession } from './auth'
import { createUser, fetchUsers } from './apiClient'

function signIn() {
  saveSession({
    user: {
      id: 'u_owner',
      name: 'Owner',
      role: 'owner',
      active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
    access_token: 'access-token',
    refresh_token: 'refresh-token',
    expires_in_seconds: 3600,
    refresh_expires_in_seconds: 86400,
    token_type: 'Bearer',
  })
}

describe('apiClient request headers', () => {
  afterEach(() => {
    clearSession()
    vi.restoreAllMocks()
  })

  it('adds idempotency key to mutating requests', async () => {
    signIn()

    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          user: {
            id: 'u_new',
            name: 'New User',
            role: 'human',
            active: true,
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
          },
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        },
      ),
    )

    await createUser('u_new', 'New User', 'human', 'Strong-pass-123')

    const [, init] = fetchSpy.mock.calls[0]
    const headers = new Headers(init?.headers)

    expect(headers.get('Authorization')).toBe('Bearer access-token')
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(headers.get('Idempotency-Key')).toBeTruthy()
  })

  it('does not add idempotency key to read requests', async () => {
    signIn()

    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          users: [],
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        },
      ),
    )

    await fetchUsers()

    const [, init] = fetchSpy.mock.calls[0]
    const headers = new Headers(init?.headers)

    expect(headers.get('Authorization')).toBe('Bearer access-token')
    expect(headers.get('Idempotency-Key')).toBeNull()
  })
})
