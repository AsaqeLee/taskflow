export type TaskStatus =
  | 'open'
  | 'assigned'
  | 'in_progress'
  | 'submitted'
  | 'approved'
  | 'completed'
  | 'cancelled'
  | 'deleted'

export type UserRole = 'owner' | 'human' | 'agent'

export interface User {
  id: string
  name: string
  role: UserRole
  active: boolean
  disabled_at?: string
  disabled_by?: string
  created_at: string
  updated_at: string
}

export interface APIKey {
  id: string
  user_id: string
  name: string
  key_prefix: string
  created_at: string
  expires_at?: string
  last_used_at?: string
  revoked_at?: string
}

export interface Task {
  id: string
  title: string
  description: string
  status: TaskStatus
  available_actions?: TaskAction[]
  creator_id: string
  assignee_id: string
  created_at: string
  updated_at: string
  deleted_at?: string
  deleted_by?: string
}

export interface TaskListQuery {
  q?: string
  status?: TaskStatus
  page?: number
  page_size?: number
}

export interface TaskListResponse {
  tasks: Task[]
  total: number
  page: number
  page_size: number
}

export interface TaskRecord {
  id: string
  task_id: string
  author_id: string
  type: string
  content: string
  metadata?: Record<string, string>
  created_at: string
}

export interface TaskRecordInput {
  content: string
  metadata?: Record<string, string>
}

export interface TaskRecordMetadataField {
  key: string
  label: string
  placeholder: string
}

export interface AuditLog {
  id: string
  task_id: string
  actor_id: string
  action: string
  request_id?: string
  from_status?: string
  to_status?: string
  created_at: string
}

export interface SessionResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_in_seconds: number
  refresh_expires_in_seconds: number
  token_type: string
}

export interface PasswordResetRequestResponse {
  status: string
  reset_token?: string
}

export interface ApiError {
  error: {
    code: string
    message: string
    request_id?: string
  }
}

export type TaskAction =
  | 'assign'
  | 'start'
  | 'submit'
  | 'reject'
  | 'approve'
  | 'close'
  | 'cancel'
  | 'reactivate'
  | 'delete'
