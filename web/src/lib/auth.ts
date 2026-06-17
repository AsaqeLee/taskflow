import type { SessionResponse, User } from '../types/api'

const ACCESS_KEY = 'taskflow.access_token'
const REFRESH_KEY = 'taskflow.refresh_token'
const USER_KEY = 'taskflow.user'

export function getAccessToken(): string | null {
  return sessionStorage.getItem(ACCESS_KEY)
}

export function getRefreshToken(): string | null {
  return sessionStorage.getItem(REFRESH_KEY)
}

export function getStoredUser(): User | null {
  const raw = sessionStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    sessionStorage.removeItem(USER_KEY)
    return null
  }
}

export function saveSession(session: SessionResponse): void {
  sessionStorage.setItem(ACCESS_KEY, session.access_token)
  sessionStorage.setItem(REFRESH_KEY, session.refresh_token)
  sessionStorage.setItem(USER_KEY, JSON.stringify(session.user))
}

export function clearSession(): void {
  sessionStorage.removeItem(ACCESS_KEY)
  sessionStorage.removeItem(REFRESH_KEY)
  sessionStorage.removeItem(USER_KEY)
}

export function isLoggedIn(): boolean {
  return Boolean(getAccessToken() && getRefreshToken() && getStoredUser())
}
