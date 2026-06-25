package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	domainaudit "github.com/AsaqeLee/taskflow/internal/domain/audit"
)

type MemoryAuditLogRepository struct {
	mu     sync.RWMutex
	logs   map[string]domainaudit.Log
	nextID int
}

func NewMemoryAuditLogRepository() *MemoryAuditLogRepository {
	return &MemoryAuditLogRepository{
		logs:   make(map[string]domainaudit.Log),
		nextID: 1,
	}
}

func (r *MemoryAuditLogRepository) Create(ctx context.Context, log domainaudit.Log) (domainaudit.Log, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domainaudit.Log{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if log.ID() == "" {
		log = log.AssignID(fmt.Sprintf("audit_%03d", r.nextID))
		r.nextID++
	}
	if log.CreatedAt().IsZero() {
		log = domainaudit.Restore(
			log.ID(),
			log.TaskID(),
			log.ActorID(),
			log.Action(),
			log.RequestID(),
			log.TraceID(),
			log.IdempotencyKey(),
			log.SourceIP(),
			log.UserAgent(),
			log.FromStatus(),
			log.ToStatus(),
			time.Now().UTC(),
		)
	}

	r.logs[log.ID()] = log
	return log, nil
}

func (r *MemoryAuditLogRepository) ListByTaskID(ctx context.Context, taskID string) ([]domainaudit.Log, error) {
	if err := errIfContextDone(ctx); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domainaudit.Log, 0)
	for _, log := range r.logs {
		if log.TaskID() == taskID {
			result = append(result, log)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt().Equal(result[j].CreatedAt()) {
			return result[i].ID() < result[j].ID()
		}
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})
	return result, nil
}

func (r *MemoryAuditLogRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	if err := errIfContextDone(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, log := range r.logs {
		if log.TaskID() == taskID {
			delete(r.logs, id)
		}
	}
	return nil
}
