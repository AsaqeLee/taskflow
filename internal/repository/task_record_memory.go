package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
)

type MemoryTaskRecordRepository struct {
	mu      sync.RWMutex
	records map[string]domainrecord.Record
	nextID  int
}

func NewMemoryTaskRecordRepository() *MemoryTaskRecordRepository {
	return &MemoryTaskRecordRepository{
		records: make(map[string]domainrecord.Record),
		nextID:  1,
	}
}

func (r *MemoryTaskRecordRepository) Create(ctx context.Context, record domainrecord.Record) (domainrecord.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if record.ID() == "" {
		record = record.AssignID(fmt.Sprintf("record_%03d", r.nextID))
		r.nextID++
	}
	if record.CreatedAt().IsZero() {
		record = domainrecord.Restore(record.ID(), record.TaskID(), record.AuthorID(), record.Type(), record.Content(), time.Now().UTC())
	}

	r.records[record.ID()] = record
	return record, nil
}

func (r *MemoryTaskRecordRepository) ListByTaskID(ctx context.Context, taskID string) ([]domainrecord.Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domainrecord.Record, 0)
	for _, record := range r.records {
		if record.TaskID() == taskID {
			result = append(result, record)
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

func (r *MemoryTaskRecordRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, record := range r.records {
		if record.TaskID() == taskID {
			delete(r.records, id)
		}
	}
	return nil
}
