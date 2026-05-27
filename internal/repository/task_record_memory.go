package repository

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
)

var ErrTaskRecordNotFound = errors.New("task record not found")

type TaskRecordRepository interface {
	Create(record model.TaskRecord) (model.TaskRecord, error)
	ListByTaskID(taskID string) ([]model.TaskRecord, error)
	DeleteByTaskID(taskID string) error
}

type MemoryTaskRecordRepository struct {
	mu      sync.RWMutex
	records map[string]model.TaskRecord
	nextID  int
}

func NewMemoryTaskRecordRepository() *MemoryTaskRecordRepository {
	return &MemoryTaskRecordRepository{
		records: make(map[string]model.TaskRecord),
		nextID:  1,
	}
}

func (r *MemoryTaskRecordRepository) Create(record model.TaskRecord) (model.TaskRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if record.ID == "" {
		record.ID = fmt.Sprintf("record_%03d", r.nextID)
		r.nextID++
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	r.records[record.ID] = record
	return record, nil
}

func (r *MemoryTaskRecordRepository) ListByTaskID(taskID string) ([]model.TaskRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.TaskRecord, 0)
	for _, record := range r.records {
		if record.TaskID == taskID {
			result = append(result, record)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

func (r *MemoryTaskRecordRepository) DeleteByTaskID(taskID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, record := range r.records {
		if record.TaskID == taskID {
			delete(r.records, id)
		}
	}
	return nil
}
