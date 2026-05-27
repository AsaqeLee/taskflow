package model

import "time"

const (
	AuditActionCreated    = "task_created"
	AuditActionAssigned   = "task_assigned"
	AuditActionStarted    = "task_started"
	AuditActionSubmitted  = "task_submitted"
	AuditActionApproved   = "task_approved"
	AuditActionRejected   = "task_rejected"
	AuditActionClosed     = "task_closed"
	AuditActionCancelled  = "task_cancelled"
	AuditActionReopened   = "task_reopened"
)

type AuditLog struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	ActorID   string    `json:"actor_id"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}
