import { clearSession } from './auth'

type SessionExpiredHandler = () => void

let onSessionExpired: SessionExpiredHandler | null = null

export function setSessionExpiredHandler(handler: SessionExpiredHandler): void {
  onSessionExpired = handler
}

export function notifySessionExpired(): void {
  clearSession()
  onSessionExpired?.()
}