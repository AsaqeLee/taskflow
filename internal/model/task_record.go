package model

import "time"

const TaskRecordTypeSubmit = "submit"
const TaskRecordTypeReject = "reject"
const TaskRecordTypeApprove = "approve"

type TaskRecord struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	AuthorID  string    `json:"author_id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
