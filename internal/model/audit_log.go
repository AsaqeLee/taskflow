package model

import "time"

const (
	AuditActionCreated   = "task_created"
	AuditActionAssigned  = "task_assigned"
	AuditActionStarted   = "task_started"
	AuditActionSubmitted = "task_submitted"
	AuditActionApproved  = "task_approved"
	AuditActionRejected  = "task_rejected"
	AuditActionClosed    = "task_closed"
	AuditActionCancelled = "task_cancelled"
	AuditActionReopened  = "task_reopened"
	AuditActionDeleted   = "task_deleted"
)

type AuditLog struct {
	ID             string    `json:"id"`
	TaskID         string    `json:"task_id"`
	ActorID        string    `json:"actor_id"`
	Action         string    `json:"action"`
	RequestID      string    `json:"request_id,omitempty"`
	TraceID        string    `json:"trace_id,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	SourceIP       string    `json:"source_ip,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	FromStatus     string    `json:"from_status,omitempty"`
	ToStatus       string    `json:"to_status,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
