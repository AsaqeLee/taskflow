import {
  getAccessToken,
  getRefreshToken,
  saveSession,
} from './auth'
import { notifySessionExpired } from './session'
import type {
  ApiError,
  AuditLog,
  SessionResponse,
  Task,
  TaskRecordInput,
  TaskRecord,
  User,
} from '../types/api'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api'

export class ApiClientError extends Error {
  code: string
  status: number
  requestId?: string

  constructor(status: number, code: string, message: string, requestId?: string) {
    super(message)
    this.name = 'ApiClientError'
    this.status = status
    this.code = code
    this.requestId = requestId
  }
}

let refreshPromise: Promise<boolean> | null = null

async function parseError(response: Response): Promise<ApiClientError> {
  let code = 'request_failed'
  let message = response.statusText || 'Request failed'
  let requestId: string | undefined

  try {
    const body = (await response.json()) as ApiError
    if (body.error) {
      code = body.error.code
      message = body.error.message
      requestId = body.error.request_id
    }
  } catch {
    // ignore parse errors
  }

  return new ApiClientError(response.status, code, message, requestId)
}

async function refreshTokens(): Promise<boolean> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return false

  const response = await fetch(`${API_BASE}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })

  if (!response.ok) {
    notifySessionExpired()
    return false
  }

  const session = (await response.json()) as SessionResponse
  saveSession(session)
  return true
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  retry = true,
): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type') && init.body) {
    headers.set('Content-Type', 'application/json')
  }

  const token = getAccessToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(`${API_BASE}${path}`, { ...init, headers })

  if (response.status === 401 && retry) {
    if (!refreshPromise) {
      refreshPromise = refreshTokens().finally(() => {
        refreshPromise = null
      })
    }
    const refreshed = await refreshPromise
    if (refreshed) {
      return request<T>(path, init, false)
    }
    notifySessionExpired()
    throw await parseError(response)
  }

  if (!response.ok) {
    throw await parseError(response)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

export async function login(id: string, password: string): Promise<SessionResponse> {
  const response = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id, password }),
  })

  if (!response.ok) {
    throw await parseError(response)
  }

  const session = (await response.json()) as SessionResponse
  saveSession(session)
  return session
}

export async function fetchMe(): Promise<User> {
  const body = await request<{ user: User }>('/me')
  return body.user
}

export async function fetchUsers(activeOnly = true): Promise<User[]> {
  const query = activeOnly ? '?active=true' : ''
  const body = await request<{ users: User[] }>(`/users${query}`)
  return body.users
}

export async function fetchTasks(): Promise<Task[]> {
  const body = await request<{ tasks: Task[] }>('/tasks')
  return body.tasks
}

export async function fetchTask(id: string): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}`)
  return body.task
}

export async function createTask(title: string, description: string): Promise<Task> {
  const body = await request<{ task: Task }>('/tasks', {
    method: 'POST',
    body: JSON.stringify({ title, description }),
  })
  return body.task
}

export async function updateTask(
  id: string,
  title: string,
  description: string,
): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ title, description }),
  })
  return body.task
}

export async function assignTask(id: string, assigneeId: string): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}/assign`, {
    method: 'POST',
    body: JSON.stringify({ assignee_id: assigneeId }),
  })
  return body.task
}

export async function startTask(id: string): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}/start`, { method: 'POST' })
  return body.task
}

export async function submitTask(id: string, input: TaskRecordInput): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}/submit`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
  return body.task
}

export async function rejectTask(id: string, input: TaskRecordInput): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}/reject`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
  return body.task
}

export async function approveTask(id: string, input: TaskRecordInput): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}/approve`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
  return body.task
}

export async function closeTask(id: string): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}/close`, { method: 'POST' })
  return body.task
}

export async function cancelTask(id: string, content: string): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}/cancel`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
  return body.task
}

export async function reactivateTask(id: string, content: string): Promise<Task> {
  const body = await request<{ task: Task }>(`/tasks/${id}/reactivate`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
  return body.task
}

export async function deleteTask(id: string): Promise<void> {
  await request<void>(`/tasks/${id}`, { method: 'DELETE' })
}

export async function fetchTaskRecords(id: string): Promise<TaskRecord[]> {
  const body = await request<{ records: TaskRecord[] }>(`/tasks/${id}/records`)
  return body.records
}

export async function fetchTaskAuditLogs(id: string): Promise<AuditLog[]> {
  const body = await request<{ audit_logs: AuditLog[] }>(`/tasks/${id}/audit_logs`)
  return body.audit_logs
}
