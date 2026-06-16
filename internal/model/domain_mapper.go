package model

import (
	domainaudit "github.com/AsaqeLee/taskflow/internal/domain/audit"
	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
)

func TaskFromDomain(t domaintask.Task) Task {
	return Task{
		ID:          t.ID(),
		Title:       t.Title(),
		Description: t.Description(),
		Status:      t.Status().String(),
		CreatorID:   t.CreatorID(),
		AssigneeID:  t.AssigneeID(),
		CreatedAt:   t.CreatedAt(),
		UpdatedAt:   t.UpdatedAt(),
		DeletedAt:   t.DeletedAt(),
		DeletedBy:   t.DeletedBy(),
	}
}

func TasksFromDomain(items []domaintask.Task) []Task {
	result := make([]Task, len(items))
	for i, item := range items {
		result[i] = TaskFromDomain(item)
	}
	return result
}

func TaskRecordFromDomain(r domainrecord.Record) TaskRecord {
	return TaskRecord{
		ID:        r.ID(),
		TaskID:    r.TaskID(),
		AuthorID:  r.AuthorID(),
		Type:      r.Type().String(),
		Content:   r.Content(),
		CreatedAt: r.CreatedAt(),
	}
}

func TaskRecordsFromDomain(items []domainrecord.Record) []TaskRecord {
	result := make([]TaskRecord, len(items))
	for i, item := range items {
		result[i] = TaskRecordFromDomain(item)
	}
	return result
}

func AuditLogFromDomain(l domainaudit.Log) AuditLog {
	return AuditLog{
		ID:             l.ID(),
		TaskID:         l.TaskID(),
		ActorID:        l.ActorID(),
		Action:         l.Action().String(),
		RequestID:      l.RequestID(),
		TraceID:        l.TraceID(),
		IdempotencyKey: l.IdempotencyKey(),
		SourceIP:       l.SourceIP(),
		UserAgent:      l.UserAgent(),
		FromStatus:     l.FromStatus(),
		ToStatus:       l.ToStatus(),
		CreatedAt:      l.CreatedAt(),
	}
}

func AuditLogsFromDomain(items []domainaudit.Log) []AuditLog {
	result := make([]AuditLog, len(items))
	for i, item := range items {
		result[i] = AuditLogFromDomain(item)
	}
	return result
}
