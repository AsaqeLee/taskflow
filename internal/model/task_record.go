package model

import "time"

const TaskRecordTypeSubmit = "submit"
const TaskRecordTypeReject = "reject"
const TaskRecordTypeApprove = "approve"
const TaskRecordTypeCancel = "cancel"
const TaskRecordTypeReactivate = "reactivate"

type TaskRecord struct {
	ID        string            `json:"id"`
	TaskID    string            `json:"task_id"`
	AuthorID  string            `json:"author_id"`
	Type      string            `json:"type"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}
