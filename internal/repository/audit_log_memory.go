package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
)

type MemoryAuditLogRepository struct {
	mu     sync.RWMutex
	logs   map[string]model.AuditLog
	nextID int
}

func NewMemoryAuditLogRepository() *MemoryAuditLogRepository {
	return &MemoryAuditLogRepository{
		logs:   make(map[string]model.AuditLog),
		nextID: 1,
	}
}

func (r *MemoryAuditLogRepository) Create(ctx context.Context, log model.AuditLog) (model.AuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if log.ID == "" {
		log.ID = fmt.Sprintf("audit_%03d", r.nextID)
		r.nextID++
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	r.logs[log.ID] = log
	return log, nil
}

func (r *MemoryAuditLogRepository) ListByTaskID(ctx context.Context, taskID string) ([]model.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.AuditLog, 0)
	for _, log := range r.logs {
		if log.TaskID == taskID {
			result = append(result, log)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

func (r *MemoryAuditLogRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, log := range r.logs {
		if log.TaskID == taskID {
			delete(r.logs, id)
		}
	}
	return nil
}
